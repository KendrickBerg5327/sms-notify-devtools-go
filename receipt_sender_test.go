package main

import "testing"

func TestMessageForBuildEvents(t *testing.T) {
	tests := []struct {
		name, status, want string
		send               bool
	}{
		{"failed build", "failed", "api build failed on main", true},
		{"release", "released", "api release completed on main", true},
		{"queued", "queued", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, send := messageFor(BuildEvent{Project: "api", Branch: "main", Status: tt.status})
			if got != tt.want || send != tt.send {
				t.Fatalf("got %q,%v want %q,%v", got, send, tt.want, tt.send)
			}
		})
	}
}
