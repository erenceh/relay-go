package protocol

import (
	"bytes"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFrameRoundTrip(t *testing.T) {
	for _, tt := range []struct {
		name     string
		input    []byte
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
