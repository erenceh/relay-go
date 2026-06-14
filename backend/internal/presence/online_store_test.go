package presence

import (
	"sort"
	"testing"
)

func TestInMemoryOnlineStore(t *testing.T) {
	tests := []struct {
		name    string
		ops     []struct {
			username string
			online   bool
		}
		wantOnline []string
	}{
		{
			name: "single user comes online",
			ops: []struct {
				username string
				online   bool
			}{
				{"alice", true},
			},
			wantOnline: []string{"alice"},
		},
		{
			name: "single user goes offline",
			ops: []struct {
				username string
				online   bool
			}{
				{"alice", true},
				{"alice", false},
			},
			wantOnline: []string{},
		},
		{
			name: "multiple users online",
			ops: []struct {
				username string
				online   bool
			}{
				{"alice", true},
				{"bob", true},
				{"carol", true},
			},
			wantOnline: []string{"alice", "bob", "carol"},
		},
		{
			name: "some users go offline",
			ops: []struct {
				username string
				online   bool
			}{
				{"alice", true},
				{"bob", true},
				{"carol", true},
				{"bob", false},
			},
			wantOnline: []string{"alice", "carol"},
		},
		{
			name:       "empty store returns no users",
			ops:        nil,
			wantOnline: []string{},
		},
		{
			name: "setting offline for unknown user is a no-op",
			ops: []struct {
				username string
				online   bool
			}{
				{"alice", true},
				{"unknown", false},
			},
			wantOnline: []string{"alice"},
		},
		{
			name: "setting online twice is idempotent",
			ops: []struct {
				username string
				online   bool
			}{
				{"alice", true},
				{"alice", true},
			},
			wantOnline: []string{"alice"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewOnlineStore()
			for _, op := range tt.ops {
				store.SetOnline(op.username, op.online)
			}

			got := store.ListOnline()
			sort.Strings(got)
			sort.Strings(tt.wantOnline)

			if len(got) != len(tt.wantOnline) {
				t.Fatalf("ListOnline() = %v, want %v", got, tt.wantOnline)
			}
			for i := range got {
				if got[i] != tt.wantOnline[i] {
					t.Errorf("ListOnline()[%d] = %q, want %q", i, got[i], tt.wantOnline[i])
				}
			}
		})
	}
}
