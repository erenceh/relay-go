package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"os"
	"sync"

	"github.com/erenceh/relay-go/internal/protocol"
)

func main() {
	network := "tcp"
	address := ":8080"

	conn, err := net.Dial(network, address)
	if err != nil {
		slog.Error("failed to connect to", "addr", address)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Println("type '/quit' to disconnect")
	slog.Info("connected to", "addr", address)

	done := make(chan struct{})
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		readMessage(conn, done)
	}()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		select {
		case <-done:
			return
		default:
			line := scanner.Text()
			if line == "/quit" {
				slog.Info("disconnecting")
				return
			}
			if err := protocol.WriteMessage(conn, []byte(line)); err != nil {
				slog.Warn("failed to send message", "err", err)
				return
			}
		}
	}

	wg.Wait()
}

func readMessage(conn net.Conn, done chan struct{}) {
	defer close(done)
	for {
		frame, err := protocol.ReadMessage(conn)
		if err != nil {
			slog.Info("disconnected from server")
			fmt.Println("you have been disconnected from the server")
			return
		}
		fmt.Printf("%s\n", frame.Data)
	}
}
