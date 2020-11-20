package codec

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

// GobCodec is the implement of Codec.
type GobCodec struct {
	// conn 是由构建函数传入，通常是通过 TCP 或者 Unix 建立 socket 时得到的链接实例
	conn io.ReadWriteCloser
	// bind with conn
	buf *bufio.Writer
	dec *gob.Decoder
	enc *gob.Encoder
}

// 确保GobCodec实现了所有Codec interface的基类
var _ Codec = (*GobCodec)(nil)

// NewGobCodec is the constructor func of GobCodec
func NewGobCodec(conn io.ReadWriteCloser) Codec {
	buf := bufio.NewWriter(conn)
	return &GobCodec{
		conn: conn,                 // 通常是通过 TCP 或者 Unix 建立 socket 时得到的链接实例
		buf:  buf,                  // buf 是为了防止阻塞而创建的带缓冲的 Writer
		dec:  gob.NewDecoder(conn), // decoder bind conn
		enc:  gob.NewEncoder(buf),  // encoder bind buffer ,buffer bind conn
	}
}

// ReadHeader decode a Header from *Header with Gob coding
func (c *GobCodec) ReadHeader(h *Header) error {
	return c.dec.Decode(h)
}

// ReadBody decode a body from *body with Gob coding
// Here body must be pointer.Todo need a assert?
func (c *GobCodec) ReadBody(body interface{}) error {
	return c.dec.Decode(body)
}

// Write header and body into conn with gob coding
func (c *GobCodec) Write(h *Header, body interface{}) (err error) {
	// flush buffer into conn
	defer func() {
		_ = c.buf.Flush()
		if err != nil {
			_ = c.Close()
		}
	}()
	if err := c.enc.Encode(h); err != nil {
		log.Println("rpc codec: gob error encoding header:", err)
		return err
	}
	if err := c.enc.Encode(body); err != nil {
		log.Println("rpc codec: gob error encoding body:", err)
		return err
	}
	return nil
}

// Close close the conn
func (c *GobCodec) Close() error {
	return c.conn.Close()
}
