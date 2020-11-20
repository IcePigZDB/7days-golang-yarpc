package registry

import (
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// YaRegistry is a simple register center, provide following functions.
// add a server and receive heartbeat to keep it alive.
// returns all alive servers and delete dead servers sync simultaneously.
type YaRegistry struct {
	timeout time.Duration
	mu      sync.Mutex // protect following
	servers map[string]*ServerItem
}

// ServerItem struct
type ServerItem struct {
	Addr  string
	start time.Time
}

const (
	defaultPath    = "/_yarpc_/registry"
	defaultTimeout = time.Minute * 5
)

// NewRegistry create a registry instance with timeout setting
func NewRegistry(timeout time.Duration) *YaRegistry {
	return &YaRegistry{
		servers: make(map[string]*ServerItem),
		timeout: timeout,
	}
}

// DefaultYaRegistey is the defaultone
var DefaultYaRegistey = NewRegistry(defaultTimeout)

// put server or update server time
func (r *YaRegistry) putServer(addr string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.servers[addr]
	if s == nil {
		r.servers[addr] = &ServerItem{Addr: addr, start: time.Now()}
	} else {
		s.start = time.Now() // if exists, update start time to keep alive
	}
}

// check aliveServers
func (r *YaRegistry) aliveServers() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var alive []string
	for addr, s := range r.servers {
		// timeout == 0 means no limit
		if r.timeout == 0 || s.start.Add(r.timeout).After(time.Now()) {
			alive = append(alive, addr)
		} else {
			delete(r.servers, addr)
		}
	}
	sort.Strings(alive)
	return alive
}

// Runs at /_yarpc_/registry
// Get：返回所有可用的服务列表，通过自定义字段 X-Geerpc-Servers 承载。
// Post：添加服务实例或发送心跳，通过自定义字段 X-Geerpc-Server 承载。
func (r *YaRegistry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		// keep it simple, server is in req.Header
		w.Header().Set("X-Yarpc-Servers", strings.Join(r.aliveServers(), ","))
	case "POST":
		// keep it simple, server is in req.Header
		addr := req.Header.Get("X-Yarpc-Server")
		if addr == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		r.putServer(addr)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// HandleHTTP registers an HTTP handler for YaRegistry messages on registryPath
func (r *YaRegistry) HandleHTTP(registryPath string) {
	http.Handle(registryPath, r)
	log.Println("rpc registry path:", registryPath)
}

// HandleHTTP exported http
func HandleHTTP() {
	DefaultYaRegistey.HandleHTTP(defaultPath)
}

// Heartbeat send a heartbeat message every once in a while
// it's a helper function for a server to register or send heartbeat
// 便于服务启动时定时向注册中心发送心跳，默认周期比注册中心设置的过期时间少1 min。
func Heartbeat(registry, addr string, duration time.Duration) {
	if duration == 0 {
		// make sure there is enough time to send heart beat
		// before it's removed from registry
		duration = defaultTimeout - time.Duration(1)*time.Minute
	}
	var err error
	err = sendHeartbeat(registry, addr)
	go func() {
		// every duration triker.C will have some to be <-t.C
		t := time.NewTicker(duration)
		for err == nil {
			<-t.C
			err = sendHeartbeat(registry, addr)
		}
	}()
}

func sendHeartbeat(registry, addr string) error {
	log.Println(addr, "send heart beat to registry", registry)
	httpClient := &http.Client{}
	req, _ := http.NewRequest("POST", registry, nil)
	req.Header.Set("X-Yarpc-Server", addr)
	// Do send and return a response
	if _, err := httpClient.Do(req); err != nil {
		log.Println("rpc server: heart beat err:", err)
		return err
	}
	return nil
}
