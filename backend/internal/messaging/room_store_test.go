package messaging

import (
	"sort"
	"testing"

	"github.com/erenceh/relay-go/backend/internal/domain"
	"github.com/erenceh/relay-go/backend/internal/repository"
)

// stubRoomRepo is an in-memory RoomRepository for testing.
type stubRoomRepo struct {
	rooms map[string]*domain.Room
}

func newStubRoomRepo() *stubRoomRepo {
	return &stubRoomRepo{rooms: make(map[string]*domain.Room)}
}

func (s *stubRoomRepo) Create(room *domain.Room) error {
	r := *room
	s.rooms[room.Name] = &r
	return nil
}

func (s *stubRoomRepo) FindByRoomName(name string) (*domain.Room, error) {
	r, ok := s.rooms[name]
	if !ok {
		return nil, nil
	}
	return r, nil
}

func (s *stubRoomRepo) FindByRoomID(id string) (*domain.Room, error) {
	for _, r := range s.rooms {
		if r.ID == id {
			return r, nil
		}
	}
	return nil, nil
}

// Ensure stubRoomRepo satisfies the interface.
var _ repository.RoomRepository = (*stubRoomRepo)(nil)

// helpers

func newStore() *InMemoryRoomStore {
	return NewRoomStore(newStubRoomRepo())
}

func sortedStrings(s []string) []string {
	out := make([]string, len(s))
	copy(out, s)
	sort.Strings(out)
	return out
}

// TestJoinRoom covers JoinRoom and the side-effects visible via ListRooms,
// ListRoomMembers, and GetRoomID.
func TestJoinRoom(t *testing.T) {
	tests := []struct {
		name        string
		joins       []struct{ room, user string }
		wantRooms   []string
		wantMembers map[string][]string
	}{
		{
			name:      "single user joins a new room",
			joins:     []struct{ room, user string }{{"general", "alice"}},
			wantRooms: []string{"general"},
			wantMembers: map[string][]string{
				"general": {"alice"},
			},
		},
		{
			name: "two users join the same room",
			joins: []struct{ room, user string }{
				{"general", "alice"},
				{"general", "bob"},
			},
			wantRooms: []string{"general"},
			wantMembers: map[string][]string{
				"general": {"alice", "bob"},
			},
		},
		{
			name: "users join different rooms",
			joins: []struct{ room, user string }{
				{"general", "alice"},
				{"random", "bob"},
			},
			wantRooms: []string{"general", "random"},
			wantMembers: map[string][]string{
				"general": {"alice"},
				"random":  {"bob"},
			},
		},
		{
			name: "same user joins same room twice is idempotent",
			joins: []struct{ room, user string }{
				{"general", "alice"},
				{"general", "alice"},
			},
			wantRooms: []string{"general"},
			wantMembers: map[string][]string{
				"general": {"alice"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newStore()

			for _, j := range tt.joins {
				if err := store.JoinRoom(j.room, j.user); err != nil {
					t.Fatalf("JoinRoom(%q, %q) unexpected error: %v", j.room, j.user, err)
				}
			}

			gotRooms := sortedStrings(store.ListRooms())
			wantRooms := sortedStrings(tt.wantRooms)
			if len(gotRooms) != len(wantRooms) {
				t.Fatalf("ListRooms() = %v, want %v", gotRooms, wantRooms)
			}
			for i := range gotRooms {
				if gotRooms[i] != wantRooms[i] {
					t.Errorf("ListRooms()[%d] = %q, want %q", i, gotRooms[i], wantRooms[i])
				}
			}

			for room, wantMembers := range tt.wantMembers {
				gotMembers, err := store.ListRoomMembers(room)
				if err != nil {
					t.Fatalf("ListRoomMembers(%q) unexpected error: %v", room, err)
				}
				gotMembers = sortedStrings(gotMembers)
				wantMembers = sortedStrings(wantMembers)
				if len(gotMembers) != len(wantMembers) {
					t.Fatalf("ListRoomMembers(%q) = %v, want %v", room, gotMembers, wantMembers)
				}
				for i := range gotMembers {
					if gotMembers[i] != wantMembers[i] {
						t.Errorf("ListRoomMembers(%q)[%d] = %q, want %q", room, i, gotMembers[i], wantMembers[i])
					}
				}
			}
		})
	}
}

func TestLeaveRoom(t *testing.T) {
	tests := []struct {
		name        string
		joins       []struct{ room, user string }
		leaves      []struct{ room, user string }
		wantRooms   []string
		wantMembers map[string][]string
		wantErrs    []string // room names whose LeaveRoom should return an error
	}{
		{
			name:      "last member leaves removes the room",
			joins:     []struct{ room, user string }{{"general", "alice"}},
			leaves:    []struct{ room, user string }{{"general", "alice"}},
			wantRooms: []string{},
		},
		{
			name: "one of two members leaves",
			joins: []struct{ room, user string }{
				{"general", "alice"},
				{"general", "bob"},
			},
			leaves:    []struct{ room, user string }{{"general", "alice"}},
			wantRooms: []string{"general"},
			wantMembers: map[string][]string{
				"general": {"bob"},
			},
		},
		{
			name:     "leaving a non-existent room returns an error",
			joins:    nil,
			leaves:   []struct{ room, user string }{{"ghost", "alice"}},
			wantErrs: []string{"ghost"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newStore()

			for _, j := range tt.joins {
				if err := store.JoinRoom(j.room, j.user); err != nil {
					t.Fatalf("JoinRoom(%q, %q) unexpected error: %v", j.room, j.user, err)
				}
			}

			wantErrSet := make(map[string]bool, len(tt.wantErrs))
			for _, r := range tt.wantErrs {
				wantErrSet[r] = true
			}

			for _, l := range tt.leaves {
				err := store.LeaveRoom(l.room, l.user)
				if wantErrSet[l.room] {
					if err == nil {
						t.Errorf("LeaveRoom(%q, %q) expected error, got nil", l.room, l.user)
					}
					continue
				}
				if err != nil {
					t.Fatalf("LeaveRoom(%q, %q) unexpected error: %v", l.room, l.user, err)
				}
			}

			gotRooms := sortedStrings(store.ListRooms())
			wantRooms := sortedStrings(tt.wantRooms)
			if len(gotRooms) != len(wantRooms) {
				t.Fatalf("ListRooms() = %v, want %v", gotRooms, wantRooms)
			}
			for i := range gotRooms {
				if gotRooms[i] != wantRooms[i] {
					t.Errorf("ListRooms()[%d] = %q, want %q", i, gotRooms[i], wantRooms[i])
				}
			}

			for room, wantMembers := range tt.wantMembers {
				gotMembers, err := store.ListRoomMembers(room)
				if err != nil {
					t.Fatalf("ListRoomMembers(%q) unexpected error: %v", room, err)
				}
				gotMembers = sortedStrings(gotMembers)
				wantMembers = sortedStrings(wantMembers)
				if len(gotMembers) != len(wantMembers) {
					t.Fatalf("ListRoomMembers(%q) = %v, want %v", room, gotMembers, wantMembers)
				}
				for i := range gotMembers {
					if gotMembers[i] != wantMembers[i] {
						t.Errorf("ListRoomMembers(%q)[%d] = %q, want %q", room, i, gotMembers[i], wantMembers[i])
					}
				}
			}
		})
	}
}

func TestListRoomMembers(t *testing.T) {
	tests := []struct {
		name        string
		joins       []struct{ room, user string }
		queryRoom   string
		wantMembers []string
		wantErr     bool
	}{
		{
			name:        "returns members of an existing room",
			joins:       []struct{ room, user string }{{"general", "alice"}, {"general", "bob"}},
			queryRoom:   "general",
			wantMembers: []string{"alice", "bob"},
		},
		{
			name:      "returns error for non-existent room",
			joins:     nil,
			queryRoom: "ghost",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newStore()
			for _, j := range tt.joins {
				if err := store.JoinRoom(j.room, j.user); err != nil {
					t.Fatalf("JoinRoom unexpected error: %v", err)
				}
			}

			got, err := store.ListRoomMembers(tt.queryRoom)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ListRoomMembers(%q) expected error, got nil", tt.queryRoom)
				}
				return
			}
			if err != nil {
				t.Fatalf("ListRoomMembers(%q) unexpected error: %v", tt.queryRoom, err)
			}

			got = sortedStrings(got)
			want := sortedStrings(tt.wantMembers)
			if len(got) != len(want) {
				t.Fatalf("ListRoomMembers(%q) = %v, want %v", tt.queryRoom, got, want)
			}
			for i := range got {
				if got[i] != want[i] {
					t.Errorf("ListRoomMembers(%q)[%d] = %q, want %q", tt.queryRoom, i, got[i], want[i])
				}
			}
		})
	}
}

func TestGetRoomID(t *testing.T) {
	tests := []struct {
		name      string
		joins     []struct{ room, user string }
		queryRoom string
		wantEmpty bool
	}{
		{
			name:      "returns non-empty ID for existing room",
			joins:     []struct{ room, user string }{{"general", "alice"}},
			queryRoom: "general",
			wantEmpty: false,
		},
		{
			name:      "returns empty string for non-existent room",
			joins:     nil,
			queryRoom: "ghost",
			wantEmpty: true,
		},
		{
			name: "ID is stable across multiple joins to the same room",
			joins: []struct{ room, user string }{
				{"general", "alice"},
				{"general", "bob"},
			},
			queryRoom: "general",
			wantEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newStore()
			for _, j := range tt.joins {
				if err := store.JoinRoom(j.room, j.user); err != nil {
					t.Fatalf("JoinRoom unexpected error: %v", err)
				}
			}

			id := store.GetRoomID(tt.queryRoom)

			if tt.wantEmpty && id != "" {
				t.Errorf("GetRoomID(%q) = %q, want empty string", tt.queryRoom, id)
			}
			if !tt.wantEmpty && id == "" {
				t.Errorf("GetRoomID(%q) = empty, want non-empty ID", tt.queryRoom)
			}

			// Verify stability: calling again returns the same value.
			if !tt.wantEmpty {
				if id2 := store.GetRoomID(tt.queryRoom); id2 != id {
					t.Errorf("GetRoomID(%q) not stable: first=%q second=%q", tt.queryRoom, id, id2)
				}
			}
		})
	}
}
