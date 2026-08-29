package bot

import (
	"quoridor/internal/domain"
)

// GreedyPathBot uses pure heuristic/greedy decision making based on Dijkstra/BFS pathfinding without Minimax search.
type GreedyPathBot struct{}

// NewGreedyPathBot creates a new instance of the JavaScript-style Bot.
func NewGreedyPathBot() *GreedyPathBot {
	return &GreedyPathBot{}
}

// -----------------------------------------------------------------------------
// CONNECTIVITY GRID & BFS SHORTEST PATH
// -----------------------------------------------------------------------------

// JSWallGrid stores blocked movement directions for each cell.
// Directions: 0 = up (y-1), 1 = down (y+1), 2 = left (x-1), 3 = right (x+1).
type JSWallGrid [domain.BoardSize][domain.BoardSize][4]bool

var jsBfsDirs = [4][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}

func buildJSWallGrid(g *domain.Game) JSWallGrid {
	var grid JSWallGrid
	for _, w := range g.Walls {
		applyJSWallToGrid(&grid, w.GridX, w.GridY, w.Orientation, true)
	}
	return grid
}

func applyJSWallToGrid(grid *JSWallGrid, wx, wy int, orient domain.WallOrientation, blocked bool) {
	if orient == domain.WallHorizontal {
		grid[wx][wy][1] = blocked
		grid[wx][wy+1][0] = blocked
		grid[wx+1][wy][1] = blocked
		grid[wx+1][wy+1][0] = blocked
	} else {
		grid[wx][wy][3] = blocked
		grid[wx+1][wy][2] = blocked
		grid[wx][wy+1][3] = blocked
		grid[wx+1][wy+1][2] = blocked
	}
}

// getShortestPathLength calculates the exact number of steps to the target row/col using BFS.
func (b *GreedyPathBot) getShortestPathLength(grid *JSWallGrid, startX, startY, targetX, targetY int) int {
	if (targetX != -1 && startX == targetX) || (targetY != -1 && startY == targetY) {
		return 0
	}

	var visited [domain.BoardSize][domain.BoardSize]bool

	type node struct {
		x, y, dist int
	}

	var queue [domain.BoardSize * domain.BoardSize]node
	head, tail := 0, 0

	queue[tail] = node{startX, startY, 0}
	tail++
	visited[startX][startY] = true

	for head < tail {
		curr := queue[head]
		head++

		if (targetY != -1 && curr.y == targetY) || (targetX != -1 && curr.x == targetX) {
			return curr.dist
		}

		for d := 0; d < 4; d++ {
			if grid[curr.x][curr.y][d] {
				continue
			}

			nx := curr.x + jsBfsDirs[d][0]
			ny := curr.y + jsBfsDirs[d][1]

			if nx >= 0 && nx < domain.BoardSize && ny >= 0 && ny < domain.BoardSize {
				if !visited[nx][ny] {
					visited[nx][ny] = true
					queue[tail] = node{nx, ny, curr.dist + 1}
					tail++
				}
			}
		}
	}

	return 999 // Unreachable
}

// -----------------------------------------------------------------------------
// MAIN ACTION SELECTION LOGIC (JS BOT RULES)
// -----------------------------------------------------------------------------

// GetAction selects the move or wall placement following the JS bot's priority rules.
func (b *GreedyPathBot) GetAction(g *domain.Game, p *domain.Player) domain.PlayerAction {
	wallGrid := buildJSWallGrid(g)

	// Get closest opponent
	var opponent *domain.Player
	for _, pl := range g.Players {
		if pl.ID != p.ID {
			opponent = pl
			break
		}
	}

	if opponent == nil {
		return domain.PlayerAction{Type: domain.ActionNone}
	}

	// 1. Calculate current shortest path distances
	pathAI := b.getShortestPathLength(&wallGrid, p.GridX, p.GridY, p.TargetX, p.TargetY)
	pathP := b.getShortestPathLength(&wallGrid, opponent.GridX, opponent.GridY, opponent.TargetX, opponent.TargetY)

	// Get legal moves for pawn
	var validMoveBuf [8]domain.Move
	validMoveCount := g.GetValidMovesInto(p, &validMoveBuf)
	if validMoveCount == 0 {
		return domain.PlayerAction{Type: domain.ActionNone}
	}
	validMoves := validMoveBuf[:validMoveCount]

	// Find the best move that minimizes AI distance to target
	bestMoveAction := domain.PlayerAction{
		Type:    domain.ActionMove,
		TargetX: validMoves[0].X,
		TargetY: validMoves[0].Y,
	}
	minMoveDist := 999

	for _, m := range validMoves {
		dist := b.getShortestPathLength(&wallGrid, m.X, m.Y, p.TargetX, p.TargetY)
		if dist < minMoveDist {
			minMoveDist = dist
			bestMoveAction = domain.PlayerAction{
				Type:    domain.ActionMove,
				TargetX: m.X,
				TargetY: m.Y,
			}
		}
	}

	// RULE 1: End Game / Sprint to Goal
	// If AI is 3 steps or fewer from winning, always move along the shortest path.
	if pathAI <= 3 {
		return bestMoveAction
	}

	// RULE 2: Defensive Wall Placement
	// If human is closer to winning (pathP < pathAI) and AI has walls left, try placing a defensive wall.
	if pathP < pathAI && p.WallsLeft > 0 {
		bestWallAction, foundWall := b.findBestDefensiveWall(g, p, opponent, &wallGrid, pathP)
		if foundWall {
			return bestWallAction
		}
	}

	// RULE 3: Default Move
	// If no defensive wall was needed/found or AI is leading, advance pawn along shortest path.
	return bestMoveAction
}

// findBestDefensiveWall finds a valid wall placement that maximizes opponent distance without blocking paths completely.
func (b *GreedyPathBot) findBestDefensiveWall(
	g *domain.Game,
	p *domain.Player,
	opponent *domain.Player,
	grid *JSWallGrid,
	currentOppDist int,
) (domain.PlayerAction, bool) {

	bestWall := domain.PlayerAction{}
	maxOppDist := currentOppDist
	found := false

	orientations := []domain.WallOrientation{domain.WallHorizontal, domain.WallVertical}

	// Evaluate all legal wall positions
	for wx := 0; wx < domain.BoardSize-1; wx++ {
		for wy := 0; wy < domain.BoardSize-1; wy++ {
			for _, orient := range orientations {
				if !g.CanPlaceWall(wx, wy, orient, p) {
					continue
				}

				// Apply wall temporarily
				applyJSWallToGrid(grid, wx, wy, orient, true)

				newAIPath := b.getShortestPathLength(grid, p.GridX, p.GridY, p.TargetX, p.TargetY)
				newOppPath := b.getShortestPathLength(grid, opponent.GridX, opponent.GridY, opponent.TargetX, opponent.TargetY)

				// Ensure neither player is completely trapped
				if newAIPath < 999 && newOppPath < 999 {
					// Select wall that causes the largest delay to opponent
					if newOppPath > maxOppDist {
						maxOppDist = newOppPath
						bestWall = domain.PlayerAction{
							Type:            domain.ActionPlaceWall,
							TargetX:         wx,
							TargetY:         wy,
							WallOrientation: orient,
						}
						found = true
					}
				}

				// Revert wall
				applyJSWallToGrid(grid, wx, wy, orient, false)
			}
		}
	}

	return bestWall, found
}
