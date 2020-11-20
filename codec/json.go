package codec

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
)

// JsonCodec is the implement of Codec.
type JsonCodec struct {
	// conn 是由构建函数传入，通常是通过 TCP 或者 Unix 建立 socket 时得到的链接实例
	conn io.ReadWriteCloser
	// bind with conn
	buf *bufio.Writer
	dec *json.Decoder
	enc *json.Encoder
}

// 确保JsonCodec实现了所有Codec interface的基类
var _ Codec = (*JsonCodec)(nil)

// NewJsonCodec is the constructor func of JsonCodec
func NewJsonCodec(conn io.ReadWriteCloser) Codec {
	buf := bufio.NewWriter(conn)
	return &JsonCodec{
		conn: conn,                  // 通常是通过 TCP 或者 Unix 建立 socket 时得到的链接实例
		buf:  buf,                   // buf 是为了防止阻塞而创建的带缓冲的 Writer
		dec:  json.NewDecoder(conn), // decoder bind conn
		enc:  json.NewEncoder(buf),  // encoder bind buffer ,buffer bind conn
	}
}

// ReadHeader decode a Header from *Header with json coding
func (c *JsonCodec) ReadHeader(h *Header) error {
	return c.dec.Decode(h)
}

// ReadBody decode a body from *body with json coding
// Here body must be pointer.Todo need a assert?
func (c *JsonCodec) ReadBody(body interface{}) error {
	return c.dec.Decode(body)
}

// Write header and body into conn with json coding
func (c *JsonCodec) Write(h *Header, body interface{}) (err error) {
	// flush buffer into conn
	defer func() {
		_ = c.buf.Flush()
		if err != nil {
			_ = c.Close()
		}
	}()
	if err := c.enc.Encode(h); err != nil {
		log.Println("rpc codec: json error encoding header:", err)
		return err
	}
	if err := c.enc.Encode(body); err != nil {
		log.Println("rpc codec: json error encoding body:", err)
		return err
	}
	return nil
}

// Close close the conn
func (c *JsonCodec) Close() error {
	return c.conn.Close()
}
