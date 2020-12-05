package yarpc

import (
	"go/ast"
	"log"
	"reflect"
	"sync/atomic"
)

// registered service 'type struct
type methodType struct {
	method    reflect.Method // 方法本身
	ArgType   reflect.Type   // 传参类型
	ReplyType reflect.Type   // 返回值类型
	numCalls  uint64         // 调用次数统计
}

// NumCalls return the number of a method called atomic
func (m *methodType) NumCalls() uint64 {
	return atomic.LoadUint64(&m.numCalls)
}

func (m *methodType) newArgv() reflect.Value {
	var argv reflect.Value
	// kind basic type:ptr
	// type            *int *float *double and so on
	// arg may be a pointer type, or a value type
	// 指针类型和值类型创建实例的方式有细微区别
	if m.ArgType.Kind() == reflect.Ptr {
		// Elem returns the value(reflect.Value) that the interface v
		// contains or that the pointer v points to.
		// It panics if v's Kind is not Interface or Ptr.
		// It returns the zero Value if v is nil.
		// New 返回Type的ptr
		// 先Elem再New出来的是ptr
		argv = reflect.New(m.ArgType.Elem())
	} else {
		// 先New再Elem出来的是值
		argv = reflect.New(m.ArgType).Elem()
	}
	return argv
}

func (m *methodType) newReplyv() reflect.Value {
	// reply must be a pointer type
	replyv := reflect.New(m.ReplyType.Elem())
	// reply有两种 map/slice
	switch m.ReplyType.Elem().Kind() {
	case reflect.Map:
		// 根据method上注册的reply生成一个空的reply
		replyv.Elem().Set(reflect.MakeMap(m.ReplyType.Elem()))
	case reflect.Slice:
		replyv.Elem().Set(reflect.MakeSlice(m.ReplyType.Elem(), 0, 0))
	}
	return replyv
}

// registered service
type service struct {
	name   string                 // 方法名字
	typ    reflect.Type           // typ 是结构体的类型
	rcvr   reflect.Value          // rcvr 即结构体的实例本身，保留 rcvr 是因为在调用时需要 rcvr 作为第 0 个参数
	method map[string]*methodType // 存储service的所有方法
}

func newService(rcvr interface{}) *service {
	s := new(service)
	s.rcvr = reflect.ValueOf(rcvr)                  // set service struct value
	s.name = reflect.Indirect(s.rcvr).Type().Name() // set service name
	s.typ = reflect.TypeOf(rcvr)                    //set service type
	if !ast.IsExported(s.name) {
		log.Fatalf("rpc server: %s is not a valid service name", s.name)
	}
	s.registerMethods() // register methods of service
	return s
}

func (s *service) registerMethods() {
	s.method = make(map[string]*methodType)
	for i := 0; i < s.typ.NumMethod(); i++ {
		method := s.typ.Method(i)
		mType := method.Type
		// 两个导出或内置类型的入参（反射时为 3 个，第 0 个是自身，
		// 类似于 python 的 self，java 中的 this）
		// 返回值有且只有 1 个，类型为 error
		if mType.NumIn() != 3 || mType.NumOut() != 1 {
			continue
		}
		// TypeOf error指针返回的是 *error,
		// Elem方法返回的是error的reflect.Value类型
		if mType.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
			continue
		}
		argType, replyType := mType.In(1), mType.In(2)
		// check Exported in first letter
		if !isExportedOrBuiltinType(argType) || !isExportedOrBuiltinType(replyType) {
			continue
		}
		s.method[method.Name] = &methodType{
			method:    method,
			ArgType:   argType,
			ReplyType: replyType,
		}
		log.Printf("rpc server: register %s,%s\n", s.name, method.Name)
	}
}

func isExportedOrBuiltinType(t reflect.Type) bool {
	return ast.IsExported(t.Name()) || t.PkgPath() == ""
}

func (s *service) call(m *methodType, argv, replyv reflect.Value) error {
	atomic.AddUint64(&m.numCalls, 1)
	// reflect.Method.Func
	f := m.method.Func
	returnValues := f.Call([]reflect.Value{s.rcvr, argv, replyv})
	if errInter := returnValues[0].Interface(); errInter != nil {
		// 类型转换
		return errInter.(error)
	}
	return nil
}
