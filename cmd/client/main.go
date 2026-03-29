package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/erenceh/relay-go/internal/protocol"
)

func main() {
	network := "tcp"
	address := flag.String("addr", "relay.erenceh.dev:8080", "server address")
	flag.Parse()

	// --- Connection setup ---
	conn, err := net.Dial(network, *address)
	if err != nil {
		slog.Error("failed to connect to", "addr", *address)
		os.Exit(1)
	}
	defer conn.Close()
	slog.Info("connected to", "addr", *address)

	inputCh := make(chan string)
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			inputCh <- scanner.Text()
		}
		close(inputCh)
	}()

	userCreds := runAuthHandshake(conn, inputCh)
	runSession(conn, userCreds, network, *address, inputCh)
}

// Credentials stores the minimum information needed to reconnect
// without prompting the user again.
// RefreshToken is used instead of password, password is never stored.
type Credentials struct {
	Username     string
	RefreshToken string
}

// runAuthHandshake recieves authentication prompts from server,
// read from local terminal, send back.
func runAuthHandshake(conn net.Conn, inputCh chan string) Credentials {
	drainInput(inputCh)
	var username string
	var refreshToken string
authLoop:
	for {
		frame, err := protocol.ReadMessage(conn)
		if err != nil {
			slog.Error("failed to receive prompt", "err", err)
			os.Exit(1)
		}

		prompt := strings.ToLower(string(frame.Data))
		if strings.Contains(prompt, "already logged in") {
			fmt.Println("another session is already active for this account")
			continue authLoop
		}

		if strings.Contains(prompt, "successful") {
			fmt.Println(prompt)
			continue authLoop
		}

		if strings.HasPrefix(prompt, "refresh:") {
			refreshToken = strings.TrimPrefix(prompt, "refresh:")
			break authLoop
		}

		if !strings.HasSuffix(strings.TrimSpace(prompt), ":") {
			fmt.Println(prompt)
			continue authLoop
		}

		var input string
		if strings.Contains(prompt, "password") {
			fmt.Print(prompt + " ")
			// TODO: hide password input in v5 with bubbletea terminal UI
			// term.ReadPassword conflicts with inputCh goroutine - known limitation
			input = <-inputCh
		} else {
			fmt.Print(prompt + " ")
			input = <-inputCh

			if strings.Contains(prompt, "username") {
				username = input
			}
		}

		if err := protocol.WriteMessage(conn, []byte(input)); err != nil {
			slog.Error("failed to send username", "err", err)
			os.Exit(1)
		}
	}

	return Credentials{
		Username:     username,
		RefreshToken: refreshToken,
	}
}

func drainInput(inputCh chan string) {
	for {
		select {
		case <-inputCh:
		default:
			return
		}
	}
}

func runSession(
	conn net.Conn,
	userCreds Credentials,
	network string,
	address string,
	inputCh chan string,
) {
	for {
		done := make(chan struct{})
		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			runMessageLoop(conn, done, userCreds.Username)
		}()

		drainInput(inputCh)
		fmt.Printf("\nwelcome %s! type '/help' for commands, '/quit' to disconnect\n", userCreds.Username)
		fmt.Print(userCreds.Username + ": ")

		disconnected := false
		for {
			select {
			case <-done:
				disconnected = true
			case line, ok := <-inputCh:
				if !ok {
					return
				}
				if line == "/quit" {
					slog.Info("disconnecting")
					return
				}
				if err := protocol.WriteMessage(conn, []byte(line)); err != nil {
					slog.Warn("failed to send message", "err", err)
					disconnected = true
				}
				fmt.Print(userCreds.Username + ": ")
			}
			if disconnected {
				break
			}
		}

		if !disconnected {
			return
		}

		wg.Wait()
		newConn, newCreds, err := reconnect(network, address, userCreds, inputCh)
		if err != nil {
			fmt.Println(err)
			return
		}
		conn = newConn
		userCreds = newCreds
	}
}

// runMessageLoop runs in a goroutine and continuously reads incoming messages
// from the server, printing them to stdout. Closes done when the server
// disconnects so the stdin loop can exit cleanly.
func runMessageLoop(conn net.Conn, done chan struct{}, username string) {
	defer close(done)
	for {
		frame, err := protocol.ReadMessage(conn)
		if err != nil {
			slog.Info("disconnected from server")
			fmt.Println("\nyou have been disconnected...")
			return
		}
		// TODO: replace with proper terminal UI library (bubbletea) in v5
		fmt.Printf("\r%s\n%s: ", frame.Data, username)
	}
}

func reconnect(network, address string, creds Credentials, inputCh chan string) (net.Conn, Credentials, error) {
	maxAttempts := 6
	delay := time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		fmt.Printf("reconnecting... attempts %d/%d\n", attempt, maxAttempts)

		conn, err := net.Dial(network, address)
		if err != nil {
			time.Sleep(delay)
			delay *= 2
			if delay > 30*time.Second {
				delay = 30 * time.Second
			}
			continue
		}

		newCreds, err := runSilentAuth(conn, creds)
		if err != nil {
			conn.Close()
			slog.Info("silent auth failed", "err", err)
			fmt.Println("session expired. please log in again.")

			freshConn, dialErr := net.Dial(network, address)
			if dialErr != nil {
				continue
			}
			conn = freshConn
			newCreds = runAuthHandshake(conn, inputCh)
		}

		return conn, newCreds, nil
	}

	return nil, Credentials{}, errors.New("failed to reconnect after max attempts")
}

func runSilentAuth(conn net.Conn, creds Credentials) (Credentials, error) {
	var refreshToken string
authLoop:
	for {
		frame, err := protocol.ReadMessage(conn)
		if err != nil {
			slog.Error("failed to receive prompt", "err", err)
			return Credentials{}, fmt.Errorf("connection lost during silent auth: %w", err)
		}
		prompt := strings.ToLower(string(frame.Data))

		if strings.Contains(prompt, "successful") {
			fmt.Println(prompt)
			continue authLoop
		}

		if strings.HasPrefix(prompt, "refresh:") {
			refreshToken = strings.TrimPrefix(prompt, "refresh:")
			break authLoop
		}

		if !strings.HasSuffix(strings.TrimSpace(prompt), ":") {
			return Credentials{}, fmt.Errorf("%s", prompt)
		}

		if strings.Contains(prompt, "refresh") {
			protocol.WriteMessage(conn, []byte(creds.RefreshToken))
		} else {
			protocol.WriteMessage(conn, []byte("/refresh"))
		}
	}

	return Credentials{
		Username:     creds.Username,
		RefreshToken: refreshToken,
	}, nil
}
