package rlv8league

import (
	"reflect"
	"testing"

	"quoridor/internal/adapters/bot"
)

func TestNewServerOpponentModes(t *testing.T) {
	for _, mode := range []string{"heuristic", "minimax1", "minimax2", "minimax3"} {
		server, err := NewServer(mode)
		if err != nil {
			t.Fatalf("NewServer(%q) error: %v", mode, err)
		}
		if server.opponent == nil {
			t.Fatalf("NewServer(%q) returned no opponent", mode)
		}
	}

	if _, err := NewServer("unknown"); err == nil {
		t.Fatal("NewServer(unknown) succeeded")
	}
}

func TestResetRecreatesMinimaxOpponentState(t *testing.T) {
	server, err := NewServer("minimax2")
	if err != nil {
		t.Fatal(err)
	}
	minimax, ok := server.opponent.(*bot.MinimaxBot)
	if !ok {
		t.Fatalf("opponent type = %T, want *bot.MinimaxBot", server.opponent)
	}
	initialPointer := reflect.ValueOf(minimax).Pointer()

	if _, err := server.Reset(nil, nil); err != nil {
		t.Fatal(err)
	}
	resetMinimax, ok := server.opponent.(*bot.MinimaxBot)
	if !ok {
		t.Fatalf("reset opponent type = %T, want *bot.MinimaxBot", server.opponent)
	}
	if reflect.ValueOf(resetMinimax).Pointer() == initialPointer {
		t.Fatal("Reset reused the previous MinimaxBot instance")
	}
}
