package protocol

import (
	"bytes"
	"encoding/binary"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFrameRoundTrip(t *testing.T) {
	for _, tt := range []struct {
		name     string
		input    []byte
		run      func(t *testing.T, conn net.Conn)
		wantData []byte
		wantErr  bool
	}{
		{
			name:     "error when empty message",
			input:    []byte{},
			wantData: []byte{},
			wantErr:  false,
		},
		{
			name:     "normal message",
			input:    []byte("hello"),
			wantData: []byte("hello"),
			wantErr:  false,
		},
		{
			name:     "max size",
			input:    bytes.Repeat([]byte("a"), 1024),
			wantData: bytes.Repeat([]byte("a"), 1024),
			wantErr:  false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server, client := net.Pipe()
			defer server.Close()
			defer client.Close()

			go func() {
				WriteMessage(client, tt.input)
			}()

			frame, err := ReadMessage(server)
			require.NoError(t, err)
			assert.Equal(t, tt.wantData, frame.Data)
			assert.Equal(t, uint32(len(tt.wantData)), frame.Length)
		})
	}
}

func TestReadMessageOversized(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	// write a header claiming 2MB payload
	go func() {
		buf := make([]byte, 4)
		binary.BigEndian.PutUint32(buf, 2<<20)
		client.Write(buf)
	}()

	_, err := ReadMessage(server)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too large")
}
