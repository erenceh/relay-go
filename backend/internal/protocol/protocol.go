package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

// Frame represents a single length-prefixed protocol message.
// Length holds the byte count of Data as encoded in the 4-byte header.
type Frame struct {
	Length uint32
	Data   []byte
}

// ReadMessage reads one length-prefixed message from conn.
// It first reads the 4-byte big-endian length header, then reads
// exactly that many bytes as the payload.
// Returns an error if either read fails or the connection is closed mid-message.
func ReadMessage(conn net.Conn) (*Frame, error) {
	buf := make([]byte, 4)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return nil, fmt.Errorf("read header error: %w", err)
	}

	length := binary.BigEndian.Uint32(buf)

	const maxMessageSize = 1 << 20
	if length > maxMessageSize {
		return nil, fmt.Errorf("message too large: %d bytes", length)
	}

	data := make([]byte, length)
	if _, err := io.ReadFull(conn, data); err != nil {
		return nil, fmt.Errorf("read body error: %w", err)
	}

	return &Frame{Length: length, Data: data}, nil
}

// WriteMessage writes data to conn as a length-prefixed message.
// It sends a 4-byte big-endian length header followed by the payload bytes.
// Returns an error if either the header or payload write fails.
func WriteMessage(conn net.Conn, data []byte) error {
	length := uint32(len(data))
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, length)
	if _, err := conn.Write(buf); err != nil {
		return fmt.Errorf("write header error: %w", err)
	}
	if _, err := conn.Write(data); err != nil {
		return fmt.Errorf("write body error: %w", err)
	}

	return nil
}
