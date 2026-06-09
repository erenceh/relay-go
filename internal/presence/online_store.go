package presence

import "sync"

// OnlineStore tracks which users are currently online.
type OnlineStore interface {
	// SetOnline marks the user as online or offline.
	SetOnline(username string, online bool)
	// ListOnline returns the usernames of all currently online users.
	ListOnline() []string
}

// InMemoryOnlineStore is a thread-safe, in-memory implementation of OnlineStore.
type InMemoryOnlineStore struct {
	mu     sync.Mutex
	online map[string]bool
}

// NewOnlineStore returns an initialized InMemoryOnlineStore.
func NewOnlineStore() *InMemoryOnlineStore {
	return &InMemoryOnlineStore{online: make(map[string]bool)}
}

func (os *InMemoryOnlineStore) SetOnline(username string, online bool) {
	os.mu.Lock()
	defer os.mu.Unlock()

	if online {
		os.online[username] = true
	} else {
		delete(os.online, username)
	}
}

func (os *InMemoryOnlineStore) ListOnline() []string {
	os.mu.Lock()
	defer os.mu.Unlock()

	users := make([]string, 0, len(os.online))
	for username := range os.online {
		users = append(users, username)
	}

	return users
}
