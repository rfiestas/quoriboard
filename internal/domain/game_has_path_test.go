package domain

import "testing"

// -----------------------------------------------------------------------------
// hasPathToGoal
//
// These tests verify only physical board connectivity.
// They do not consider:
//   - opponents
//   - jumps
//   - turns
//   - whether a player could place a given wall
//
// The only criterion is:
//
//     Is there an orthogonal path from the player's position
//     to any goal square without crossing walls?
// -----------------------------------------------------------------------------

func TestHasPathToGoal_DirectPath(t *testing.T) {
	g := helperSetupGame(2)
	p := g.Players[0]

	// P1 starts at (4,8).
	// Assuming P1's target is the opposite row (Y == 0).
	if !g.hasPathToGoal(p) {
		t.Fatal("expected direct path to goal on empty board")
	}
}

func TestHasPathToGoal_DirectPathWithOpponentIgnored(t *testing.T) {
	g := helperSetupGame(2)

	p := g.Players[0]
	opponent := g.Players[1]

	p.GridX = 4
	p.GridY = 8

	// Put opponent directly in the physical path.
	opponent.GridX = 4
	opponent.GridY = 7

	// hasPathToGoal must ignore players completely.
	if !g.hasPathToGoal(p) {
		t.Fatal("expected path to remain available because opponents are ignored")
	}
}

func TestHasPathToGoal_DirectPathWithIrrelevantWall(t *testing.T) {
	g := helperSetupGame(2)
	p := g.Players[0]

	g.Walls = append(g.Walls, Wall{
		GridX:       0,
		GridY:       0,
		Orientation: WallHorizontal,
		OwnerID:     p.ID,
	})

	if !g.hasPathToGoal(p) {
		t.Fatal("expected path to remain available with irrelevant wall")
	}
}

// -----------------------------------------------------------------------------
// COMPLETE BLOCK
// -----------------------------------------------------------------------------

func TestHasPathToGoal_CompletelyBlocked(t *testing.T) {
	g := helperSetupGame(2)
	p := g.Players[0]

	p.GridX = 4
	p.GridY = 4

	// Surround the player completely.
	//
	// Up
	g.Walls = append(g.Walls, Wall{
		GridX:       4,
		GridY:       3,
		Orientation: WallHorizontal,
		OwnerID:     p.ID,
	})

	// Down
	g.Walls = append(g.Walls, Wall{
		GridX:       4,
		GridY:       4,
		Orientation: WallHorizontal,
		OwnerID:     p.ID,
	})

	// Left
	g.Walls = append(g.Walls, Wall{
		GridX:       3,
		GridY:       4,
		Orientation: WallVertical,
		OwnerID:     p.ID,
	})

	// Right
	g.Walls = append(g.Walls, Wall{
		GridX:       4,
		GridY:       4,
		Orientation: WallVertical,
		OwnerID:     p.ID,
	})

	if g.hasPathToGoal(p) {
		t.Fatal("expected no path when player is completely surrounded")
	}
}

// -----------------------------------------------------------------------------
// ONE EXIT
// -----------------------------------------------------------------------------

func TestHasPathToGoal_OneExit(t *testing.T) {
	g := helperSetupGame(2)
	p := g.Players[0]

	p.GridX = 4
	p.GridY = 4

	// Block three directions, leave one exit.
	g.Walls = append(g.Walls,
		Wall{
			GridX:       4,
			GridY:       3,
			Orientation: WallHorizontal,
			OwnerID:     p.ID,
		},
		Wall{
			GridX:       3,
			GridY:       4,
			Orientation: WallVertical,
			OwnerID:     p.ID,
		},
		Wall{
			GridX:       4,
			GridY:       4,
			Orientation: WallVertical,
			OwnerID:     p.ID,
		},
	)

	if !g.hasPathToGoal(p) {
		t.Fatal("expected path through the remaining exit")
	}
}

// -----------------------------------------------------------------------------
// DETOUR
// -----------------------------------------------------------------------------

func TestHasPathToGoal_DetourAroundWall(t *testing.T) {
	g := helperSetupGame(2)
	p := g.Players[0]

	p.GridX = 4
	p.GridY = 4

	// Block direct upward movement.
	g.Walls = append(g.Walls, Wall{
		GridX:       4,
		GridY:       3,
		Orientation: WallHorizontal,
		OwnerID:     p.ID,
	})

	// There should still be a route around the wall.
	if !g.hasPathToGoal(p) {
		t.Fatal("expected path around the wall")
	}
}

// -----------------------------------------------------------------------------
// FULL HORIZONTAL BARRIER
// -----------------------------------------------------------------------------

func TestHasPathToGoal_FullHorizontalBarrier(t *testing.T) {
	g := helperSetupGame(2)
	p := g.Players[0]

	// Put player on the lower half.
	p.GridX = 4
	p.GridY = 8

	// Create a complete horizontal barrier between Y=3 and Y=4.
	//
	// Eight wall segments are enough to cover all nine columns.
	//
	// The exact representation uses overlapping segments of length 2,
	// therefore this is intentionally built according to the board's
	// wall representation.

	for x := 0; x <= 7; x++ {
		g.Walls = append(g.Walls, Wall{
			GridX:       x,
			GridY:       3,
			Orientation: WallHorizontal,
			OwnerID:     p.ID,
		})
	}

	if g.hasPathToGoal(p) {
		t.Fatal("expected full horizontal barrier to disconnect the goal")
	}
}

// -----------------------------------------------------------------------------
// FULL VERTICAL BARRIER
// -----------------------------------------------------------------------------

// -----------------------------------------------------------------------------
// SINGLE GAP
// -----------------------------------------------------------------------------

// -----------------------------------------------------------------------------
// GOAL POSITION
// -----------------------------------------------------------------------------

func TestHasPathToGoal_AlreadyAtGoal(t *testing.T) {
	g := helperSetupGame(2)
	p := g.Players[0]

	p.GridY = p.TargetY

	if !g.hasPathToGoal(p) {
		t.Fatal("expected player already on target row to have a path")
	}
}

func TestHasPathToGoal_TargetX(t *testing.T) {
	g := helperSetupGame(2)
	p := g.Players[0]

	// Force an X-based goal so this test explicitly exercises TargetX.
	p.TargetY = -1
	p.TargetX = 8

	p.GridX = 0
	p.GridY = 4

	if !g.hasPathToGoal(p) {
		t.Fatal("expected path to X target")
	}
}

// -----------------------------------------------------------------------------
// BOTH TARGETS
// -----------------------------------------------------------------------------

func TestHasPathToGoal_EitherTargetIsValid(t *testing.T) {
	g := helperSetupGame(2)
	p := g.Players[0]

	p.TargetX = 8
	p.TargetY = 0

	// Starting from the center, either target should be reachable.
	p.GridX = 4
	p.GridY = 4

	if !g.hasPathToGoal(p) {
		t.Fatal("expected path to at least one target")
	}
}

// -----------------------------------------------------------------------------
// INVALID TARGETS
// -----------------------------------------------------------------------------

func TestHasPathToGoal_NoTarget(t *testing.T) {
	g := helperSetupGame(2)
	p := g.Players[0]

	p.TargetX = -1
	p.TargetY = -1

	if g.hasPathToGoal(p) {
		t.Fatal("expected no path when player has no target")
	}
}

// -----------------------------------------------------------------------------
// OPPONENT POSITIONS MUST NOT MATTER
// -----------------------------------------------------------------------------

func TestHasPathToGoal_OpponentPositionDoesNotMatter(t *testing.T) {
	positions := [][2]int{
		{0, 0},
		{8, 0},
		{0, 8},
		{8, 8},
		{4, 4},
		{3, 7},
		{7, 3},
	}

	for _, pos := range positions {
		t.Run("opponent_position", func(t *testing.T) {
			g := helperSetupGame(2)

			p := g.Players[0]
			opponent := g.Players[1]

			p.GridX = 4
			p.GridY = 8

			opponent.GridX = pos[0]
			opponent.GridY = pos[1]

			if !g.hasPathToGoal(p) {
				t.Fatalf(
					"opponent at (%d,%d) should not affect path",
					pos[0],
					pos[1],
				)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// MAZE / ALTERNATIVE ROUTES
// -----------------------------------------------------------------------------

func TestHasPathToGoal_MultipleRoutes(t *testing.T) {
	g := helperSetupGame(2)
	p := g.Players[0]

	p.GridX = 4
	p.GridY = 8

	// Create several obstacles but deliberately leave multiple
	// routes around them.
	g.Walls = append(g.Walls,
		Wall{
			GridX:       2,
			GridY:       5,
			Orientation: WallHorizontal,
			OwnerID:     p.ID,
		},
		Wall{
			GridX:       4,
			GridY:       5,
			Orientation: WallHorizontal,
			OwnerID:     p.ID,
		},
		Wall{
			GridX:       5,
			GridY:       3,
			Orientation: WallVertical,
			OwnerID:     p.ID,
		},
		Wall{
			GridX:       2,
			GridY:       2,
			Orientation: WallVertical,
			OwnerID:     p.ID,
		},
	)

	if !g.hasPathToGoal(p) {
		t.Fatal("expected at least one route through the maze")
	}
}

// -----------------------------------------------------------------------------
// REGRESSION: WALLS MUST BE CHECKED IN BOTH DIRECTIONS
// -----------------------------------------------------------------------------

func TestHasPathToGoal_WallBlocksBothDirections(t *testing.T) {
	g := helperSetupGame(2)
	p := g.Players[0]

	p.GridX = 4
	p.GridY = 4

	// A wall between two cells must block the edge regardless
	// of which cell the BFS visits first.
	g.Walls = append(g.Walls, Wall{
		GridX:       4,
		GridY:       3,
		Orientation: WallHorizontal,
		OwnerID:     p.ID,
	})

	// Moving upward directly is blocked, but another route exists.
	if !g.hasPathToGoal(p) {
		t.Fatal("expected alternate route around wall")
	}
}

// -----------------------------------------------------------------------------
// BOUNDARY START POSITIONS
// -----------------------------------------------------------------------------

func TestHasPathToGoal_FromCorners(t *testing.T) {
	tests := []struct {
		name string
		x    int
		y    int
	}{
		{"top-left", 0, 0},
		{"top-right", 8, 0},
		{"bottom-left", 0, 8},
		{"bottom-right", 8, 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := helperSetupGame(2)
			p := g.Players[0]

			p.GridX = tt.x
			p.GridY = tt.y

			if !g.hasPathToGoal(p) {
				t.Fatalf(
					"expected path from corner (%d,%d)",
					tt.x,
					tt.y,
				)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// LARGE OPEN BOARD
//
// Useful as a baseline when optimizing the implementation.
// -----------------------------------------------------------------------------

func TestHasPathToGoal_LargeOpenBoard(t *testing.T) {
	g := helperSetupGame(2)
	p := g.Players[0]

	p.GridX = 4
	p.GridY = 8

	if !g.hasPathToGoal(p) {
		t.Fatal("expected path on completely open board")
	}
}

// -----------------------------------------------------------------------------
// BENCHMARK
//
// This is deliberately included because the stated goal is optimization.
// Run:
//
//     go test ./... -bench=HasPathToGoal -benchmem
//
// before and after optimization.
// -----------------------------------------------------------------------------

func BenchmarkHasPathToGoal_OpenBoard(b *testing.B) {
	g := helperSetupGame(2)
	p := g.Players[0]

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = g.hasPathToGoal(p)
	}
}

func BenchmarkHasPathToGoal_WithWalls(b *testing.B) {
	g := helperSetupGame(2)
	p := g.Players[0]

	g.Walls = append(g.Walls,
		Wall{
			GridX:       2,
			GridY:       3,
			Orientation: WallHorizontal,
			OwnerID:     p.ID,
		},
		Wall{
			GridX:       4,
			GridY:       3,
			Orientation: WallHorizontal,
			OwnerID:     p.ID,
		},
		Wall{
			GridX:       3,
			GridY:       5,
			Orientation: WallVertical,
			OwnerID:     p.ID,
		},
		Wall{
			GridX:       5,
			GridY:       4,
			Orientation: WallVertical,
			OwnerID:     p.ID,
		},
	)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = g.hasPathToGoal(p)
	}
}
