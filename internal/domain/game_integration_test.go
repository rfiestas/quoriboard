package domain_test

import (
	"testing"

	"quoridor/internal/domain"
)

func TestIntegration_PlayerMoveSequenceAndWin(t *testing.T) {
	g := helperSetupGame(2)
	p1 := g.Players[0]
	p2 := g.Players[1]

	// Keep player 2 out of the way for a direct run to the goal.
	p2.GridX = 8
	p2.GridY = 8

	p1.GridX = 4
	p1.GridY = 2

	if g.ActivePlayer() == nil || g.ActivePlayer().ID != 1 {
		t.Fatal("expected player 1 to start")
	}

	for step := 2; step > 0; step-- {
		assertContainsMove(t, getValidMoves(g, p1), 4, step-1)
		p1.GridY = step - 1

		if step > 1 {
			g.NextTurn()
			if g.ActivePlayer() == nil || g.ActivePlayer().ID != 2 {
				t.Fatal("expected player 2 to be active between player 1 moves")
			}
			g.NextTurn()
			if g.ActivePlayer() == nil || g.ActivePlayer().ID != 1 {
				t.Fatal("expected player 1 to resume after player 2")
			}
		}
	}

	if !g.CheckWin(p1) {
		t.Fatal("expected player 1 to win after reaching the target row")
	}
}

func TestIntegration_PlaceWallKeepsPaths(t *testing.T) {
	g := helperSetupGame(2)
	p1 := g.Players[0]
	p2 := g.Players[1]

	// Move second player away so the wall placement only tests domain path logic.
	p2.GridX = 8
	p2.GridY = 8

	if !g.PlaceWall(3, 4, domain.WallHorizontal, p1) {
		t.Fatal("expected PlaceWall to succeed for a valid placement")
	}

	if len(g.Walls) != 1 {
		t.Fatalf("expected one wall after placement, got %d", len(g.Walls))
	}

	if p1.WallsLeft != 9 {
		t.Fatalf("expected player 1 to have 9 walls left, got %d", p1.WallsLeft)
	}

	moves1 := getValidMoves(g, p1)
	if len(moves1) == 0 {
		t.Fatalf("expected player 1 to still have valid moves after placing a valid wall")
	}

	moves2 := getValidMoves(g, p2)
	if len(moves2) == 0 {
		t.Fatalf("expected player 2 to still have valid moves after player 1 placed a wall")
	}
}

func TestIntegration_PlaceWallBlockedByInvalidPlacement(t *testing.T) {
	g := helperSetupGame(2)
	p1 := g.Players[0]
	p2 := g.Players[1]

	// Move player 2 into the corner and create a wall that already blocks one side.
	p2.GridX = 0
	p2.GridY = 0
	g.Walls = append(g.Walls, domain.Wall{
		GridX:       0,
		GridY:       0,
		Orientation: domain.WallVertical,
		OwnerID:     p1.ID,
	})

	initialWalls := len(g.Walls)
	initialWallsLeft := p1.WallsLeft

	if g.PlaceWall(0, 0, domain.WallHorizontal, p1) {
		t.Fatal("expected PlaceWall to fail for an invalid blocked placement")
	}

	if len(g.Walls) != initialWalls {
		t.Fatalf("expected no new wall to be added, got %d walls", len(g.Walls))
	}

	if p1.WallsLeft != initialWallsLeft {
		t.Fatalf("expected player 1 walls left to remain %d, got %d", initialWallsLeft, p1.WallsLeft)
	}
}
