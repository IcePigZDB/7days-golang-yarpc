package yarpc

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Foo int

type Args struct{ Num1, Num2 int }

func (f Foo) Sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

func (f Foo) sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

// test util
func _assert(condition bool, msg string, v ...interface{}) {
	if !condition {
		panic(fmt.Sprintf("assertion failed:"+msg, v...))
	}
}
func TestNewService(t *testing.T) {
	var foo Foo
	s := newService(&foo)
	_assert(len(s.method) == 1, "wrong service Mehtod,expect 1, but got %d", len(s.method))
	// Sum pass sum fail
	mType := s.method["Sum"]
	_assert(mType != nil, "wrong Method,Sum shouldn't nil ")
}
func TestMethodType_call(t *testing.T) {
	var foo Foo
	s := newService(&foo)
	mType := s.method["Sum"]
	argv := mType.newArgv()
	replyv := mType.newReplyv()
	argv.Set(reflect.ValueOf(Args{Num1: 1, Num2: 2}))
	err := s.call(mType, argv, replyv)
	assert.Equal(t, err, nil, "call should not return error")
}
