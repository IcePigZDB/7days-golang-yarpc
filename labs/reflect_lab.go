package main

import (
	"fmt"
	"log"
	"reflect"
	"strings"
	"sync"
)

// a simple reflect lab to unstand the usage of reflect
func main() {
	var wg sync.WaitGroup
	typ := reflect.TypeOf(&wg)
	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)
		argv := make([]string, 0, method.Type.NumIn())
		returns := make([]string, 0, method.Type.NumOut())
		// j 从 1 开始，第 0 个入参是 wg 自己。
		for j := 1; j < method.Type.NumIn(); j++ {
			argv = append(argv, method.Type.In(j).Name())
		}
		for j := 0; j < method.Type.NumOut(); j++ {
			returns = append(returns, method.Type.Out(j).Name())
		}
		log.Printf("func (w *%s) %s(%s) %s",
			typ.Elem().Name(),
			method.Name,
			strings.Join(argv, ","),
			strings.Join(returns, ","))
	}
	// reflect.ValueOf 的逆操作是reflect.Value.Interface 方法.
	// 它返回一个interface{} 类型，装载着与reflect.Value 相同的具体值
	v := reflect.ValueOf(3) // a reflect.Value
	x := v.Interface()      // an interface{}
	i := x.(int)            // an int interface to int
	fmt.Printf("%d\n", i)   // "3"
	fmt.Printf("type of i is %s \n", reflect.TypeOf(i))
	// reflect.Value 和interface{} 都能装载任意的值. 所不同的是,
	// 一个空的接口隐藏了值内部的表示方式和所有方法,
	// 因此只有我们知道具体的动态类型才能使用类型断言来访问内部的值(就像上面那样),
	// 内部值我们没法访问.
	// 相比之下, 一个Value 则有很多方法来检查其内容, 无论它的具体类型是什么.

	// kind 和type的区别 kind基本类型
	number := 3
	numberPtr := &number
	fmt.Println(reflect.ValueOf(numberPtr).Kind().String())
	fmt.Println(reflect.ValueOf(numberPtr).Type().String())
}
