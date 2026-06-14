package events

import (
	"encoding/json"
	"fmt"

	nats "github.com/nats-io/nats.go"
)

const (
	SubjectUserJoined   = "user.joined"
	SubjectUserLeft     = "user.left"
	SubjectMessageSent  = "message.sent"
	SubjectUserPresence = "user.presence"
)

func Connect(url string) (*nats.Conn, error) {
	conn, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to nats server: %w", err)
	}

	return conn, nil
}

func Publish(nc *nats.Conn, subject string, event any) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("error serializing event: %w", err)
	}

	if err := nc.Publish(subject, data); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}
	return nil
}
