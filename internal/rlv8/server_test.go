package rlv8

import (
	"context"
	"testing"

	pb "quoridor/internal/rlv8/pb"
)

func TestResetUsesV8Contract(t *testing.T) {
	server := NewServer()
	state, err := server.Reset(context.Background(), &pb.ResetRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(state.GetObservation()) != ObservationSize {
		t.Fatalf("observation size = %d, want %d", len(state.GetObservation()), ObservationSize)
	}
	if len(state.GetActionMask()) != ActionSpaceSize {
		t.Fatalf("mask size = %d, want %d", len(state.GetActionMask()), ActionSpaceSize)
	}
	if state.GetCurrentPlayer() != 1 {
		t.Fatalf("current player = %d, want 1", state.GetCurrentPlayer())
	}
}

func TestStepRejectsInvalidMaskedAction(t *testing.T) {
	server := NewServer()
	state, _ := server.Reset(context.Background(), &pb.ResetRequest{})
	invalidAction := -1
	for action, valid := range state.GetActionMask() {
		if !valid {
			invalidAction = action
			break
		}
	}
	if invalidAction < 0 {
		t.Fatal("expected the initial mask to contain an invalid action")
	}
	if _, err := server.Step(context.Background(), &pb.StepRequest{Action: int32(invalidAction)}); err == nil {
		t.Fatal("expected an invalid wall action to be rejected")
	}
}

func TestStepMovesAndAdvancesTurn(t *testing.T) {
	server := NewServer()
	_, _ = server.Reset(context.Background(), &pb.ResetRequest{})
	state, err := server.Step(context.Background(), &pb.StepRequest{Action: 0})
	if err != nil {
		t.Fatal(err)
	}
	if state.GetCurrentPlayer() != 2 {
		t.Fatalf("current player = %d, want 2", state.GetCurrentPlayer())
	}
	if state.GetObservation()[4*9+8] != 1 {
		t.Fatal("player 2 position was not vertically normalized")
	}
	if !state.GetActionMask()[0] {
		t.Fatal("canonical forward action should be valid for player 2")
	}
	if state.GetInfo()["turns"] != 1 {
		t.Fatalf("turns = %v, want 1", state.GetInfo()["turns"])
	}
}
