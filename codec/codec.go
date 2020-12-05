package codec

import "io"

// Header will be used by client to request and server to reply
type Header struct {
	ServiceMethod string // format "Service.Method"
	Seq           uint64 // sequence number chosen by client
	ServerID      int    // sequence used by server
	Error         string
}

// Codec is the gob/json encoder/decoder interface.
type Codec interface {
	io.Closer
	// decode header
	ReadHeader(*Header) error
	// decode body
	ReadBody(interface{}) error
	// encode header & body
	Write(*Header, interface{}) error
}

// NewCodecFunc is a Codec encoder/decoder constructor func
// return a Codec object
// ReadWriteCloser is the interface that groups the basic Read, Write and Close methods.
type NewCodecFunc func(io.ReadWriteCloser) Codec

// Type enmu coding type.
type Type string

const (
	GobType  Type = "application/gob"
	JsonType Type = "application/json"
)

// NewCodecFuncMap is a Codec encoder/decoder constructor function map
var NewCodecFuncMap map[Type]NewCodecFunc

func init() {
	NewCodecFuncMap = make(map[Type]NewCodecFunc)
	NewCodecFuncMap[GobType] = NewGobCodec
	NewCodecFuncMap[JsonType] = NewJsonCodec
}
