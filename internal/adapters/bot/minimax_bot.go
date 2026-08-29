package bot

import (
	"math"
	"sort"
	"sync"
	"time"

	"quoridor/internal/domain"
)

const (
	//MaxDepth        = 3
	InfScore        = 100000.0
	PathSearchLimit = 999
)

/*type MinimaxBot struct {
	Delay         time.Duration
	MaxDepth      int
	thinkingStart time.Time
	isThinking    bool
	history       [][2]int // Historial exclusivo de turnos reales
}

func NewMinimaxBot(delay time.Duration, maxDepth int) *MinimaxBot {
	return &MinimaxBot{
		Delay:    delay,
		MaxDepth: maxDepth,
		history:  make([][2]int, 0, 6),
	}
}*/

type MinimaxBot struct {
	Delay         time.Duration
	MaxDepth      int
	thinkingStart time.Time
	isThinking    bool
	history       [][2]int
	tt            map[uint64]TTEntry // Tabla de Transposición
	mu            sync.Mutex

	// Root-level parallelism can speed up wide positions on multicore CPUs.
	ParallelRoot       bool
	ParallelMinActions int

	// Skip wall generation when remaining depth is small to reduce branching.
	WallSearchDepthCutoff int
}

func NewMinimaxBot(delay time.Duration, maxDepth int) *MinimaxBot {
	return &MinimaxBot{
		Delay:                 delay,
		MaxDepth:              maxDepth,
		history:               make([][2]int, 0, 6),
		tt:                    make(map[uint64]TTEntry, 100000), // Capacidad reservada de origen
		ParallelRoot:          false,
		ParallelMinActions:    4,
		WallSearchDepthCutoff: 1,
	}
}

// computeHash calcula la firma Zobrist única del estado actual del tablero en O(W)
func (b *MinimaxBot) computeHash(g *domain.Game, currentPlayer *domain.Player) uint64 {
	var hash uint64

	// 1. Hash de posiciones de los jugadores
	for _, p := range g.Players {
		hash ^= zobristPlayers[p.ID][p.GridX][p.GridY]
	}

	// 2. Hash de los muros en el tablero
	for _, w := range g.Walls {
		hash ^= zobristWalls[w.GridX][w.GridY][w.Orientation]
	}

	// 3. Hash del turno activo
	hash ^= zobristTurn[currentPlayer.ID]

	return hash
}

func (b *MinimaxBot) GetAction(g *domain.Game, p *domain.Player) domain.PlayerAction {
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
	return b.selectBestActionMinimax(g, p)
}

func (b *MinimaxBot) selectBestActionMinimax(g *domain.Game, me *domain.Player) domain.PlayerAction {
	if b.tt == nil {
		b.tt = make(map[uint64]TTEntry, 100000)
	} else {
		clear(b.tt)
	}

	opponent := b.getOpponent(g, me)
	if opponent == nil {
		return domain.PlayerAction{Type: domain.ActionNone}
	}

	var validMoveBuf [8]domain.Move
	validMoveCount := g.GetValidMovesInto(me, &validMoveBuf)
	if validMoveCount == 0 {
		return domain.PlayerAction{Type: domain.ActionNone}
	}

	validMoves := validMoveBuf[:validMoveCount]

	// Red de seguridad inmediata: primer movimiento legal
	fallbackAction := domain.PlayerAction{
		Type:    domain.ActionMove,
		TargetX: validMoves[0].X,
		TargetY: validMoves[0].Y,
	}

	var bestAction domain.PlayerAction = fallbackAction
	bestScore := -InfScore

	alpha := -InfScore
	beta := InfScore

	actions := b.generateCandidateActions(g, me, opponent, b.MaxDepth)
	if len(actions) == 0 {
		return fallbackAction
	}

	if b.ParallelRoot && len(actions) >= b.ParallelMinActions && b.MaxDepth > 1 {
		scores := b.evaluateRootActionsParallel(g, me, opponent, actions)
		for i, act := range actions {
			score := scores[i]
			if score > bestScore {
				bestScore = score
				bestAction = act
			}
			if score > alpha {
				alpha = score
			}
		}
	} else {
		for _, act := range actions {
			revert := b.applyAction(g, me, act)

			// Evaluamos el árbol Minimax
			score := b.minimax(g, me, opponent, b.MaxDepth-1, false, alpha, beta)

			// Aplicamos penalización de repetición SÓLO al movimiento candidato actual
			if act.Type == domain.ActionMove {
				score += b.getRepetitionPenalty(act.TargetX, act.TargetY)
			}

			revert()

			if score > bestScore {
				bestScore = score
				bestAction = act
			}
			if score > alpha {
				alpha = score
			}
		}
	}

	if bestAction.Type == domain.ActionNone {
		bestAction = fallbackAction
	}

	// Registrar en el historial únicamente la decisión tomada en el turno real
	if bestAction.Type == domain.ActionMove {
		b.recordPosition(bestAction.TargetX, bestAction.TargetY)
	}

	return bestAction
}

/*
	func (b *MinimaxBot) minimax(g *domain.Game, bot *domain.Player, opp *domain.Player, depth int, isMaximizing bool, alpha, beta float64) float64 {
		botDist := b.getShortestPathLength(g, bot)
		oppDist := b.getShortestPathLength(g, opp)

		if botDist == 0 {
			return InfScore + float64(depth)
		}
		if oppDist == 0 {
			return -InfScore - float64(depth)
		}
		if depth <= 0 {
			return b.evaluateState(g, bot, opp, botDist, oppDist)
		}

		currentPlayer := bot
		if !isMaximizing {
			currentPlayer = opp
		}

		actions := b.generateCandidateActions(g, currentPlayer, b.getOther(bot, opp, currentPlayer))
		if len(actions) == 0 {
			return b.evaluateState(g, bot, opp, botDist, oppDist)
		}

		if isMaximizing {
			maxEval := -InfScore
			for _, act := range actions {
				revert := b.applyAction(g, currentPlayer, act)
				eval := b.minimax(g, bot, opp, depth-1, false, alpha, beta)
				revert()

				if eval > maxEval {
					maxEval = eval
				}
				if eval > alpha {
					alpha = eval
				}
				if beta <= alpha {
					break // Poda Alpha-Beta
				}
			}
			return maxEval
		} else {
			minEval := InfScore
			for _, act := range actions {
				revert := b.applyAction(g, currentPlayer, act)
				eval := b.minimax(g, bot, opp, depth-1, true, alpha, beta)
				revert()

				if eval < minEval {
					minEval = eval
				}
				if eval < beta {
					beta = eval
				}
				if beta <= alpha {
					break // Poda Alpha-Beta
				}
			}
			return minEval
		}
	}
*/

/*
func (b *MinimaxBot) minimax(g *domain.Game, bot *domain.Player, opp *domain.Player, depth int, isMaximizing bool, alpha, beta float64) float64 {
	// 1. Verificación instantánea de victoria en O(1) para nodos internos
	if g.CheckWin(bot) {
		return InfScore + float64(depth)
	}
	if g.CheckWin(opp) {
		return -InfScore - float64(depth)
	}

	// 2. Si alcanzamos el límite de profundidad, calculamos el BFS para evaluar la hoja
	if depth <= 0 {
		botDist := b.getShortestPathLength(g, bot)
		oppDist := b.getShortestPathLength(g, opp)
		return b.evaluateState(g, bot, opp, botDist, oppDist)
	}

	currentPlayer := bot
	if !isMaximizing {
		currentPlayer = opp
	}

	actions := b.generateCandidateActions(g, currentPlayer, b.getOther(bot, opp, currentPlayer), depth)
	if len(actions) == 0 {
		// Si no hay acciones legales, evaluamos el estado actual
		botDist := b.getShortestPathLength(g, bot)
		oppDist := b.getShortestPathLength(g, opp)
		return b.evaluateState(g, bot, opp, botDist, oppDist)
	}

	if isMaximizing {
		maxEval := -InfScore
		for _, act := range actions {
			revert := b.applyAction(g, currentPlayer, act)
			eval := b.minimax(g, bot, opp, depth-1, false, alpha, beta)
			revert()

			if eval > maxEval {
				maxEval = eval
			}
			if eval > alpha {
				alpha = eval
			}
			if beta <= alpha {
				break // Poda Alpha-Beta
			}
		}
		return maxEval
	} else {
		minEval := InfScore
		for _, act := range actions {
			revert := b.applyAction(g, currentPlayer, act)
			eval := b.minimax(g, bot, opp, depth-1, true, alpha, beta)
			revert()

			if eval < minEval {
				minEval = eval
			}
			if eval < beta {
				beta = eval
			}
			if beta <= alpha {
				break // Poda Alpha-Beta
			}
		}
		return minEval
	}
}*/

func (b *MinimaxBot) minimax(g *domain.Game, bot *domain.Player, opp *domain.Player, depth int, isMaximizing bool, alpha, beta float64) float64 {
	// Check directo de victoria
	if g.CheckWin(bot) {
		return InfScore + float64(depth)
	}
	if g.CheckWin(opp) {
		return -InfScore - float64(depth)
	}

	if depth <= 0 {
		botDist := b.getShortestPathLength(g, bot)
		oppDist := b.getShortestPathLength(g, opp)
		return b.evaluateState(g, bot, opp, botDist, oppDist)
	}

	currentPlayer := bot
	if !isMaximizing {
		currentPlayer = opp
	}

	// --- 1. CONSULTA DE TABLA DE TRANSPOSICIÓN ---
	alphaOrig := alpha
	betaOrig := beta
	hash := b.computeHash(g, currentPlayer)

	if b.tt != nil {
		if entry, found := b.tt[hash]; found && entry.Depth >= depth {
			if entry.Flag == TTExact {
				return entry.Score
			} else if entry.Flag == TTLowerBound {
				if entry.Score > alpha {
					alpha = entry.Score
				}
			} else if entry.Flag == TTUpperBound {
				if entry.Score < beta {
					beta = entry.Score
				}
			}
			if alpha >= beta {
				return entry.Score
			}
		}
	}

	actions := b.generateCandidateActions(g, currentPlayer, b.getOther(bot, opp, currentPlayer), depth)
	if len(actions) == 0 {
		botDist := b.getShortestPathLength(g, bot)
		oppDist := b.getShortestPathLength(g, opp)
		return b.evaluateState(g, bot, opp, botDist, oppDist)
	}

	var bestValue float64

	if isMaximizing {
		bestValue = -InfScore
		for _, act := range actions {
			revert := b.applyAction(g, currentPlayer, act)
			eval := b.minimax(g, bot, opp, depth-1, false, alpha, beta)
			revert()

			if eval > bestValue {
				bestValue = eval
			}
			if eval > alpha {
				alpha = eval
			}
			if beta <= alpha {
				break
			}
		}
	} else {
		bestValue = InfScore
		for _, act := range actions {
			revert := b.applyAction(g, currentPlayer, act)
			eval := b.minimax(g, bot, opp, depth-1, true, alpha, beta)
			revert()

			if eval < bestValue {
				bestValue = eval
			}
			if eval < beta {
				beta = eval
			}
			if beta <= alpha {
				break
			}
		}
	}

	// --- 2. ALMACENAMIENTO EN TABLA DE TRANSPOSICIÓN ---
	var flag TTFlag
	if bestValue <= alphaOrig {
		flag = TTUpperBound
	} else if bestValue >= betaOrig {
		flag = TTLowerBound
	} else {
		flag = TTExact
	}

	if b.tt != nil {
		b.tt[hash] = TTEntry{
			Depth: depth,
			Score: bestValue,
			Flag:  flag,
		}
	}

	return bestValue
}

func (b *MinimaxBot) evaluateRootActionsParallel(g *domain.Game, me *domain.Player, opponent *domain.Player, actions []domain.PlayerAction) []float64 {
	scores := make([]float64, len(actions))
	var wg sync.WaitGroup

	for i, act := range actions {
		wg.Add(1)
		go func(idx int, action domain.PlayerAction) {
			defer wg.Done()

			gClone := cloneGameState(g)
			meClone := getPlayerByID(gClone, me.ID)
			oppClone := getPlayerByID(gClone, opponent.ID)
			if meClone == nil || oppClone == nil {
				scores[idx] = -InfScore
				return
			}

			localBot := MinimaxBot{
				Delay:              b.Delay,
				MaxDepth:           b.MaxDepth,
				history:            b.history,
				tt:                 nil,
				ParallelRoot:       false,
				ParallelMinActions: b.ParallelMinActions,
			}

			revert := localBot.applyAction(gClone, meClone, action)
			score := localBot.minimax(gClone, meClone, oppClone, localBot.MaxDepth-1, false, -InfScore, InfScore)
			revert()

			if action.Type == domain.ActionMove {
				score += localBot.getRepetitionPenalty(action.TargetX, action.TargetY)
			}

			scores[idx] = score
		}(i, act)
	}

	wg.Wait()
	return scores
}

func cloneGameState(g *domain.Game) *domain.Game {
	players := make([]*domain.Player, len(g.Players))
	for i, p := range g.Players {
		pCopy := *p
		players[i] = &pCopy
	}

	walls := make([]domain.Wall, len(g.Walls))
	copy(walls, g.Walls)

	return &domain.Game{
		State:     g.State,
		Players:   players,
		TurnIndex: g.TurnIndex,
		Walls:     walls,
	}
}

func getPlayerByID(g *domain.Game, id int) *domain.Player {
	for _, p := range g.Players {
		if p.ID == id {
			return p
		}
	}
	return nil
}

func (b *MinimaxBot) generateCandidateActions(g *domain.Game, active *domain.Player, passive *domain.Player, depth int) []domain.PlayerAction {
	//var actions []domain.PlayerAction
	/// Reservamos espacio suficiente para evitar reallocations de memoria durante el bucle
	actions := make([]domain.PlayerAction, 0, 60)

	// 1. Movimientos de peón
	var validMoveBuf [8]domain.Move
	validMoveCount := g.GetValidMovesInto(active, &validMoveBuf)
	for i := 0; i < validMoveCount; i++ {
		m := validMoveBuf[i]
		actions = append(actions, domain.PlayerAction{
			Type:    domain.ActionMove,
			TargetX: m.X,
			TargetY: m.Y,
		})
	}

	// 2. Colocación de muros (limitado a la ventana táctica)
	if active.WallsLeft > 0 && depth > b.WallSearchDepthCutoff {
		orientations := []domain.WallOrientation{domain.WallHorizontal, domain.WallVertical}

		minX := math.Max(0, float64(min(active.GridX, passive.GridX)-2))
		maxX := math.Min(float64(domain.BoardSize-2), float64(max(active.GridX, passive.GridX)+2))
		minY := math.Max(0, float64(min(active.GridY, passive.GridY)-2))
		maxY := math.Min(float64(domain.BoardSize-2), float64(max(active.GridY, passive.GridY)+2))

		for wx := int(minX); wx <= int(maxX); wx++ {
			for wy := int(minY); wy <= int(maxY); wy++ {
				for _, orient := range orientations {
					if g.CanPlaceWall(wx, wy, orient, active) {
						actions = append(actions, domain.PlayerAction{
							Type:            domain.ActionPlaceWall,
							TargetX:         wx,
							TargetY:         wy,
							WallOrientation: orient,
						})
					}
				}
			}
		}
	}

	sort.Slice(actions, func(i, j int) bool {
		return b.scoreAction(active, actions[i]) > b.scoreAction(active, actions[j])
	})

	return actions
}

func (b *MinimaxBot) evaluateState(g *domain.Game, bot *domain.Player, opp *domain.Player, botDist, oppDist int) float64 {
	distScore := float64(oppDist-botDist) * 10.0
	wallScore := float64(bot.WallsLeft-opp.WallsLeft) * 2.0

	// 1. Calcular el centro del tablero de forma dinámica
	boardCenter := float64(domain.BoardSize-1) / 2.0 // (9-1)/2 = 4.0

	// 2. Determinar el eje de desviación lateral según la meta del jugador
	var lateralOffset float64
	if bot.TargetY != -1 {
		// Se mueve verticalmente (P1/P2): la desviación es en el eje X
		lateralOffset = math.Abs(float64(bot.GridX) - boardCenter)
	} else {
		// Se mueve horizontalmente (P3/P4): la desviación es en el eje Y
		lateralOffset = math.Abs(float64(bot.GridY) - boardCenter)
	}

	// 3. Puntuación de centralidad simétrica (máxima en el centro, disminuye hacia los bordes)
	centerScore := (boardCenter - lateralOffset) * 0.5

	return distScore + wallScore + centerScore
}

func (b *MinimaxBot) applyAction(g *domain.Game, p *domain.Player, act domain.PlayerAction) func() {
	if act.Type == domain.ActionMove {
		oldX, oldY := p.GridX, p.GridY
		p.GridX, p.GridY = act.TargetX, act.TargetY
		return func() {
			p.GridX, p.GridY = oldX, oldY
		}
	} else if act.Type == domain.ActionPlaceWall {
		wall := domain.Wall{
			GridX:       act.TargetX,
			GridY:       act.TargetY,
			Orientation: act.WallOrientation,
			OwnerID:     p.ID,
		}
		g.Walls = append(g.Walls, wall)
		p.WallsLeft--
		return func() {
			g.Walls = g.Walls[:len(g.Walls)-1]
			p.WallsLeft++
		}
	}
	return func() {}
}

func (b *MinimaxBot) recordPosition(x, y int) {
	b.history = append(b.history, [2]int{x, y})
	if len(b.history) > 6 {
		b.history = b.history[1:]
	}
}

func (b *MinimaxBot) getRepetitionPenalty(x, y int) float64 {
	penalty := 0.0
	for i, pos := range b.history {
		if pos[0] == x && pos[1] == y {
			penalty -= float64(i+1) * 4.0
		}
	}
	return penalty
}

/*
	func (b *MinimaxBot) getShortestPathLength(g *domain.Game, p *domain.Player) int {
		// Copia por valor limpia para evitar mutaciones de memoria
		simPlayer := domain.Player{
			ID:        p.ID,
			GridX:     p.GridX,
			GridY:     p.GridY,
			TargetX:   p.TargetX,
			TargetY:   p.TargetY,
			WallsLeft: p.WallsLeft,
		}

		visited := make(map[[2]int]int)
		queue := [][3]int{{simPlayer.GridX, simPlayer.GridY, 0}}
		visited[[2]int{simPlayer.GridX, simPlayer.GridY}] = 0

		for len(queue) > 0 {
			curr := queue[0]
			queue = queue[1:]

			cx, cy, dist := curr[0], curr[1], curr[2]

			if (simPlayer.TargetY != -1 && cy == simPlayer.TargetY) ||
				(simPlayer.TargetX != -1 && cx == simPlayer.TargetX) {
				return dist
			}

			simPlayer.GridX = cx
			simPlayer.GridY = cy
			validMoves := g.GetValidMoves(&simPlayer)

			for _, m := range validMoves {
				coord := [2]int{m[0], m[1]}
				if _, seen := visited[coord]; !seen {
					visited[coord] = dist + 1
					queue = append(queue, [3]int{m[0], m[1], dist + 1})
				}
			}
		}

		return PathSearchLimit
	}
*/
func (b *MinimaxBot) getShortestPathLength(g *domain.Game, p *domain.Player) int {
	// Matriz estática 9x9 con distancias (-1 = no visitado)
	var visited [domain.BoardSize][domain.BoardSize]int
	for i := 0; i < domain.BoardSize; i++ {
		for j := 0; j < domain.BoardSize; j++ {
			visited[i][j] = -1
		}
	}

	// Cola estática de tamaño fijo [X, Y, Distancia]
	var queue [domain.BoardSize * domain.BoardSize][3]int
	head, tail := 0, 0

	queue[tail] = [3]int{p.GridX, p.GridY, 0}
	tail++
	visited[p.GridX][p.GridY] = 0

	// Jugador simulado para reutilizar en GetValidMoves
	simPlayer := domain.Player{
		ID:        p.ID,
		GridX:     p.GridX,
		GridY:     p.GridY,
		TargetX:   p.TargetX,
		TargetY:   p.TargetY,
		WallsLeft: p.WallsLeft,
	}

	for head < tail {
		curr := queue[head]
		head++

		cx, cy, dist := curr[0], curr[1], curr[2]

		// Meta alcanzada
		if (simPlayer.TargetY != -1 && cy == simPlayer.TargetY) ||
			(simPlayer.TargetX != -1 && cx == simPlayer.TargetX) {
			return dist
		}

		simPlayer.GridX = cx
		simPlayer.GridY = cy
		var validMoveBuf [8]domain.Move
		validMoveCount := g.GetValidMovesInto(&simPlayer, &validMoveBuf)

		for i := 0; i < validMoveCount; i++ {
			m := validMoveBuf[i]
			nx, ny := m.X, m.Y
			if visited[nx][ny] == -1 {
				visited[nx][ny] = dist + 1
				queue[tail] = [3]int{nx, ny, dist + 1}
				tail++
			}
		}
	}

	return PathSearchLimit
}

func (b *MinimaxBot) getOpponent(g *domain.Game, me *domain.Player) *domain.Player {
	for _, p := range g.Players {
		if p.ID != me.ID {
			return p
		}
	}
	return nil
}

func (b *MinimaxBot) getOther(p1, p2, current *domain.Player) *domain.Player {
	if current.ID == p1.ID {
		return p2
	}
	return p1
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// scoreAction asigna una prioridad a cada acción para la poda Alpha-Beta.
// Valores más altos indican jugadas potencialmente mejores que deben evaluarse primero.
func (b *MinimaxBot) scoreAction(p *domain.Player, act domain.PlayerAction) int {
	if act.Type == domain.ActionMove {
		// Calcular la ganancia de distancia hacia la meta
		currentDist := 0
		newDist := 0

		if p.TargetY != -1 {
			currentDist = abs(p.GridY - p.TargetY)
			newDist = abs(act.TargetY - p.TargetY)
		} else {
			currentDist = abs(p.GridX - p.TargetX)
			newDist = abs(act.TargetX - p.TargetX)
		}

		// Si el movimiento reduce la distancia a la meta, asignamos máxima prioridad
		if newDist < currentDist {
			return 100
		}
		return 50 // Movimiento lateral/atrás
	}

	// Colocación de muros (prioridad base inferior al movimiento directo)
	return 10
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}
