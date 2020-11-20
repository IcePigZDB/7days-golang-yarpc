package xclient

import (
	"context"
	"io"
	"reflect"
	"sync"
	. "yarpc"
)

// though it tell us not to use dot but actual it is in the same package,i think it is ok
// import   "lib/math"         math.Sin
// import M "lib/math"         M.Sin
// import . "lib/math"         Sin

// XClient is a client support load balance.
type XClient struct {
	d       Discovery
	mode    SelectMode
	opt     *Option
	mu      sync.Mutex // protect following
	clients map[string]*Client
}

var _ io.Closer = (*XClient)(nil)

// NewXClient return a XClient
func NewXClient(d Discovery, mode SelectMode, opt *Option) *XClient {
	return &XClient{d: d, mode: mode, opt: opt, clients: make(map[string]*Client)}
}

// XClient close
func (xc *XClient) Close() error {
	xc.mu.Lock()
	defer xc.mu.Unlock()
	for key, client := range xc.clients {
		// I have no idea how to deal with error, just ignore it.
		_ = client.Close()
		delete(xc.clients, key)
	}
	return nil
}

// 复用Client的能力
// 1)检查 xc.clients 是否有缓存的 Client，如果有，检查是否是可用状态，
// 如果是则返回缓存的 Client，如果不可用，则从缓存中删除。
// 2)如果步骤 1) 没有返回缓存的 Client，则说明需要创建新的 Client，缓存并返回。
func (xc *XClient) dial(rpcAddr string) (*Client, error) {
	xc.mu.Lock()
	defer xc.mu.Unlock()
	client, ok := xc.clients[rpcAddr]
	// cache client but is not availabel
	if ok && !client.IsAvailable() {
		_ = client.Close()
		delete(xc.clients, rpcAddr)
		client = nil
	}
	// no cache client
	if client == nil {
		var err error
		client, err = XDial(rpcAddr, xc.opt)
		if err != nil {
			return nil, err
		}
		xc.clients[rpcAddr] = client
	}
	// cache client & available
	return client, nil
}

func (xc *XClient) call(rpcAddr string, ctx context.Context, serviceMethod string, args, reply interface{}) error {
	client, err := xc.dial(rpcAddr)
	if err != nil {
		return err
	}
	return client.Call(ctx, serviceMethod, args, reply)
}

// Call invokes the named function, waits for it to complete,
// and returns its error status.
// xc will choose a proper server.
func (xc *XClient) Call(ctx context.Context, serviceMethod string, args, reply interface{}) error {
	rpcAddr, err := xc.d.Get(xc.mode)
	if err != nil {
		return err
	}
	return xc.call(rpcAddr, ctx, serviceMethod, args, reply)
}

// Broadcast 将请求广播到所有的服务实例，如果任意一个实例发生错误，
// 则返回其中一个错误；如果调用成功，则返回其中一个的结果。有以下几点需要注意：
// 为了提升性能，请求是并发的。
// 并发情况下需要使用互斥锁保证 error 和 reply 能被正确赋值。
// 借助 context.WithCancel 确保有错误发生时，快速失败ast invokes the named function for every server registered in discovery.
func (xc *XClient) Broadcast(ctx context.Context, serviceMethod string, args, reply interface{}) error {
	// get all server instance
	servers, err := xc.d.GetAll()
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	var mu sync.Mutex // protect e and replyDone
	var e error
	// := is prior to ==
	// if reply is nil, don't need to set value
	replyDone := reply == nil
	// check all go routine use this context
	ctx, cancel := context.WithCancel(ctx)
	for _, rpcAddr := range servers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// clonedReply to reply to multi request
			var clonedReply interface{}
			if reply != nil {
				clonedReply = reflect.New(reflect.ValueOf(reply).Elem().Type()).Interface()
			}
			// it is ok rpcAddr in the for range use only one address,but value will change
			err := xc.call(rpcAddr, ctx, serviceMethod, args, clonedReply)
			mu.Lock()
			if err != nil && e == nil {
				e = err
				cancel() // if any call failed, cancel unfinished calls
			}
			if err == nil && !replyDone {
				// reply set to clonedReply
				reflect.ValueOf(reply).Elem().Set(reflect.ValueOf(clonedReply).Elem())
				replyDone = true
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
	return e
}
