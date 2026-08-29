package bot

import (
	"math"
	"sort"
	"sync"
	"time"

	"quoridor/internal/domain"
)

// BotWeights contains the coefficients used by the heuristic evaluation.
type BotWeights struct {
	MyDistanceWeight    float64 `json:"my_distance_weight"`
	RivalDistanceWeight float64 `json:"rival_distance_weight"`
	WallReserveWeight   float64 `json:"wall_reserve_weight"`
	CentralityWeight    float64 `json:"centrality_weight"`
	JumpConcededPenalty float64 `json:"jump_conceded_penalty"`
	FunnelingBonus      float64 `json:"funneling_bonus"`
}

// BotConfig defines the heuristic profiles used by the bot.
type BotConfig struct {
	PanicThreshold int        `json:"panic_threshold"`
	NormalWeights  BotWeights `json:"normal_weights"`
	PanicWeights   BotWeights `json:"panic_weights"`
}

// DefaultBotConfig returns the baseline tuning used by the heuristic bot.
func DefaultBotConfig() BotConfig {
	return BotConfig{
		PanicThreshold: 1,
		NormalWeights: BotWeights{
			MyDistanceWeight:    -10.0,
			RivalDistanceWeight: 8.0,
			WallReserveWeight:   2.5,
			CentralityWeight:    1.0,
			JumpConcededPenalty: -30.0,
			FunnelingBonus:      15.0,
		},
		PanicWeights: BotWeights{
			MyDistanceWeight:    -4.0,
			RivalDistanceWeight: 20.0,
			WallReserveWeight:   0.5,
			CentralityWeight:    0.0,
			JumpConcededPenalty: -50.0,
			FunnelingBonus:      25.0,
		},
	}
}

// HeuristicBot chooses moves using a hand-tuned evaluation function.
type HeuristicBot struct {
	Config          BotConfig
	Delay           time.Duration
	PruneCandidates bool
	thinkingStart   time.Time
	isThinking      bool
	history         [][2]int
	mu              sync.Mutex
}

// NewHeuristicBot creates a bot instance with the supplied profile and delay.
func NewHeuristicBot(config BotConfig, delay time.Duration) *HeuristicBot {
	return &HeuristicBot{
		Config:          config,
		Delay:           delay,
		PruneCandidates: true,
		history:         make([][2]int, 0, 6),
	}
}

// NewHeuristicBotWithPruning creates a bot similar to NewHeuristicBot but lets callers
// explicitly enable or disable the candidate-pruning optimization used to compare
// the full-evaluation path against the faster filtered path.
func NewHeuristicBotWithPruning(config BotConfig, delay time.Duration, pruneCandidates bool) *HeuristicBot {
	bot := NewHeuristicBot(config, delay)
	bot.PruneCandidates = pruneCandidates
	return bot
}

func (b *HeuristicBot) recordPositionLocked(x, y int) {
	b.history = append(b.history, [2]int{x, y})
	if len(b.history) > 6 {
		b.history = b.history[1:]
	}
}

func (b *HeuristicBot) recordPosition(x, y int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.recordPositionLocked(x, y)
}

func (b *HeuristicBot) getRepetitionPenaltyLocked(x, y int) float64 {
	penalty := 0.0
	for i, pos := range b.history {
		if pos[0] == x && pos[1] == y {
			penalty -= float64(i+1) * 4.0
		}
	}
	return penalty
}

func (b *HeuristicBot) getRepetitionPenalty(x, y int) float64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.getRepetitionPenaltyLocked(x, y)
}

// -----------------------------------------------------------------------------
// STRUCTURES AND CONNECTIVITY GRID (BFS OPTIMIZATION)
// -----------------------------------------------------------------------------

// WallGrid stores the blocked directions for each square.
// Directions: 0 = up (y-1), 1 = down (y+1), 2 = left (x-1), 3 = right (x+1).
type WallGrid [domain.BoardSize][domain.BoardSize][4]bool

var bfsDirs = [4][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}

func buildWallGrid(g *domain.Game) WallGrid {
	var grid WallGrid
	for _, w := range g.Walls {
		applyWallToGrid(&grid, w.GridX, w.GridY, w.Orientation, true)
	}
	return grid
}

func applyWallToGrid(grid *WallGrid, wx, wy int, orient domain.WallOrientation, blocked bool) {
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

// getFastShortestPath computes the shortest path using the wall grid and a BFS queue.
func (b *HeuristicBot) getFastShortestPath(grid *WallGrid, startX, startY, targetX, targetY int) int {
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

			nx := curr.x + bfsDirs[d][0]
			ny := curr.y + bfsDirs[d][1]

			if nx >= 0 && nx < domain.BoardSize && ny >= 0 && ny < domain.BoardSize {
				if !visited[nx][ny] {
					visited[nx][ny] = true
					queue[tail] = node{nx, ny, curr.dist + 1}
					tail++
				}
			}
		}
	}

	return 999
}

// -----------------------------------------------------------------------------
// MAIN LOGIC AND ACTION SELECTION
// -----------------------------------------------------------------------------

// GetAction chooses the best move or wall placement for the provided player.
func (b *HeuristicBot) GetAction(g *domain.Game, p *domain.Player) domain.PlayerAction {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.Delay != 0 {
		if !b.isThinking {
			b.isThinking = true
			b.thinkingStart = time.Now()
			return domain.PlayerAction{Type: domain.ActionNone}
		}

		if time.Since(b.thinkingStart) < b.Delay {
			return domain.PlayerAction{Type: domain.ActionNone}
		}
	}

	b.isThinking = false
	return b.selectBestActionLocked(g, p)
}

func (b *HeuristicBot) selectBestAction(g *domain.Game, p *domain.Player) domain.PlayerAction {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.selectBestActionLocked(g, p)
}

func (b *HeuristicBot) fallbackLegalAction(g *domain.Game, p *domain.Player) domain.PlayerAction {
	var moveBuf [8]domain.Move
	moveCount := g.GetValidMovesInto(p, &moveBuf)
	if moveCount > 0 {
		m := moveBuf[0]
		return domain.PlayerAction{Type: domain.ActionMove, TargetX: m.X, TargetY: m.Y}
	}

	if p.WallsLeft > 0 {
		for wx := 0; wx < domain.BoardSize-1; wx++ {
			for wy := 0; wy < domain.BoardSize-1; wy++ {
				for _, orient := range []domain.WallOrientation{domain.WallHorizontal, domain.WallVertical} {
					if g.CanPlaceWall(wx, wy, orient, p) {
						return domain.PlayerAction{
							Type:            domain.ActionPlaceWall,
							TargetX:         wx,
							TargetY:         wy,
							WallOrientation: orient,
						}
					}
				}
			}
		}
	}

	return domain.PlayerAction{Type: domain.ActionNone}
}

func (b *HeuristicBot) selectBestActionLocked(g *domain.Game, p *domain.Player) domain.PlayerAction {
	var validMoveBuf [8]domain.Move
	validMoveCount := g.GetValidMovesInto(p, &validMoveBuf)
	if validMoveCount == 0 {
		if wallFallback := b.fallbackLegalAction(g, p); wallFallback.Type != domain.ActionNone {
			return wallFallback
		}
		return domain.PlayerAction{Type: domain.ActionNone}
	}

	validMoves := validMoveBuf[:validMoveCount]

	// Safe fallback to avoid invalid states during evaluation.
	bestAction := domain.PlayerAction{
		Type:    domain.ActionMove,
		TargetX: validMoves[0].X,
		TargetY: validMoves[0].Y,
	}
	bestScore := -math.MaxFloat64

	wallGrid := buildWallGrid(g)
	opponent := b.getClosestOpponent(g, p, &wallGrid)
	if opponent == nil {
		if fallback := b.fallbackLegalAction(g, p); fallback.Type != domain.ActionNone {
			return fallback
		}
		return bestAction
	}

	currentMyDist := b.getFastShortestPath(&wallGrid, p.GridX, p.GridY, p.TargetX, p.TargetY)
	currentOppDist := b.getFastShortestPath(&wallGrid, opponent.GridX, opponent.GridY, opponent.TargetX, opponent.TargetY)

	weights := b.Config.NormalWeights
	if (currentMyDist - currentOppDist) >= b.Config.PanicThreshold {
		weights = b.Config.PanicWeights
	}

	var oppMoveBuf [8]domain.Move
	oppMoveCount := g.GetValidMovesInto(opponent, &oppMoveBuf)
	opponentMoves := oppMoveBuf[:oppMoveCount]

	// This pruning path is optional so we can compare filtered vs exhaustive selection in benchmarks.
	if !b.PruneCandidates {
		for _, m := range validMoves {
			oldX, oldY := p.GridX, p.GridY
			p.GridX, p.GridY = m.X, m.Y

			grantsJump := b.doesMoveConcedeForwardJumpWithMoves(g, p, opponent, &wallGrid, opponentMoves)
			score := b.evaluateWorstCaseOpponentResponse(g, p, opponent, weights, &wallGrid)

			if grantsJump {
				score += weights.JumpConcededPenalty
			}
			score += b.getRepetitionPenaltyLocked(m.X, m.Y)

			p.GridX, p.GridY = oldX, oldY

			if score > bestScore {
				bestScore = score
				bestAction = domain.PlayerAction{
					Type:    domain.ActionMove,
					TargetX: m.X,
					TargetY: m.Y,
				}
			}
		}
	} else {
		type moveCandidate struct {
			move       domain.Move
			quickScore float64
		}

		moveCandidates := make([]moveCandidate, 0, len(validMoves))
		for _, m := range validMoves {
			oldX, oldY := p.GridX, p.GridY
			p.GridX, p.GridY = m.X, m.Y

			myDistAfter := b.getFastShortestPath(&wallGrid, p.GridX, p.GridY, p.TargetX, p.TargetY)
			opponentDistAfter := b.getFastShortestPath(&wallGrid, opponent.GridX, opponent.GridY, opponent.TargetX, opponent.TargetY)
			grantsJump := b.doesMoveConcedeForwardJumpWithMoves(g, p, opponent, &wallGrid, opponentMoves)
			quickScore := b.evaluateStateWithDistances(float64(myDistAfter), float64(opponentDistAfter)-0.5, p, opponent, weights, &wallGrid)
			if grantsJump {
				quickScore += weights.JumpConcededPenalty
			}
			quickScore += b.getRepetitionPenaltyLocked(m.X, m.Y)

			p.GridX, p.GridY = oldX, oldY
			moveCandidates = append(moveCandidates, moveCandidate{move: m, quickScore: quickScore})
		}

		limit := len(moveCandidates)
		if limit > 3 {
			limit = 3
		}
		if limit > 0 {
			sort.Slice(moveCandidates, func(i, j int) bool {
				return moveCandidates[i].quickScore > moveCandidates[j].quickScore
			})
		}

		for i := 0; i < limit; i++ {
			m := moveCandidates[i].move
			oldX, oldY := p.GridX, p.GridY
			p.GridX, p.GridY = m.X, m.Y

			grantsJump := b.doesMoveConcedeForwardJumpWithMoves(g, p, opponent, &wallGrid, opponentMoves)
			score := b.evaluateWorstCaseOpponentResponse(g, p, opponent, weights, &wallGrid)

			if grantsJump {
				score += weights.JumpConcededPenalty
			}
			score += b.getRepetitionPenaltyLocked(m.X, m.Y)

			p.GridX, p.GridY = oldX, oldY

			if score > bestScore {
				bestScore = score
				bestAction = domain.PlayerAction{
					Type:    domain.ActionMove,
					TargetX: m.X,
					TargetY: m.Y,
				}
			}
		}
	}

	// 2. Evaluate wall placements with spatial filtering.
	if p.WallsLeft > 0 {
		orientations := []domain.WallOrientation{domain.WallHorizontal, domain.WallVertical}

		minX := minInt(p.GridX, opponent.GridX) - 2
		maxX := maxInt(p.GridX, opponent.GridX) + 2
		minY := minInt(p.GridY, opponent.GridY) - 2
		maxY := maxInt(p.GridY, opponent.GridY) + 2

		wallCandidates := make([]struct {
			wx, wy     int
			orient     domain.WallOrientation
			quickScore float64
		}, 0, 32)

		for wx := 0; wx < domain.BoardSize-1; wx++ {
			if wx < minX || wx > maxX {
				continue
			}
			for wy := 0; wy < domain.BoardSize-1; wy++ {
				if wy < minY || wy > maxY {
					continue
				}

				for _, orient := range orientations {
					if !g.CanPlaceWall(wx, wy, orient, p) {
						continue
					}

					applyWallToGrid(&wallGrid, wx, wy, orient, true)

					newOppDist := b.getFastShortestPath(&wallGrid, opponent.GridX, opponent.GridY, opponent.TargetX, opponent.TargetY)
					newMyDist := b.getFastShortestPath(&wallGrid, p.GridX, p.GridY, p.TargetX, p.TargetY)

					oppDelay := newOppDist - currentOppDist
					myDelay := newMyDist - currentMyDist
					isTacticalWall := (oppDelay >= 2 && myDelay <= 1) ||
						(oppDelay >= 1 && myDelay == 0 && currentOppDist <= 5)

					if isTacticalWall {
						quickScore := float64(oppDelay*3 - myDelay*2)
						if currentOppDist <= 5 {
							quickScore += 2.0
						}
						wallCandidates = append(wallCandidates, struct {
							wx, wy     int
							orient     domain.WallOrientation
							quickScore float64
						}{
							wx:         wx,
							wy:         wy,
							orient:     orient,
							quickScore: quickScore,
						})
					}

					applyWallToGrid(&wallGrid, wx, wy, orient, false)
				}
			}
		}

		if len(wallCandidates) > 0 {
			sort.Slice(wallCandidates, func(i, j int) bool {
				return wallCandidates[i].quickScore > wallCandidates[j].quickScore
			})
			limitWalls := len(wallCandidates)
			if b.PruneCandidates && limitWalls > 6 {
				limitWalls = 6
			}
			for i := 0; i < limitWalls; i++ {
				candidate := wallCandidates[i]
				applyWallToGrid(&wallGrid, candidate.wx, candidate.wy, candidate.orient, true)
				score := b.evaluateWorstCaseOpponentResponse(g, p, opponent, weights, &wallGrid)
				if score > bestScore {
					bestScore = score
					bestAction = domain.PlayerAction{
						Type:            domain.ActionPlaceWall,
						TargetX:         candidate.wx,
						TargetY:         candidate.wy,
						WallOrientation: candidate.orient,
					}
				}
				applyWallToGrid(&wallGrid, candidate.wx, candidate.wy, candidate.orient, false)
			}
		}
	}

	if bestAction.Type == domain.ActionMove {
		b.recordPositionLocked(bestAction.TargetX, bestAction.TargetY)
	}

	return bestAction
}

func (b *HeuristicBot) doesMoveConcedeForwardJump(g *domain.Game, bot *domain.Player, opp *domain.Player, grid *WallGrid) bool {
	var oppMoveBuf [8]domain.Move
	oppMoveCount := g.GetValidMovesInto(opp, &oppMoveBuf)
	return b.doesMoveConcedeForwardJumpWithMoves(g, bot, opp, grid, oppMoveBuf[:oppMoveCount])
}

func (b *HeuristicBot) doesMoveConcedeForwardJumpWithMoves(g *domain.Game, bot *domain.Player, opp *domain.Player, grid *WallGrid, oppMoves []domain.Move) bool {
	initialOppDist := b.getFastShortestPath(grid, opp.GridX, opp.GridY, opp.TargetX, opp.TargetY)
	for _, om := range oppMoves {
		dx := om.X - opp.GridX
		dy := om.Y - opp.GridY
		if dx < 0 {
			dx = -dx
		}
		if dy < 0 {
			dy = -dy
		}

		if dx > 1 || dy > 1 {
			newOppDist := b.getFastShortestPath(grid, om.X, om.Y, opp.TargetX, opp.TargetY)
			if newOppDist < initialOppDist {
				return true
			}
		}
	}
	return false
}

func (b *HeuristicBot) evaluateWorstCaseOpponentResponse(g *domain.Game, me *domain.Player, opponent *domain.Player, w BotWeights, grid *WallGrid) float64 {
	worstScoreForBot := math.MaxFloat64
	meDist := float64(b.getFastShortestPath(grid, me.GridX, me.GridY, me.TargetX, me.TargetY))

	var oppMoveBuf [8]domain.Move
	oppMoveCount := g.GetValidMovesInto(opponent, &oppMoveBuf)
	if oppMoveCount == 0 {
		return b.evaluateStateWithDistances(meDist, float64(b.getFastShortestPath(grid, opponent.GridX, opponent.GridY, opponent.TargetX, opponent.TargetY))-0.5, me, opponent, w, grid)
	}

	for i := 0; i < oppMoveCount; i++ {
		om := oppMoveBuf[i]
		oldOppX, oldOppY := opponent.GridX, opponent.GridY
		opponent.GridX, opponent.GridY = om.X, om.Y

		opponentDist := float64(b.getFastShortestPath(grid, opponent.GridX, opponent.GridY, opponent.TargetX, opponent.TargetY)) - 0.5
		score := b.evaluateStateWithDistances(meDist, opponentDist, me, opponent, w, grid)

		opponent.GridX, opponent.GridY = oldOppX, oldOppY

		if score < worstScoreForBot {
			worstScoreForBot = score
		}
	}

	return worstScoreForBot
}

func (b *HeuristicBot) evaluateState(me *domain.Player, opponent *domain.Player, w BotWeights, grid *WallGrid) float64 {
	myDist := float64(b.getFastShortestPath(grid, me.GridX, me.GridY, me.TargetX, me.TargetY))
	oppDist := float64(b.getFastShortestPath(grid, opponent.GridX, opponent.GridY, opponent.TargetX, opponent.TargetY)) - 0.5
	return b.evaluateStateWithDistances(myDist, oppDist, me, opponent, w, grid)
}

func (b *HeuristicBot) evaluateStateWithDistances(myDist, oppDist float64, me *domain.Player, opponent *domain.Player, w BotWeights, grid *WallGrid) float64 {
	wallsLeft := float64(me.WallsLeft)

	boardCenter := float64(domain.BoardSize-1) / 2.0
	var lateralOffset float64
	if me.TargetY != -1 {
		lateralOffset = math.Abs(float64(me.GridX) - boardCenter)
	} else {
		lateralOffset = math.Abs(float64(me.GridY) - boardCenter)
	}
	centrality := boardCenter - lateralOffset

	// Bonus for funneling: limiting the opponent's mobility around its current square.
	oppFreePaths := 0
	for d := 0; d < 4; d++ {
		if !grid[opponent.GridX][opponent.GridY][d] {
			oppFreePaths++
		}
	}
	funneling := float64(4 - oppFreePaths)

	return (w.MyDistanceWeight * myDist) +
		(w.RivalDistanceWeight * oppDist) +
		(w.WallReserveWeight * wallsLeft) +
		(w.CentralityWeight * centrality) +
		(w.FunnelingBonus * funneling)
}

func (b *HeuristicBot) getClosestOpponent(g *domain.Game, me *domain.Player, grid *WallGrid) *domain.Player {
	var closest *domain.Player
	minDist := 999

	for _, p := range g.Players {
		if p.ID == me.ID {
			continue
		}
		dist := b.getFastShortestPath(grid, p.GridX, p.GridY, p.TargetX, p.TargetY)
		if dist < minDist {
			minDist = dist
			closest = p
		}
	}
	return closest
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
