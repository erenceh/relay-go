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

	slog.Info("connected to", "addr", address)

	// receive prompt from server
	frame, err := protocol.ReadMessage(conn)
	if err != nil {
		slog.Error("failed to receive prompt", "err", err)
		os.Exit(1)
	}

	// read username from local terminal
	var username string
	fmt.Print(string(frame.Data) + " ")
	fmt.Scan(&username)

	// send username to server
	if err := protocol.WriteMessage(conn, []byte(username)); err != nil {
		slog.Error("failed to send username", "err", err)
		os.Exit(1)
	}

	done := make(chan struct{})
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		readMessage(conn, done, username)
	}()

	fmt.Printf("\nwelcome %s! type '/help' for commands, '/quit' to disconnect\n", username)
	fmt.Print(username + ": ")

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
			fmt.Print(username + ": ")
		}
	}
	wg.Wait()
}

func readMessage(conn net.Conn, done chan struct{}, username string) {
	defer close(done)
	for {
		frame, err := protocol.ReadMessage(conn)
		if err != nil {
			slog.Info("disconnected from server")
			fmt.Println("you have been disconnected from the server")
			return
		}
		// TODO: replace with proper terminal UI library (bubbletea) in a future version
		fmt.Printf("\r%s\n%s: ", frame.Data, username)
	}
}
