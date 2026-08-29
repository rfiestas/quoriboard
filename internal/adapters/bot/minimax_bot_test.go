package bot

import (
	"fmt"
	"image/color"
	"sync"
	"testing"

	"quoridor/internal/domain"
)

func newMinimaxScenario(depth int) (*MinimaxBot, *domain.Game, *domain.Player) {
	g := &domain.Game{}
	g.InitBoard([]domain.PlayerConfig{
		{ID: 1, Color: color.White},
		{ID: 2, Color: color.Black},
	})

	bot := NewMinimaxBot(0, depth)
	return bot, g, g.Players[0]
}

func TestMinimaxBot_InitialMoveIsLegal(t *testing.T) {
	g := &domain.Game{}
	g.InitBoard([]domain.PlayerConfig{
		{ID: 1, Color: color.White},
		{ID: 2, Color: color.Black},
	})

	bot := NewMinimaxBot(0, 2)

	for _, p := range g.Players {
		act := bot.GetAction(g, p)
		if act.Type == domain.ActionNone {
			t.Fatalf("expected a legal action for player %d, got %+v", p.ID, act)
		}
		if act.Type == domain.ActionMove {
			if act.TargetX < 0 || act.TargetX >= domain.BoardSize || act.TargetY < 0 || act.TargetY >= domain.BoardSize {
				t.Fatalf("move out of bounds for player %d: %+v", p.ID, act)
			}
		}
	}
}

func TestMinimaxBot_ApplyActionRevertsState(t *testing.T) {
	bot, g, me := newMinimaxScenario(2)

	origX, origY := me.GridX, me.GridY
	move := domain.PlayerAction{Type: domain.ActionMove, TargetX: origX, TargetY: origY - 1}
	revertMove := bot.applyAction(g, me, move)
	if me.GridX != move.TargetX || me.GridY != move.TargetY {
		t.Fatalf("move action did not mutate player as expected: got (%d,%d)", me.GridX, me.GridY)
	}
	revertMove()
	if me.GridX != origX || me.GridY != origY {
		t.Fatalf("move revert failed, got (%d,%d), expected (%d,%d)", me.GridX, me.GridY, origX, origY)
	}

	origWallsLeft := me.WallsLeft
	origWallsCount := len(g.Walls)
	wallAction := domain.PlayerAction{Type: domain.ActionPlaceWall, TargetX: 3, TargetY: 6, WallOrientation: domain.WallHorizontal}
	revertWall := bot.applyAction(g, me, wallAction)
	if len(g.Walls) != origWallsCount+1 {
		t.Fatalf("wall action did not append wall, got %d want %d", len(g.Walls), origWallsCount+1)
	}
	if me.WallsLeft != origWallsLeft-1 {
		t.Fatalf("wall action did not decrement wall count, got %d want %d", me.WallsLeft, origWallsLeft-1)
	}
	revertWall()
	if len(g.Walls) != origWallsCount {
		t.Fatalf("wall revert did not restore wall count, got %d want %d", len(g.Walls), origWallsCount)
	}
	if me.WallsLeft != origWallsLeft {
		t.Fatalf("wall revert did not restore wallsLeft, got %d want %d", me.WallsLeft, origWallsLeft)
	}
}

func TestMinimaxBot_ParallelRootMatchesSequentialDecision(t *testing.T) {
	g := &domain.Game{}
	g.InitBoard([]domain.PlayerConfig{
		{ID: 1, Color: color.White},
		{ID: 2, Color: color.Black},
	})

	seqBot := NewMinimaxBot(0, 2)
	seqBot.ParallelRoot = false

	parBot := NewMinimaxBot(0, 2)
	parBot.ParallelRoot = true
	parBot.ParallelMinActions = 1

	seq := seqBot.GetAction(g, g.Players[0])
	par := parBot.GetAction(g, g.Players[0])

	if seq.Type != par.Type || seq.TargetX != par.TargetX || seq.TargetY != par.TargetY || seq.WallOrientation != par.WallOrientation {
		t.Fatalf("parallel root decision mismatch: sequential=%+v parallel=%+v", seq, par)
	}
}

func TestMinimaxBot_ConcurrentGetActionIsRaceSafe(t *testing.T) {
	bot, g, me := newMinimaxScenario(2)
	bot.ParallelRoot = false

	const workers = 12
	const rounds = 100
	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < rounds; j++ {
				_ = bot.GetAction(g, me)
			}
		}()
	}

	close(start)
	wg.Wait()
}

func BenchmarkMinimaxBotGetActionDepth2(b *testing.B) {
	bot, g, p := newMinimaxScenario(2)
	bot.ParallelRoot = false

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = bot.GetAction(g, p)
	}
}

var minimaxDepthSweep = []int{1, 2, 3, 4}

func benchmarkMinimaxDepth(b *testing.B, depth int, parallelRoot bool) {
	bot, g, p := newMinimaxScenario(depth)
	bot.ParallelRoot = parallelRoot
	if parallelRoot {
		bot.ParallelMinActions = 1
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = bot.GetAction(g, p)
	}
}

func BenchmarkMinimaxBotGetActionDepthSweepSequential(b *testing.B) {
	for _, depth := range minimaxDepthSweep {
		depth := depth
		b.Run(fmt.Sprintf("depth_%d", depth), func(b *testing.B) {
			benchmarkMinimaxDepth(b, depth, false)
		})
	}
}

func BenchmarkMinimaxBotGetActionDepthSweepParallelRoot(b *testing.B) {
	for _, depth := range minimaxDepthSweep {
		depth := depth
		b.Run(fmt.Sprintf("depth_%d", depth), func(b *testing.B) {
			benchmarkMinimaxDepth(b, depth, true)
		})
	}
}

func BenchmarkMinimaxBotGetActionDepth2ParallelRoot(b *testing.B) {
	bot, g, p := newMinimaxScenario(2)
	bot.ParallelRoot = true
	bot.ParallelMinActions = 1

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = bot.GetAction(g, p)
	}
}

func BenchmarkMinimaxBotGetActionParallelIsolatedInstances(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		bot, g, p := newMinimaxScenario(2)
		bot.ParallelRoot = false
		for pb.Next() {
			_ = bot.GetAction(g, p)
		}
	})
}

func BenchmarkMinimaxBotShortestPathLength(b *testing.B) {
	bot, g, p := newMinimaxScenario(2)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = bot.getShortestPathLength(g, p)
	}
}
