package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
	"yarpc"
	"yarpc/registry"
	"yarpc/xclient"
)

type Foo int

type Args struct{ Num1, Num2 int }

type ArgsStr struct{ Str1 string }

func (f Foo) Sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

func (f Foo) Uppercase(args ArgsStr, reply *string) error {
	*reply = strings.ToUpper(args.Str1)
	return nil
}

// func (f Foo) Sleep(args Args, reply *int) error {
// 	time.Sleep(time.Second * time.Duration(args.Num1))
// 	*reply = args.Num1 + args.Num2
// 	return nil
// }

func startRegistry(wg *sync.WaitGroup) {
	l, _ := net.Listen("tcp", ":9999")
	// handle http server on
	// http://localhost:9999/_yarpc_/registry
	registry.HandleHTTP()
	wg.Done()
	_ = http.Serve(l, nil)
}

func startServer(serverID int, registryAddr string, wg *sync.WaitGroup) {
	var foo Foo
	l, _ := net.Listen("tcp", ":0")                               // listen tcp port
	server := yarpc.NewServer(serverID)                           // create new server
	_ = server.Register(&foo)                                     // register foo service
	registry.Heartbeat(registryAddr, "tcp@"+l.Addr().String(), 0) // heart beat with server
	wg.Done()
	server.Accept(l) // server listen to tpc conn
}

// func foo(xc *xclient.XClient, ctx context.Context, typ, serviceMethod string, args interface{}) {
// 	var reply int
// 	var err error
// 	switch typ {
// 	case "call":
// 		err = xc.Call(ctx, serviceMethod, args, &reply)
// 	case "broadcast":
// 		err = xc.Broadcast(ctx, serviceMethod, args, &reply)
// 	}
// 	fmt.Println(reflect.TypeOf(args))
// 	if err != nil {
// 		log.Printf("%s %s error: %v", typ, serviceMethod, err)
// 	} else {
// 		// log.Printf("%s %s success: %d + %d = %d", typ, serviceMethod, args.Num1, args.Num2, reply)
// 	}
// }

func call(registry string, callNum int, serverNum int) {
	d := xclient.NewYaRegistryDiscovery(registry, 0)
	// randomSelect
	xc := xclient.NewXClient(d, xclient.RandomSelect, nil)
	// round roubin
	// xc := xclient.NewXClient(d, xclient.RoundRobinSelect, nil)
	defer func() { _ = xc.Close() }()
	// send request & receive response
	var wg sync.WaitGroup
	var serverChoosedSlice = make([]int, serverNum, serverNum)
	for i := 0; i < callNum; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			var reply int
			args := &Args{Num1: i, Num2: i * i}
			serverID, err := xc.Call(context.Background(), "Foo.Sum", &args, &reply)
			if err != nil {
				log.Fatal("call Foo Sum error:", err)
			}
			log.Printf("\n%s processed by server%d success:\n %d + %d = %d", "Foo.Sum", serverID, args.Num1, args.Num2, reply)
			serverChoosedSlice[serverID]++
		}(i)
	}

	// for i := 0; i < callNum; i++ {
	// 	wg.Add(1)
	// 	go func(i int) {
	// 		defer wg.Done()
	// 		argsStr := &ArgsStr{Str1: fmt.Sprintf("number %d rpc called from room 319,caller 202021080301&202022080315", i)}
	// 		var replyStr string
	// 		serverID, err := xc.Call(context.Background(), "Foo.Uppercase", &argsStr, &replyStr)
	// 		if err != nil {
	// 			log.Fatal("call Foo Uppercase error:", err)
	// 		}
	// 		log.Printf("\n%s processed by server%d success:\nbefore:%s \nafter:%s", "Foo.Uppstram", serverID, argsStr.Str1, replyStr)
	// 		serverChoosedSlice[serverID]++
	// 	}(i)
	// }

	wg.Wait()
	for i := 0; i < serverNum; i++ {
		fmt.Printf("%d ", serverChoosedSlice[i])
	}
}

// func broadcast(registry string) {
// 	d := xclient.NewYaRegistryDiscovery(registry, 0)
// 	xc := xclient.NewXClient(d, xclient.RandomSelect, nil)
// 	defer func() { _ = xc.Close() }()
// 	var wg sync.WaitGroup
// 	for i := 0; i < 5; i++ {
// 		wg.Add(1)
// 		go func(i int) {
// 			defer wg.Done()
// 			foo(xc, context.Background(), "broadcast", "Foo.Sum", &Args{Num1: i, Num2: i * i})
// 			// expect 2 - 5 timeout
// 			ctx, _ := context.WithTimeout(context.Background(), time.Second*2)
// 			foo(xc, ctx, "broadcast", "Foo.Sleep", &Args{Num1: i, Num2: i * i})
// 		}(i)
// 	}
// 	wg.Wait()
// }

func main() {
	log.SetFlags(0)
	registryAddr := "http://localhost:9999/_yarpc_/registry"
	var wg sync.WaitGroup
	wg.Add(1)
	go startRegistry(&wg)
	wg.Wait()

	time.Sleep(time.Second)
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go startServer(i, registryAddr, &wg)
	}
	wg.Wait()

	time.Sleep(time.Second)
	call(registryAddr, 99, 3)
	// broadcast(registryAddr)
}
