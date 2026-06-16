package websocket

import (
	"net"
	"time"

	gorilla "github.com/gorilla/websocket"
)

// Adapter wraps a gorilla WebSocket connection and exposes it as an io.ReadWriter,
// making it compatible with the protocol.Conn interface used by ReadMessage and WriteMessage.
// Incoming WebSocket messages are buffered so that multiple small reads can be served
// from a single received frame.
type Adapter struct {
	conn *gorilla.Conn
	buf  []byte
}

// NewAdapter returns an Adapter backed by the given gorilla WebSocket connection.
func NewAdapter(conn *gorilla.Conn) *Adapter {
	return &Adapter{conn: conn}
}

// Read satisfies io.Reader. It reads the next WebSocket message into an internal buffer
// when the buffer is empty, then copies bytes into b until b is full or the buffer
// is exhausted.
func (a *Adapter) Read(b []byte) (int, error) {
	if len(a.buf) == 0 {
		_, data, err := a.conn.ReadMessage()
		if err != nil {
			return 0, err
		}
		a.buf = data
	}

	n := copy(b, a.buf)
	a.buf = a.buf[n:]
	return n, nil
}

// Write satisfies io.Writer. It sends b as a single WebSocket text message.
func (a *Adapter) Write(b []byte) (n int, err error) {
	if err := a.conn.WriteMessage(gorilla.TextMessage, b); err != nil {
		return 0, err
	}
	return len(b), nil
}

func (a *Adapter) Close() error {
	return a.conn.Close()
}

func (a *Adapter) LocalAddr() net.Addr {
	return a.conn.LocalAddr()
}

func (a *Adapter) RemoteAddr() net.Addr {
	return a.conn.RemoteAddr()
}

func (a *Adapter) SetDeadline(t time.Time) error {
	if err := a.conn.SetReadDeadline(t); err != nil {
		return err
	}
	return a.conn.SetWriteDeadline(t)
}

func (a *Adapter) SetReadDeadline(t time.Time) error {
	return a.conn.SetReadDeadline(t)
}

func (a *Adapter) SetWriteDeadline(t time.Time) error {
	return a.conn.SetWriteDeadline(t)
}
