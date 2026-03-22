package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

type Frame struct {
	Length uint32
	Data   []byte
}

func ReadMessage(conn net.Conn) (*Frame, error) {
	buf := make([]byte, 4)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return nil, fmt.Errorf("read header error: %w", err)
	}

	length := binary.BigEndian.Uint32(buf)
	data := make([]byte, length)
	if _, err := io.ReadFull(conn, data); err != nil {
		return nil, fmt.Errorf("read body error: %w", err)
	}

	return &Frame{Length: length, Data: data}, nil
}

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
