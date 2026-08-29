package domain_test

import (
	"image/color"
	"reflect"
	"sort"
	"testing"

	"quoridor/internal/domain"
)

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

// assertMoves compares moves as sets, ignoring order.
func assertMoves(t *testing.T, got []domain.Move, expected [][]int) {
	t.Helper()

	converted := make([][]int, 0, len(got))
	for _, move := range got {
		converted = append(converted, []int{move.X, move.Y})
	}
	converted = normalizeMoves(converted)
	expected = normalizeMoves(expected)

	if !reflect.DeepEqual(converted, expected) {
		t.Fatalf(
			"unexpected valid moves\nexpected: %v\ngot:      %v",
			expected,
			converted,
		)
	}
}

func helperSetupGame(numPlayers int) *domain.Game {
	g := domain.NewGame()
	configs := []domain.PlayerConfig{
		{ID: 1, Color: color.White},
		{ID: 2, Color: color.Black},
	}
	if numPlayers == 4 {
		configs = append(configs,
			domain.PlayerConfig{ID: 3, Color: color.White},
			domain.PlayerConfig{ID: 4, Color: color.White},
		)
	}
	g.InitBoard(configs)
	return g
}

func getValidMoves(g *domain.Game, p *domain.Player) []domain.Move {
	var buf [8]domain.Move
	count := g.GetValidMovesInto(p, &buf)
	return buf[:count]
}

func assertMovesEqual(t *testing.T, got, expected []domain.Move) {
	t.Helper()

	expectedInts := make([][]int, 0, len(expected))
	for _, move := range expected {
		expectedInts = append(expectedInts, []int{move.X, move.Y})
	}

	assertMoves(t, got, expectedInts)
}

func normalizeMoves(moves [][]int) [][]int {
	result := make([][]int, 0, len(moves))

	for _, move := range moves {
		if len(move) != 2 {
			result = append(result, move)
			continue
		}

		result = append(result, []int{move[0], move[1]})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i][0] != result[j][0] {
			return result[i][0] < result[j][0]
		}

		return result[i][1] < result[j][1]
	})

	return result
}

func containsMove(moves []domain.Move, x, y int) bool {
	for _, move := range moves {
		if move.X == x && move.Y == y {
			return true
		}
	}

	return false
}

func assertContainsMove(t *testing.T, moves []domain.Move, x, y int) {
	t.Helper()

	if !containsMove(moves, x, y) {
		t.Errorf("expected move (%d,%d), got %v", x, y, moves)
	}
}

func assertDoesNotContainMove(t *testing.T, moves []domain.Move, x, y int) {
	t.Helper()

	if containsMove(moves, x, y) {
		t.Errorf("did not expect move (%d,%d), got %v", x, y, moves)
	}
}

func TestCanPlaceWall_OverlapValidation(t *testing.T) {
	g := helperSetupGame(2)
	p1 := g.Players[0]

	// Place initial horizontal wall at (2, 2)
	placed := g.PlaceWall(2, 2, domain.WallHorizontal, p1)
	if !placed {
		t.Fatalf("failed to place initial valid wall")
	}

	// 1. Exact overlap test
	if g.CanPlaceWall(2, 2, domain.WallHorizontal, p1) {
		t.Errorf("expected duplicate horizontal wall at (2,2) to be invalid")
	}

	// 2. Overlapping horizontal segment (x-1 overlap)
	if g.CanPlaceWall(1, 2, domain.WallHorizontal, p1) {
		t.Errorf("expected overlapping horizontal segment at (1,2) to be invalid")
	}

	// 3. Crossing vertical wall test at same intersection
	if g.CanPlaceWall(2, 2, domain.WallVertical, p1) {
		t.Errorf("expected crossing vertical wall at (2,2) to be invalid")
	}

	// 4. Out of bounds test
	if g.CanPlaceWall(-1, 0, domain.WallHorizontal, p1) || g.CanPlaceWall(8, 2, domain.WallHorizontal, p1) {
		t.Errorf("expected out of bounds wall placement to be invalid")
	}
}

func TestCanPlaceWall_PathBlockingValidation(t *testing.T) {
	g := helperSetupGame(2)
	p1 := g.Players[0]
	p2 := g.Players[1]

	// Move P2 into corner (0, 0)
	p2.GridX = 0
	p2.GridY = 0

	// Block P2 exit right: vertical wall at (0, 0)
	g.Walls = append(g.Walls, domain.Wall{
		GridX:       0,
		GridY:       0,
		Orientation: domain.WallVertical,
		OwnerID:     1,
	})

	// Placing horizontal wall at (0, 0) would completely enclose P2 at (0,0)
	isLegal := g.CanPlaceWall(0, 0, domain.WallHorizontal, p1)

	if isLegal {
		t.Errorf("expected wall placement to be invalid as it completely traps player 2")
	}
}

func TestCanPlaceWall_ZeroWallsLeft(t *testing.T) {
	g := helperSetupGame(2)
	p1 := g.Players[0]
	p1.WallsLeft = 0

	if g.CanPlaceWall(3, 3, domain.WallHorizontal, p1) {
		t.Errorf("expected wall placement to be invalid when player has 0 walls left")
	}
}

func TestNewGame_InitialState(t *testing.T) {
	g := domain.NewGame()

	if g.State != domain.StateMenu {
		t.Fatalf("expected initial state to be StateMenu, got %v", g.State)
	}
	if len(g.Players) != 0 {
		t.Fatalf("expected no players in new game, got %d", len(g.Players))
	}
	if g.TurnIndex != 0 {
		t.Fatalf("expected TurnIndex 0, got %d", g.TurnIndex)
	}
	if len(g.Walls) != 0 {
		t.Fatalf("expected no walls in new game, got %d", len(g.Walls))
	}
}

func TestInitBoard_TwoPlayerSetup(t *testing.T) {
	g := domain.NewGame()
	configs := []domain.PlayerConfig{
		{ID: 1, Color: color.White},
		{ID: 2, Color: color.Black},
	}
	g.InitBoard(configs)

	if len(g.Players) != 2 {
		t.Fatalf("expected 2 players, got %d", len(g.Players))
	}

	p1 := g.Players[0]
	p2 := g.Players[1]

	if p1.GridX != 4 || p1.GridY != 8 || p1.TargetY != 0 {
		t.Fatalf("unexpected starting position or target for P1: %+v", p1)
	}
	if p2.GridX != 4 || p2.GridY != 0 || p2.TargetY != 8 {
		t.Fatalf("unexpected starting position or target for P2: %+v", p2)
	}
	if p1.WallsLeft != 10 || p2.WallsLeft != 10 {
		t.Fatalf("expected 10 walls for each player, got %d and %d", p1.WallsLeft, p2.WallsLeft)
	}
}

func TestInitBoard_FourPlayerSetup(t *testing.T) {
	g := domain.NewGame()
	configs := []domain.PlayerConfig{
		{ID: 1, Color: color.White},
		{ID: 2, Color: color.Black},
		{ID: 3, Color: color.White},
		{ID: 4, Color: color.Black},
	}
	g.InitBoard(configs)

	if len(g.Players) != 4 {
		t.Fatalf("expected 4 players, got %d", len(g.Players))
	}

	p1 := g.Players[0]
	p2 := g.Players[1]
	p3 := g.Players[2]
	p4 := g.Players[3]

	if p1.GridX != 4 || p1.GridY != 8 || p1.TargetY != 0 {
		t.Fatalf("unexpected starting position or target for P1: %+v", p1)
	}
	if p2.GridX != 4 || p2.GridY != 0 || p2.TargetY != 8 {
		t.Fatalf("unexpected starting position or target for P2: %+v", p2)
	}
	if p3.GridX != 0 || p3.GridY != 4 || p3.TargetX != 8 {
		t.Fatalf("unexpected starting position or target for P3: %+v", p3)
	}
	if p4.GridX != 8 || p4.GridY != 4 || p4.TargetX != 0 {
		t.Fatalf("unexpected starting position or target for P4: %+v", p4)
	}
	if p1.WallsLeft != 5 || p2.WallsLeft != 5 || p3.WallsLeft != 5 || p4.WallsLeft != 5 {
		t.Fatalf("expected 5 walls for each player in four-player setup, got %d, %d, %d, %d",
			p1.WallsLeft, p2.WallsLeft, p3.WallsLeft, p4.WallsLeft)
	}
}

func TestGetValidMoves_Player3AndPlayer4PreferHorizontalMoves(t *testing.T) {
	g := helperSetupGame(4)
	p3 := g.Players[2]
	p4 := g.Players[3]

	moves3 := getValidMoves(g, p3)
	if !containsMove(moves3, 1, 4) {
		t.Fatalf("expected player 3 to prefer east move from start, got %v", moves3)
	}

	moves4 := getValidMoves(g, p4)
	if !containsMove(moves4, 7, 4) {
		t.Fatalf("expected player 4 to prefer west move from start, got %v", moves4)
	}
}

func TestActivePlayerAndNextTurn(t *testing.T) {
	g := helperSetupGame(2)

	first := g.ActivePlayer()
	if first == nil || first.ID != 1 {
		t.Fatalf("expected active player ID 1, got %v", first)
	}

	g.NextTurn()
	second := g.ActivePlayer()
	if second == nil || second.ID != 2 {
		t.Fatalf("expected active player ID 2 after next turn, got %v", second)
	}

	g.NextTurn()
	wrapped := g.ActivePlayer()
	if wrapped == nil || wrapped.ID != 1 {
		t.Fatalf("expected turn to wrap back to player 1, got %v", wrapped)
	}
}

func TestActivePlayer_NoPlayers(t *testing.T) {
	g := domain.NewGame()

	if got := g.ActivePlayer(); got != nil {
		t.Fatalf("expected ActivePlayer to return nil when no players are present, got %v", got)
	}
}

func TestCheckWin_RowGoal(t *testing.T) {
	g := helperSetupGame(2)
	p := g.Players[0]

	p.GridY = 0
	if !g.CheckWin(p) {
		t.Fatal("expected CheckWin to return true when player reaches target row")
	}
}

func TestCheckWin_ColumnGoal(t *testing.T) {
	g := helperSetupGame(2)
	p := g.Players[0]

	p.TargetY = -1
	p.TargetX = 8
	p.GridX = 8

	if !g.CheckWin(p) {
		t.Fatal("expected CheckWin to return true when player reaches target column")
	}
}

func TestPlaceWall_Success(t *testing.T) {
	g := helperSetupGame(2)
	p := g.Players[0]
	initialWalls := p.WallsLeft

	if !g.PlaceWall(2, 2, domain.WallHorizontal, p) {
		t.Fatal("expected PlaceWall to succeed on empty board")
	}
	if len(g.Walls) != 1 {
		t.Fatalf("expected 1 wall after placement, got %d", len(g.Walls))
	}
	if p.WallsLeft != initialWalls-1 {
		t.Fatalf("expected WallsLeft decremented by 1, got %d", p.WallsLeft)
	}
}

func TestCanPlaceWall_ValidPlacement(t *testing.T) {
	g := helperSetupGame(2)
	p := g.Players[0]

	if !g.CanPlaceWall(3, 3, domain.WallHorizontal, p) {
		t.Fatal("expected CanPlaceWall to return true for a valid placement")
	}
}

type fakePlayerController struct{}

func (fakePlayerController) GetAction(g *domain.Game, p *domain.Player) domain.PlayerAction {
	return domain.PlayerAction{Type: domain.ActionNone}
}

func TestPlayerController_InterfaceCompliance(t *testing.T) {
	var _ domain.PlayerController = fakePlayerController{}
}

// -----------------------------------------------------------------------------
// BASIC MOVEMENT
// -----------------------------------------------------------------------------

func TestGetValidMoves_StandardMovement(t *testing.T) {
	g := helperSetupGame(2)
	p1 := g.Players[0]

	// P1 starts at (4,8).
	//
	// Valid moves:
	//   (4,7)
	//   (3,8)
	//   (5,8)
	//
	// South (4,9) is outside the board.
	expected := [][]int{
		{4, 7},
		{3, 8},
		{5, 8},
	}

	assertMoves(t, getValidMoves(g, p1), expected)
}

func TestGetValidMoves_Corners(t *testing.T) {
	tests := []struct {
		name string
		x    int
		y    int
		want [][]int
	}{
		{
			name: "top left",
			x:    0,
			y:    0,
			want: [][]int{
				{1, 0},
				{0, 1},
			},
		},
		{
			name: "top right",
			x:    8,
			y:    0,
			want: [][]int{
				{7, 0},
				{8, 1},
			},
		},
		{
			name: "bottom left",
			x:    0,
			y:    8,
			want: [][]int{
				{1, 8},
				{0, 7},
			},
		},
		{
			name: "bottom right",
			x:    8,
			y:    8,
			want: [][]int{
				{7, 8},
				{8, 7},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := helperSetupGame(2)

			p1 := g.Players[0]
			p1.GridX = tt.x
			p1.GridY = tt.y

			assertMoves(t, getValidMoves(g, p1), tt.want)
		})
	}
}

func TestGetValidMoves_BoardEdges(t *testing.T) {
	tests := []struct {
		name string
		x    int
		y    int
		want [][]int
	}{
		{
			name: "top edge",
			x:    4,
			y:    0,
			want: [][]int{
				{3, 0},
				{5, 0},
				{4, 1},
			},
		},
		{
			name: "left edge",
			x:    0,
			y:    4,
			want: [][]int{
				{0, 3},
				{1, 4},
				{0, 5},
			},
		},
		{
			name: "right edge",
			x:    8,
			y:    4,
			want: [][]int{
				{7, 4},
				{8, 3},
				{8, 5},
			},
		},
		{
			name: "bottom edge",
			x:    4,
			y:    8,
			want: [][]int{
				{3, 8},
				{5, 8},
				{4, 7},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := helperSetupGame(2)

			p1 := g.Players[0]
			p1.GridX = tt.x
			p1.GridY = tt.y

			assertMoves(t, getValidMoves(g, p1), tt.want)
		})
	}
}

// -----------------------------------------------------------------------------
// HORIZONTAL WALLS
//
// WallHorizontal(x,y) blocks:
//
//   (x,y)   <-> (x,y+1)
//   (x+1,y) <-> (x+1,y+1)
// -----------------------------------------------------------------------------

func TestGetValidMoves_HorizontalWall(t *testing.T) {
	tests := []struct {
		name    string
		playerX int
		playerY int
		wallX   int
		wallY   int
		want    [][]int
	}{
		{
			name:    "blocks upward movement on left segment",
			playerX: 4,
			playerY: 4,
			wallX:   4,
			wallY:   3,
			want: [][]int{
				{3, 4},
				{5, 4},
				{4, 5},
			},
		},
		{
			name:    "blocks upward movement on right segment",
			playerX: 5,
			playerY: 4,
			wallX:   4,
			wallY:   3,
			want: [][]int{
				{4, 4},
				{6, 4},
				{5, 5},
			},
		},
		{
			name:    "blocks downward movement on left segment",
			playerX: 4,
			playerY: 3,
			wallX:   4,
			wallY:   3,
			want: [][]int{
				{3, 3},
				{5, 3},
				{4, 2},
			},
		},
		{
			name:    "blocks downward movement on right segment",
			playerX: 5,
			playerY: 3,
			wallX:   4,
			wallY:   3,
			want: [][]int{
				{4, 3},
				{6, 3},
				{5, 2},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := helperSetupGame(2)

			p1 := g.Players[0]
			p1.GridX = tt.playerX
			p1.GridY = tt.playerY

			g.Walls = append(g.Walls, domain.Wall{
				GridX:       tt.wallX,
				GridY:       tt.wallY,
				Orientation: domain.WallHorizontal,
				OwnerID:     p1.ID,
			})

			assertMoves(t, getValidMoves(g, p1), tt.want)
		})
	}
}

// -----------------------------------------------------------------------------
// VERTICAL WALLS
//
// WallVertical(x,y) blocks:
//
//   (x,y)   <-> (x+1,y)
//   (x,y+1) <-> (x+1,y+1)
// -----------------------------------------------------------------------------

func TestGetValidMoves_VerticalWall(t *testing.T) {
	tests := []struct {
		name    string
		playerX int
		playerY int
		wallX   int
		wallY   int
		want    [][]int
	}{
		{
			name:    "blocks left movement on upper segment",
			playerX: 4,
			playerY: 4,
			wallX:   3,
			wallY:   4,
			want: [][]int{
				{5, 4},
				{4, 3},
				{4, 5},
			},
		},
		{
			name:    "blocks right movement on upper segment",
			playerX: 3,
			playerY: 4,
			wallX:   3,
			wallY:   4,
			want: [][]int{
				{2, 4},
				{3, 3},
				{3, 5},
			},
		},
		{
			name:    "blocks left movement on lower segment",
			playerX: 4,
			playerY: 5,
			wallX:   3,
			wallY:   4,
			want: [][]int{
				{5, 5},
				{4, 4},
				{4, 6},
			},
		},
		{
			name:    "blocks right movement on lower segment",
			playerX: 3,
			playerY: 5,
			wallX:   3,
			wallY:   4,
			want: [][]int{
				{2, 5},
				{3, 4},
				{3, 6},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := helperSetupGame(2)

			p1 := g.Players[0]
			p1.GridX = tt.playerX
			p1.GridY = tt.playerY

			g.Walls = append(g.Walls, domain.Wall{
				GridX:       tt.wallX,
				GridY:       tt.wallY,
				Orientation: domain.WallVertical,
				OwnerID:     p1.ID,
			})

			assertMoves(t, getValidMoves(g, p1), tt.want)
		})
	}
}

// -----------------------------------------------------------------------------
// IRRELEVANT WALLS
// -----------------------------------------------------------------------------

func TestGetValidMoves_IrrelevantWallDoesNotChangeMoves(t *testing.T) {
	g := helperSetupGame(2)

	p1 := g.Players[0]
	p1.GridX = 4
	p1.GridY = 4

	before := getValidMoves(g, p1)

	g.Walls = append(g.Walls, domain.Wall{
		GridX:       0,
		GridY:       0,
		Orientation: domain.WallHorizontal,
		OwnerID:     p1.ID,
	})

	after := getValidMoves(g, p1)

	assertMovesEqual(t, after, before)
}

// -----------------------------------------------------------------------------
// MULTIPLE WALLS
// -----------------------------------------------------------------------------

func TestGetValidMoves_MultipleWalls(t *testing.T) {
	tests := []struct {
		name  string
		walls []domain.Wall
		want  [][]int
	}{
		{
			name: "two walls block two directions",
			walls: []domain.Wall{
				{
					GridX:       4,
					GridY:       3,
					Orientation: domain.WallHorizontal,
				},
				{
					GridX:       4,
					GridY:       4,
					Orientation: domain.WallVertical,
				},
			},
			want: [][]int{
				{3, 4},
				{4, 5},
			},
		},
		{
			name: "three walls leave one exit",
			walls: []domain.Wall{
				{
					GridX:       4,
					GridY:       3,
					Orientation: domain.WallHorizontal,
				},
				{
					GridX:       4,
					GridY:       4,
					Orientation: domain.WallHorizontal,
				},
				{
					GridX:       4,
					GridY:       4,
					Orientation: domain.WallVertical,
				},
			},
			want: [][]int{
				{3, 4},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := helperSetupGame(2)

			p1 := g.Players[0]
			p1.GridX = 4
			p1.GridY = 4

			g.Walls = append(g.Walls, tt.walls...)

			assertMoves(t, getValidMoves(g, p1), tt.want)
		})
	}
}

// -----------------------------------------------------------------------------
// STRAIGHT JUMP
// -----------------------------------------------------------------------------

func TestGetValidMoves_StraightJump(t *testing.T) {
	tests := []struct {
		name      string
		playerX   int
		playerY   int
		opponentX int
		opponentY int
		want      []int
	}{
		{
			name:      "jump upward",
			playerX:   4,
			playerY:   5,
			opponentX: 4,
			opponentY: 4,
			want:      []int{4, 3},
		},
		{
			name:      "jump downward",
			playerX:   4,
			playerY:   3,
			opponentX: 4,
			opponentY: 4,
			want:      []int{4, 5},
		},
		{
			name:      "jump left",
			playerX:   5,
			playerY:   4,
			opponentX: 4,
			opponentY: 4,
			want:      []int{3, 4},
		},
		{
			name:      "jump right",
			playerX:   3,
			playerY:   4,
			opponentX: 4,
			opponentY: 4,
			want:      []int{5, 4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := helperSetupGame(2)

			p1 := g.Players[0]
			p2 := g.Players[1]

			p1.GridX = tt.playerX
			p1.GridY = tt.playerY

			p2.GridX = tt.opponentX
			p2.GridY = tt.opponentY

			moves := getValidMoves(g, p1)

			assertContainsMove(t, moves, tt.want[0], tt.want[1])
		})
	}
}

// -----------------------------------------------------------------------------
// OPPONENT NOT ADJACENT
// -----------------------------------------------------------------------------

func TestGetValidMoves_NonAdjacentOpponentDoesNotBlock(t *testing.T) {
	g := helperSetupGame(2)

	p1 := g.Players[0]
	p2 := g.Players[1]

	p1.GridX = 4
	p1.GridY = 4

	p2.GridX = 4
	p2.GridY = 6

	expected := [][]int{
		{3, 4},
		{5, 4},
		{4, 3},
		{4, 5},
	}

	assertMoves(t, getValidMoves(g, p1), expected)
}

func TestGetValidMoves_DiagonalOpponentDoesNotBlock(t *testing.T) {
	g := helperSetupGame(2)

	p1 := g.Players[0]
	p2 := g.Players[1]

	p1.GridX = 4
	p1.GridY = 4

	p2.GridX = 5
	p2.GridY = 5

	expected := [][]int{
		{3, 4},
		{5, 4},
		{4, 3},
		{4, 5},
	}

	assertMoves(t, getValidMoves(g, p1), expected)
}

// -----------------------------------------------------------------------------
// DIAGONAL MOVEMENT WHEN STRAIGHT JUMP IS BLOCKED
// -----------------------------------------------------------------------------

func TestGetValidMoves_DiagonalJumpWhenBlocked(t *testing.T) {
	g := helperSetupGame(2)

	p1 := g.Players[0]
	p2 := g.Players[1]

	p1.GridX = 4
	p1.GridY = 5

	p2.GridX = 4
	p2.GridY = 4

	// Horizontal wall behind P2:
	//
	// (4,3) <-> (4,4)
	// (5,3) <-> (5,4)
	//
	// This prevents the straight jump from (4,5)
	// through P2 at (4,4) to (4,3).
	g.Walls = append(g.Walls, domain.Wall{
		GridX:       4,
		GridY:       3,
		Orientation: domain.WallHorizontal,
		OwnerID:     p1.ID,
	})

	moves := getValidMoves(g, p1)

	assertContainsMove(t, moves, 3, 4)
	assertContainsMove(t, moves, 5, 4)

	// Straight jump must NOT be available.
	assertDoesNotContainMove(t, moves, 4, 3)
}

func TestGetValidMoves_DiagonalJumpWhenBlocked_HorizontalApproach(t *testing.T) {
	g := helperSetupGame(2)

	p1 := g.Players[0]
	p2 := g.Players[1]

	p1.GridX = 5
	p1.GridY = 4

	p2.GridX = 4
	p2.GridY = 4

	// Vertical wall behind P2.
	g.Walls = append(g.Walls, domain.Wall{
		GridX:       3,
		GridY:       4,
		Orientation: domain.WallVertical,
		OwnerID:     p1.ID,
	})

	moves := getValidMoves(g, p1)

	assertContainsMove(t, moves, 4, 3)
	assertContainsMove(t, moves, 4, 5)

	assertDoesNotContainMove(t, moves, 3, 4)
}

// -----------------------------------------------------------------------------
// DIAGONALS AT BOARD EDGE
// -----------------------------------------------------------------------------

func TestGetValidMoves_OpponentAtBoardEdgeAllowsDiagonal(t *testing.T) {
	tests := []struct {
		name      string
		playerX   int
		playerY   int
		opponentX int
		opponentY int
		want      [][]int
	}{
		{
			name:      "opponent at top edge",
			playerX:   4,
			playerY:   1,
			opponentX: 4,
			opponentY: 0,
			want: [][]int{
				{3, 0},
				{3, 1},
				{4, 2},
				{5, 0},
				{5, 1},
			},
		},
		{
			name:      "opponent at bottom edge",
			playerX:   4,
			playerY:   7,
			opponentX: 4,
			opponentY: 8,
			want: [][]int{
				{3, 7},
				{3, 8},
				{4, 6},
				{5, 7},
				{5, 8},
			},
		},
		{
			name:      "opponent at left edge",
			playerX:   1,
			playerY:   4,
			opponentX: 0,
			opponentY: 4,
			want: [][]int{
				{0, 3},
				{0, 5},
				{1, 3},
				{1, 5},
				{2, 4},
			},
		},
		{
			name:      "opponent at right edge",
			playerX:   7,
			playerY:   4,
			opponentX: 8,
			opponentY: 4,
			want: [][]int{
				{6, 4},
				{7, 3},
				{7, 5},
				{8, 3},
				{8, 5},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := helperSetupGame(2)

			p1 := g.Players[0]
			p2 := g.Players[1]

			p1.GridX = tt.playerX
			p1.GridY = tt.playerY

			p2.GridX = tt.opponentX
			p2.GridY = tt.opponentY

			assertMoves(t, getValidMoves(g, p1), tt.want)
		})
	}
}

// -----------------------------------------------------------------------------
// DIAGONALS BLOCKED INDIVIDUALLY
// -----------------------------------------------------------------------------

func TestGetValidMoves_OneDiagonalBlocked(t *testing.T) {
	// Player:   (4,4)
	// Opponent: (4,5)
	//
	// Jump to (4,6) is blocked.
	// Diagonal destinations are (3,5) and (5,5).
	//
	// Block the left diagonal only.

	g := helperSetupGame(2)

	p1 := g.Players[0]
	p2 := g.Players[1]

	p1.GridX = 4
	p1.GridY = 4

	p2.GridX = 4
	p2.GridY = 5

	// Block jump.
	g.Walls = append(g.Walls, domain.Wall{
		GridX:       4,
		GridY:       5,
		Orientation: domain.WallHorizontal,
		OwnerID:     p1.ID,
	})

	// Block movement from opponent (4,5) to (3,5).
	g.Walls = append(g.Walls, domain.Wall{
		GridX:       3,
		GridY:       5,
		Orientation: domain.WallVertical,
		OwnerID:     p1.ID,
	})

	moves := getValidMoves(g, p1)

	assertDoesNotContainMove(t, moves, 3, 5)
	assertContainsMove(t, moves, 5, 5)
}

func TestGetValidMoves_BothDiagonalsBlocked(t *testing.T) {
	g := helperSetupGame(2)

	p1 := g.Players[0]
	p2 := g.Players[1]

	p1.GridX = 4
	p1.GridY = 4

	p2.GridX = 4
	p2.GridY = 5

	// Block straight jump.
	g.Walls = append(g.Walls,
		domain.Wall{
			GridX:       4,
			GridY:       5,
			Orientation: domain.WallHorizontal,
			OwnerID:     p1.ID,
		},

		// Block left diagonal.
		domain.Wall{
			GridX:       3,
			GridY:       5,
			Orientation: domain.WallVertical,
			OwnerID:     p1.ID,
		},

		// Block right diagonal.
		domain.Wall{
			GridX:       4,
			GridY:       5,
			Orientation: domain.WallVertical,
			OwnerID:     p1.ID,
		},
	)

	moves := getValidMoves(g, p1)

	assertDoesNotContainMove(t, moves, 4, 6)
	assertDoesNotContainMove(t, moves, 3, 5)
	assertDoesNotContainMove(t, moves, 5, 5)
}

// -----------------------------------------------------------------------------
// OPPONENT + IRRELEVANT WALL
// -----------------------------------------------------------------------------

func TestGetValidMoves_JumpUnaffectedByIrrelevantWall(t *testing.T) {
	g := helperSetupGame(2)

	p1 := g.Players[0]
	p2 := g.Players[1]

	p1.GridX = 4
	p1.GridY = 5

	p2.GridX = 4
	p2.GridY = 4

	g.Walls = append(g.Walls, domain.Wall{
		GridX:       0,
		GridY:       0,
		Orientation: domain.WallHorizontal,
		OwnerID:     p1.ID,
	})

	moves := getValidMoves(g, p1)

	assertContainsMove(t, moves, 4, 3)
}

// -----------------------------------------------------------------------------
// NO DUPLICATES / VALID BOARD POSITIONS
// -----------------------------------------------------------------------------

func TestGetValidMoves_Invariants_EmptyBoard(t *testing.T) {
	for y := 0; y < 9; y++ {
		for x := 0; x < 9; x++ {
			t.Run(
				"position",
				func(t *testing.T) {
					g := helperSetupGame(2)

					p1 := g.Players[0]
					p2 := g.Players[1]

					p1.GridX = x
					p1.GridY = y

					// Keep opponent away from P1.
					p2.GridX = 8
					p2.GridY = 8

					moves := getValidMoves(g, p1)
					seen := make(map[[2]int]bool)

					for _, move := range moves {
						if move.X < 0 || move.X > 8 || move.Y < 0 || move.Y > 8 {
							t.Errorf(
								"move outside board: player=(%d,%d), move=%v",
								x,
								y,
								move,
							)
						}

						if move.X == x && move.Y == y {
							t.Errorf(
								"current player position returned as valid move: (%d,%d)",
								x,
								y,
							)
						}

						key := [2]int{move.X, move.Y}

						if seen[key] {
							t.Errorf("duplicate move returned: %v", move)
						}

						seen[key] = true
					}
				},
			)
		}
	}
}

// -----------------------------------------------------------------------------
// WALL + BASIC MOVEMENT COMBINATIONS
// -----------------------------------------------------------------------------

func TestGetValidMoves_AllFourDirectionsBlocked(t *testing.T) {
	g := helperSetupGame(2)

	p1 := g.Players[0]
	p2 := g.Players[1]

	p1.GridX = 4
	p1.GridY = 4

	// Keep opponent away.
	p2.GridX = 8
	p2.GridY = 8

	g.Walls = append(g.Walls,
		// Up.
		domain.Wall{
			GridX:       4,
			GridY:       3,
			Orientation: domain.WallHorizontal,
			OwnerID:     p1.ID,
		},
		// Down.
		domain.Wall{
			GridX:       4,
			GridY:       4,
			Orientation: domain.WallHorizontal,
			OwnerID:     p1.ID,
		},
		// Left.
		domain.Wall{
			GridX:       3,
			GridY:       4,
			Orientation: domain.WallVertical,
			OwnerID:     p1.ID,
		},
		// Right.
		domain.Wall{
			GridX:       4,
			GridY:       4,
			Orientation: domain.WallVertical,
			OwnerID:     p1.ID,
		},
	)

	moves := getValidMoves(g, p1)

	if len(moves) != 0 {
		t.Fatalf("expected no valid moves, got %v", moves)
	}
}

// -----------------------------------------------------------------------------
// SYMMETRY / X-Y REGRESSION TESTS
// -----------------------------------------------------------------------------

func TestGetValidMoves_HorizontalAndVerticalSymmetry(t *testing.T) {
	// This test is deliberately simple.
	// It catches accidentally mixing X and Y when evaluating walls.

	g1 := helperSetupGame(2)
	p1 := g1.Players[0]
	p2 := g1.Players[1]

	p1.GridX = 4
	p1.GridY = 4
	p2.GridX = 8
	p2.GridY = 8

	g1.Walls = append(g1.Walls, domain.Wall{
		GridX:       4,
		GridY:       3,
		Orientation: domain.WallHorizontal,
		OwnerID:     p1.ID,
	})

	movesHorizontal := getValidMoves(g1, p1)

	// Equivalent rotated configuration:
	// horizontal wall -> vertical wall
	// up/down -> left/right
	g2 := helperSetupGame(2)
	q1 := g2.Players[0]
	q2 := g2.Players[1]

	q1.GridX = 4
	q1.GridY = 4
	q2.GridX = 8
	q2.GridY = 8

	g2.Walls = append(g2.Walls, domain.Wall{
		GridX:       3,
		GridY:       4,
		Orientation: domain.WallVertical,
		OwnerID:     q1.ID,
	})

	movesVertical := getValidMoves(g2, q1)

	// Both configurations should leave exactly 3 moves.
	if len(movesHorizontal) != 3 {
		t.Fatalf(
			"horizontal-wall configuration expected 3 moves, got %v",
			movesHorizontal,
		)
	}

	if len(movesVertical) != 3 {
		t.Fatalf(
			"vertical-wall configuration expected 3 moves, got %v",
			movesVertical,
		)
	}
}
