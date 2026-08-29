package domain

const BoardSize = 9

type GameState int

const (
	StateMenu GameState = iota
	StatePlaying
	StateWin
	StateFinished
)

type Game struct {
	State     GameState
	Players   []*Player
	TurnIndex int
	Turns     int
	Walls     []Wall
}

// NewGame creates a new Game instance in the menu state.
func NewGame() *Game {
	return &Game{
		State: StateMenu,
	}
}

// ActivePlayer returns the current player for this turn.
func (g *Game) ActivePlayer() *Player {
	if len(g.Players) == 0 {
		return nil
	}
	return g.Players[g.TurnIndex]
}

// NextTurn advances the active player index to the next player.
func (g *Game) NextTurn() {
	g.Turns++
	if len(g.Players) > 0 {
		g.TurnIndex = (g.TurnIndex + 1) % len(g.Players)
	}
}

// InitBoard initializes players, starting positions, and wall counts.
func (g *Game) InitBoard(configs []PlayerConfig) {
	g.Players = []*Player{}
	g.TurnIndex = 0
	g.Turns = 0

	wallsPerPlayer := 10
	if len(configs) == 4 {
		wallsPerPlayer = 5
	}
	g.Walls = []Wall{} // Reset active walls

	startPos := []struct{ x, y, tx, ty int }{
		{4, 8, -1, 0}, // P1: Bottom -> Target Y=0
		{4, 0, -1, 8}, // P2: Top    -> Target Y=8
		{0, 4, 8, -1}, // P3: Left   -> Target X=8
		{8, 4, 0, -1}, // P4: Right  -> Target X=0
	}

	for i, cfg := range configs {
		pos := startPos[i]
		g.Players = append(g.Players, &Player{
			ID:        cfg.ID,
			Color:     cfg.Color,
			GridX:     pos.x,
			GridY:     pos.y,
			TargetX:   pos.tx,
			TargetY:   pos.ty,
			WallsLeft: wallsPerPlayer,
		})
	}
}

func (g *Game) buildWallMaps() (horizontal [BoardSize - 1][BoardSize - 1]bool, vertical [BoardSize - 1][BoardSize - 1]bool) {
	for _, w := range g.Walls {
		if w.Orientation == WallHorizontal {
			horizontal[w.GridX][w.GridY] = true
		} else {
			vertical[w.GridX][w.GridY] = true
		}
	}
	return horizontal, vertical
}

func buildOccupancyMap(players []*Player) [BoardSize][BoardSize]bool {
	var occupied [BoardSize][BoardSize]bool
	for _, p := range players {
		occupied[p.GridX][p.GridY] = true
	}
	return occupied
}

func (g *Game) isWallBlockingMap(horizontal *[BoardSize - 1][BoardSize - 1]bool, vertical *[BoardSize - 1][BoardSize - 1]bool, x1, y1, x2, y2 int) bool {
	if x1 == x2 {
		if y1 == y2+1 || y1+1 == y2 {
			wy := y1
			if y2 < y1 {
				wy = y2
			}
			if x1 > 0 && horizontal[x1-1][wy] {
				return true
			}
			if x1 < BoardSize-1 && horizontal[x1][wy] {
				return true
			}
		}
	} else if y1 == y2 {
		if x1 == x2+1 || x1+1 == x2 {
			wx := x1
			if x2 < x1 {
				wx = x2
			}
			if y1 > 0 && vertical[wx][y1-1] {
				return true
			}
			if y1 < BoardSize-1 && vertical[wx][y1] {
				return true
			}
		}
	}
	return false
}

// Predefined diagonal directions outside the loop to avoid heap allocations
var (
	diagDirsVert  = [2][2]int{{-1, 0}, {1, 0}}
	diagDirsHoriz = [2][2]int{{0, -1}, {0, 1}}
)

type Move struct {
	X int
	Y int
}

// GetValidMovesInto writes the legal moves for the player into the provided output buffer.
// It returns the number of valid moves written. The buffer must have capacity for at
// most 8 moves.
func (g *Game) GetValidMovesInto(p *Player, out *[8]Move) int {
	horizontal, vertical := g.buildWallMaps()
	occupied := buildOccupancyMap(g.Players)
	dirs := g.getPreferredDirections(p)
	count := 0

	for _, d := range dirs {
		nx, ny := p.GridX+d[0], p.GridY+d[1]

		if nx >= 0 && nx < BoardSize && ny >= 0 && ny < BoardSize {
			if !g.isWallBlockingMap(&horizontal, &vertical, p.GridX, p.GridY, nx, ny) {

				if !occupied[nx][ny] {
					out[count] = Move{X: nx, Y: ny}
					count++
				} else {
					jumpX, jumpY := nx+d[0], ny+d[1]

					if jumpX >= 0 && jumpX < BoardSize && jumpY >= 0 && jumpY < BoardSize &&
						!g.isWallBlockingMap(&horizontal, &vertical, nx, ny, jumpX, jumpY) && !occupied[jumpX][jumpY] {

						out[count] = Move{X: jumpX, Y: jumpY}
						count++
					} else {
						var diagDirs *[2][2]int
						if d[0] == 0 {
							diagDirs = &diagDirsVert
						} else {
							diagDirs = &diagDirsHoriz
						}

						for _, dd := range diagDirs {
							diagX, diagY := nx+dd[0], ny+dd[1]

							if diagX >= 0 && diagX < BoardSize && diagY >= 0 && diagY < BoardSize {
								if !g.isWallBlockingMap(&horizontal, &vertical, nx, ny, diagX, diagY) && !occupied[diagX][diagY] {
									out[count] = Move{X: diagX, Y: diagY}
									count++
								}
							}
						}
					}
				}
			}
		}
	}

	return count
}

// getPreferredDirections returns a movement direction ordering toward the player's goal.
// It uses a fixed-size array to avoid heap allocations on the hot path.
func (g *Game) getPreferredDirections(p *Player) [4][2]int {
	if p.TargetY == 0 { // P1 (moving up)
		return [4][2]int{{0, -1}, {-1, 0}, {1, 0}, {0, 1}} // North, West, East, South
	} else if p.TargetY == 8 { // P2 (moving down)
		return [4][2]int{{0, 1}, {-1, 0}, {1, 0}, {0, -1}} // South, West, East, North
	} else if p.TargetX == 8 { // P3 (left to right)
		return [4][2]int{{1, 0}, {0, -1}, {0, 1}, {-1, 0}} // East, North, South, West
	}

	return [4][2]int{{-1, 0}, {0, -1}, {0, 1}, {1, 0}} // P4 (right to left): West, North, South, East
}

// CheckWin returns true if the player has reached their goal line.
func (g *Game) CheckWin(p *Player) bool {
	if p.TargetX != -1 && p.GridX == p.TargetX {
		return true
	}
	if p.TargetY != -1 && p.GridY == p.TargetY {
		return true
	}
	return false
}

// CanPlaceWall checks whether a wall placement is legal and preserves path connectivity for all players.
func (g *Game) CanPlaceWall(x, y int, orient WallOrientation, player *Player) bool {
	if player.WallsLeft <= 0 {
		return false
	}

	// Boundary check (intersections range from 0 to BoardSize-2)
	if x < 0 || x >= BoardSize-1 || y < 0 || y >= BoardSize-1 {
		return false
	}

	// Overlap check
	for _, w := range g.Walls {
		if w.GridX == x && w.GridY == y {
			return false // Exact same intersection
		}
		if w.Orientation == orient {
			if orient == WallHorizontal && w.GridY == y && (w.GridX == x-1 || w.GridX == x+1) {
				return false // Overlapping horizontal segment
			}
			if orient == WallVertical && w.GridX == x && (w.GridY == y-1 || w.GridY == y+1) {
				return false // Overlapping vertical segment
			}
		} else {
			if w.GridX == x && w.GridY == y {
				return false // Crossing middle intersection
			}
		}
	}

	// Temporarily place wall to check pathing
	tempWall := Wall{GridX: x, GridY: y, Orientation: orient, OwnerID: player.ID}
	g.Walls = append(g.Walls, tempWall)
	defer func() { g.Walls = g.Walls[:len(g.Walls)-1] }() // Remove temp wall

	// Ensure ALL players still have a valid path to their goal
	for _, p := range g.Players {
		if !g.hasPathToGoal(p) {
			return false
		}
	}

	return true
}

// PlaceWall attempts to place a wall for the player and decrements the player's remaining walls.
func (g *Game) PlaceWall(x, y int, orient WallOrientation, player *Player) bool {
	if !g.CanPlaceWall(x, y, orient, player) {
		return false
	}
	g.Walls = append(g.Walls, Wall{
		GridX:       x,
		GridY:       y,
		Orientation: orient,
		OwnerID:     player.ID,
	})
	player.WallsLeft--
	return true
}

// isWallBlocking returns whether a wall blocks direct movement between two adjacent cells.
func (g *Game) isWallBlocking(x1, y1, x2, y2 int) bool {
	for _, w := range g.Walls {
		if w.Orientation == WallHorizontal {
			// Horizontal wall blocks vertical movement between y1 and y2
			if (y1 == w.GridY && y2 == w.GridY+1) || (y1 == w.GridY+1 && y2 == w.GridY) {
				if x1 == w.GridX || x1 == w.GridX+1 {
					if x1 == x2 {
						return true
					}
				}
			}
		} else {
			// Vertical wall blocks horizontal movement between x1 and x2
			if (x1 == w.GridX && x2 == w.GridX+1) || (x1 == w.GridX+1 && x2 == w.GridX) {
				if y1 == w.GridY || y1 == w.GridY+1 {
					if y1 == y2 {
						return true
					}
				}
			}
		}
	}
	return false
}

// isOccupied returns true if any player occupies the specified cell.
func (g *Game) isOccupied(x, y int) bool {
	occupied := buildOccupancyMap(g.Players)
	return occupied[x][y]
}

var orthogonalDirs = [4][2]int{
	{0, -1}, // North
	{0, 1},  // South
	{-1, 0}, // West
	{1, 0},  // East
}

// hasPathToGoal checks whether a player still has a reachable path to their goal.
// It ignores opponents and only considers wall blocking.
func (g *Game) hasPathToGoal(p *Player) bool {
	// Static 9x9 visited grid on the stack avoids allocation.
	var visited [BoardSize][BoardSize]bool

	// Fixed-size queue for up to 81 board cells.
	var queue [BoardSize * BoardSize][2]int
	head, tail := 0, 0

	// Enqueue the player's starting position.
	queue[tail] = [2]int{p.GridX, p.GridY}
	tail++
	visited[p.GridX][p.GridY] = true

	for head < tail {
		curr := queue[head]
		head++

		cx, cy := curr[0], curr[1]

		// Check if the goal row or column has been reached.
		if (p.TargetY != -1 && cy == p.TargetY) || (p.TargetX != -1 && cx == p.TargetX) {
			return true
		}

		// Only explore the four orthogonal neighbors.
		for _, d := range orthogonalDirs {
			nx, ny := cx+d[0], cy+d[1]

			// 1. Validate position is inside the board.
			if nx >= 0 && nx < BoardSize && ny >= 0 && ny < BoardSize {
				// 2. Visit only unvisited cells that are not blocked by a wall.
				if !visited[nx][ny] && !g.isWallBlocking(cx, cy, nx, ny) {
					visited[nx][ny] = true
					queue[tail] = [2]int{nx, ny}
					tail++
				}
			}
		}
	}

	return false
}

// ShortestPathLen returns the length (in steps) of the player's shortest
// path to their goal, ignoring opponents (only walls block). Returns -1 if
// unreachable (should not happen if CanPlaceWall validated correctly).
func (g *Game) ShortestPathLen(p *Player) int {
	var visited [BoardSize][BoardSize]bool
	var dist [BoardSize * BoardSize]int
	var queue [BoardSize * BoardSize][2]int
	head, tail := 0, 0

	queue[tail] = [2]int{p.GridX, p.GridY}
	tail++
	visited[p.GridX][p.GridY] = true

	for head < tail {
		curr := queue[head]
		d := dist[head]
		head++
		cx, cy := curr[0], curr[1]

		if (p.TargetY != -1 && cy == p.TargetY) || (p.TargetX != -1 && cx == p.TargetX) {
			return d
		}

		for _, dir := range orthogonalDirs {
			nx, ny := cx+dir[0], cy+dir[1]
			if nx >= 0 && nx < BoardSize && ny >= 0 && ny < BoardSize {
				if !visited[nx][ny] && !g.isWallBlocking(cx, cy, nx, ny) {
					visited[nx][ny] = true
					dist[tail] = d + 1
					queue[tail] = [2]int{nx, ny}
					tail++
				}
			}
		}
	}
	return -1
}
