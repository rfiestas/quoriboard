package bot

import (
	"image/color"
	"sync"
	"testing"

	"quoridor/internal/domain"
)

func TestHeuristicBot_InitialMovesRemainConsistentWithBoardTargets(t *testing.T) {
	g := &domain.Game{}
	g.InitBoard([]domain.PlayerConfig{
		{ID: 1, Color: color.White},
		{ID: 2, Color: color.Black},
	})

	bot := NewHeuristicBot(DefaultBotConfig(), 0)

	for _, p := range g.Players {
		move := bot.GetAction(g, p)
		if move.Type != domain.ActionMove {
			t.Fatalf("expected a move for player %d, got %+v", p.ID, move)
		}
		if move.TargetX < 0 || move.TargetX >= domain.BoardSize || move.TargetY < 0 || move.TargetY >= domain.BoardSize {
			t.Fatalf("move out of bounds for player %d: %+v", p.ID, move)
		}
	}
}

func TestHeuristicBot_MirroredBoardProducesMirroredDecision(t *testing.T) {
	base := []domain.PlayerConfig{
		{ID: 1, Color: color.White},
		{ID: 2, Color: color.Black},
	}

	g1 := &domain.Game{}
	g1.InitBoard(base)
	bot := NewHeuristicBot(DefaultBotConfig(), 0)

	first := bot.GetAction(g1, g1.Players[0])
	second := bot.GetAction(g1, g1.Players[1])

	g2 := &domain.Game{}
	g2.InitBoard(base)
	g2.Players[0].GridX, g2.Players[0].GridY = 4, 0
	g2.Players[0].TargetX, g2.Players[0].TargetY = -1, 8
	g2.Players[1].GridX, g2.Players[1].GridY = 4, 8
	g2.Players[1].TargetX, g2.Players[1].TargetY = -1, 0

	mirroredFirst := bot.GetAction(g2, g2.Players[0])
	mirroredSecond := bot.GetAction(g2, g2.Players[1])

	if first.Type != mirroredSecond.Type || first.TargetX != mirroredSecond.TargetX || first.TargetY != mirroredSecond.TargetY {
		t.Fatalf("mirrored decision mismatch: first=%+v mirroredSecond=%+v", first, mirroredSecond)
	}
	if second.Type != mirroredFirst.Type || second.TargetX != mirroredFirst.TargetX || second.TargetY != mirroredFirst.TargetY {
		t.Fatalf("mirrored decision mismatch: second=%+v mirroredFirst=%+v", second, mirroredFirst)
	}
}

func TestHeuristicBot_UsesRepetitionPenaltyOnLoopingMoves(t *testing.T) {
	g := &domain.Game{}
	g.InitBoard([]domain.PlayerConfig{
		{ID: 1, Color: color.White},
		{ID: 2, Color: color.Black},
	})

	bot := NewHeuristicBot(DefaultBotConfig(), 0)
	bot.recordPosition(4, 7)
	bot.recordPosition(4, 6)
	bot.recordPosition(4, 7)

	penalty := bot.getRepetitionPenalty(4, 7)
	if penalty >= 0 {
		t.Fatalf("expected repetition penalty to be negative, got %v", penalty)
	}
}

func TestHeuristicBot_GetFastShortestPath_ReturnsExpectedDistance(t *testing.T) {
	g := &domain.Game{}
	g.InitBoard([]domain.PlayerConfig{
		{ID: 1, Color: color.White},
		{ID: 2, Color: color.Black},
	})

	grid := buildWallGrid(g)
	p := g.Players[0]
	path := NewHeuristicBot(DefaultBotConfig(), 0).getFastShortestPath(&grid, p.GridX, p.GridY, p.TargetX, p.TargetY)
	if path != 8 {
		t.Fatalf("expected distance 8 from start to goal, got %d", path)
	}
}

func TestHeuristicBot_EvaluateStateRewardsCloserGoalAndDisincentivizesOpponent(t *testing.T) {
	g := &domain.Game{}
	g.InitBoard([]domain.PlayerConfig{
		{ID: 1, Color: color.White},
		{ID: 2, Color: color.Black},
	})

	me := g.Players[0]
	opponent := g.Players[1]
	grid := buildWallGrid(g)
	bot := NewHeuristicBot(DefaultBotConfig(), 0)

	baseScore := bot.evaluateState(me, opponent, DefaultBotConfig().NormalWeights, &grid)

	me.GridX, me.GridY = 4, 7
	opponent.GridX, opponent.GridY = 4, 1
	closerScore := bot.evaluateState(me, opponent, DefaultBotConfig().NormalWeights, &grid)

	if closerScore <= baseScore {
		t.Fatalf("expected closer position to improve score, got base=%v closer=%v", baseScore, closerScore)
	}
}

func TestHeuristicBot_GetClosestOpponentUsesOpponentGoal(t *testing.T) {
	g := &domain.Game{}
	g.Players = []*domain.Player{
		{ID: 1, GridX: 0, GridY: 0, TargetX: -1, TargetY: 8, WallsLeft: 10},
		{ID: 2, GridX: 8, GridY: 8, TargetX: -1, TargetY: 0, WallsLeft: 10},
		{ID: 3, GridX: 0, GridY: 7, TargetX: -1, TargetY: 0, WallsLeft: 10},
	}

	bot := NewHeuristicBot(DefaultBotConfig(), 0)
	grid := buildWallGrid(g)

	closest := bot.getClosestOpponent(g, g.Players[0], &grid)
	if closest == nil || closest.ID != 3 {
		t.Fatalf("expected opponent 3 to be closest by its own goal distance, got %+v", closest)
	}
}

func newHeuristicBotScenario() (*HeuristicBot, *domain.Game, *domain.Player) {
	g := &domain.Game{}
	g.InitBoard([]domain.PlayerConfig{
		{ID: 1, Color: color.White},
		{ID: 2, Color: color.Black},
	})

	bot := NewHeuristicBot(DefaultBotConfig(), 0)
	return bot, g, g.Players[0]
}

// BenchmarkHeuristicBotGetAction measures the baseline cost of a single decision cycle.
func TestHeuristicBot_ConcurrentGetActionIsRaceSafe(t *testing.T) {
	g := &domain.Game{}
	g.InitBoard([]domain.PlayerConfig{
		{ID: 1, Color: color.White},
		{ID: 2, Color: color.Black},
	})

	botInstance := NewHeuristicBot(DefaultBotConfig(), 0)
	const workers = 16
	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 200; j++ {
				_ = botInstance.GetAction(g, g.Players[0])
			}
		}()
	}

	close(start)
	wg.Wait()
}

func TestHeuristicBot_CandidatePruningTogglesBetweenModes(t *testing.T) {
	g := &domain.Game{}
	g.InitBoard([]domain.PlayerConfig{
		{ID: 1, Color: color.White},
		{ID: 2, Color: color.Black},
	})

	pruned := NewHeuristicBotWithPruning(DefaultBotConfig(), 0, true)
	full := NewHeuristicBotWithPruning(DefaultBotConfig(), 0, false)

	movePruned := pruned.GetAction(g, g.Players[0])
	moveFull := full.GetAction(g, g.Players[0])

	if movePruned.Type == domain.ActionNone || moveFull.Type == domain.ActionNone {
		t.Fatalf("expected valid actions in both pruning modes, got pruned=%+v full=%+v", movePruned, moveFull)
	}

	if movePruned.Type != moveFull.Type {
		t.Fatalf("same state should keep action type stable, got pruned=%+v full=%+v", movePruned, moveFull)
	}
}

func BenchmarkHeuristicBotGetAction(b *testing.B) {
	botInstance, g, p := newHeuristicBotScenario()
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = botInstance.GetAction(g, p)
	}
}

// BenchmarkHeuristicBotGetActionParallel verifies the isolated-per-goroutine pattern.
// Each goroutine owns its own bot and game instance, so memory is not shared across workers.
func BenchmarkHeuristicBotGetActionParallel(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		botInstance, g, p := newHeuristicBotScenario()
		for pb.Next() {
			_ = botInstance.GetAction(g, p)
		}
	})
}

// BenchmarkHeuristicBotEvaluateState measures the core evaluation cost without the move selection loop.
func BenchmarkHeuristicBotEvaluateState(b *testing.B) {
	g := &domain.Game{}
	g.InitBoard([]domain.PlayerConfig{
		{ID: 1, Color: color.White},
		{ID: 2, Color: color.Black},
	})

	me := g.Players[0]
	opponent := g.Players[1]
	grid := buildWallGrid(g)
	botInstance := NewHeuristicBot(DefaultBotConfig(), 0)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = botInstance.evaluateState(me, opponent, DefaultBotConfig().NormalWeights, &grid)
	}
}
