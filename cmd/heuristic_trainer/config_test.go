package main

import (
	"os"
	"path/filepath"
	"quoridor/internal/adapters/bot"
	"quoridor/internal/domain"
	"testing"
)

func TestLoadConfigFromYAML(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "trainer.yaml")

	content := `epochs: 7
generations: 12
population_size: 32
top_keep: 8
mutation_rate_start: 0.4
mutation_rate_end: 0.1
mutation_amount_start: 2.5
mutation_amount_end: 0.8
max_turns: 200
seed_bots:
  - e2g2m69
  - e1g4m38
benchmark_bots:
  - default_heuristic
  - minimax_3
`

	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig returned an error: %v", err)
	}

	if cfg.Epochs != 7 {
		t.Fatalf("expected epochs 7, got %d", cfg.Epochs)
	}
	if cfg.Generations != 12 {
		t.Fatalf("expected generations 12, got %d", cfg.Generations)
	}
	if cfg.PopulationSize != 32 {
		t.Fatalf("expected population size 32, got %d", cfg.PopulationSize)
	}
	if cfg.TopKeep != 8 {
		t.Fatalf("expected topKeep 8, got %d", cfg.TopKeep)
	}
	if len(cfg.SeedBots) != 2 {
		t.Fatalf("expected 2 seed bots, got %d", len(cfg.SeedBots))
	}
	if len(cfg.BenchmarkBots) != 2 {
		t.Fatalf("expected 2 benchmark bots, got %d", len(cfg.BenchmarkBots))
	}
}

func TestApplyDefaultsUsesSeedAndBenchmarkFallbacks(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SeedBots = nil
	cfg.BenchmarkBots = nil

	ApplyDefaults(&cfg)

	if len(cfg.SeedBots) == 0 {
		t.Fatal("expected default seed bots to be populated")
	}
	if len(cfg.BenchmarkBots) == 0 {
		t.Fatal("expected default benchmark bots to be populated")
	}
}

func TestBuildInitialPopulationIncludesAllSeedBots(t *testing.T) {
	seedNames := []string{"aggressive", "defensive", "balanced", "chaotic"}
	seedConfigs, seedIDs := buildSeedPopulation(seedNames)
	population := buildInitialPopulation(seedConfigs, seedIDs, 8, 0.3, 2.0)

	if len(population) != 8 {
		t.Fatalf("expected 8 individuals, got %d", len(population))
	}

	seen := map[string]bool{}
	for i := 0; i < len(seedConfigs); i++ {
		seen[population[i].ID] = true
	}

	if len(seen) != 4 {
		t.Fatalf("expected all 4 seed configurations to be present in the initial population, got %d unique first entries", len(seen))
	}

	for i := 0; i < len(seedIDs); i++ {
		if population[i].ID != seedIDs[i] {
			t.Fatalf("expected seed id %q at index %d, got %q", seedIDs[i], i, population[i].ID)
		}
	}
}

func TestMutationProfileInterpolatesByEpoch(t *testing.T) {
	cfg := TrainingConfig{
		Epochs:              5,
		MutationRateStart:   0.3,
		MutationRateEnd:     0.08,
		MutationAmountStart: 2.0,
		MutationAmountEnd:   0.6,
	}

	if got := cfg.mutationRateForEpoch(1); got != 0.3 {
		t.Fatalf("expected first epoch rate to be 0.3, got %f", got)
	}
	if got := cfg.mutationRateForEpoch(5); got != 0.08 {
		t.Fatalf("expected last epoch rate to be 0.08, got %f", got)
	}
	if got := cfg.mutationAmountForEpoch(3); got <= 0.6 || got >= 2.0 {
		t.Fatalf("expected middle epoch mutation amount to be between min and max, got %f", got)
	}
}

func TestResolveGamesPerEvalTracksBenchmarkGrowth(t *testing.T) {
	cfg := TrainingConfig{GamesPerEval: 0}
	benchmarks := []BenchmarkRival{
		{Name: "r1", Factory: func() domain.PlayerController { return nil }},
		{Name: "r2", Factory: func() domain.PlayerController { return nil }},
		{Name: "r3", Factory: func() domain.PlayerController { return nil }},
	}

	if got := resolveGamesPerEval(cfg, benchmarks); got != 6 {
		t.Fatalf("expected 6 games for 3 rivals, got %d", got)
	}

	benchmarks = append(benchmarks, BenchmarkRival{Name: "r4", Factory: func() domain.PlayerController { return nil }})
	if got := resolveGamesPerEval(cfg, benchmarks); got != 8 {
		t.Fatalf("expected 8 games after adding another rival, got %d", got)
	}
}

func TestApplyDefaultsLeavesGamesPerEvalDynamicWhenUnset(t *testing.T) {
	cfg := TrainingConfig{BenchmarkBots: []string{"aggressive", "defensive", "balanced", "chaotic"}}
	ApplyDefaults(&cfg)

	if cfg.GamesPerEval != 0 {
		t.Fatalf("expected games_per_evaluation to remain dynamic when unset, got %d", cfg.GamesPerEval)
	}
}

func TestBenchmarkGrowthUsesSeparateEpochBudget(t *testing.T) {
	cfg := DefaultConfig()
	cfg.BenchmarkAddPerEpoch = 2
	cfg.BenchmarkMaxSize = 5
	benchmarks := buildBenchmarkFactories([]string{"aggressive", "defensive"})
	benchmarks = applyEpochBenchmarkPromotion(benchmarks, cfg, 2, 4, []promotedBenchmarkCandidate{
		{Config: bot.Bot_Aggressive(), Rank: 1},
		{Config: bot.Bot_Balanced(), Rank: 2},
		{Config: bot.Bot_Chaotic(), Rank: 3},
	})

	if len(benchmarks) != 4 {
		t.Fatalf("expected 4 rivals after adding 2 new unique benchmark bots, got %d", len(benchmarks))
	}
}

func TestBenchmarkPoolAccumulatesWithoutDuplicates(t *testing.T) {
	benchmarks := buildBenchmarkFactories([]string{"aggressive", "defensive"})
	benchmarks = appendUniqueBenchmarkFactories(benchmarks, []bot.BotConfig{
		bot.Bot_Aggressive(),
		bot.Bot_Balanced(),
	})
	benchmarks = appendUniqueBenchmarkFactories(benchmarks, []bot.BotConfig{
		bot.Bot_Balanced(),
		bot.Bot_Chaotic(),
	})

	if len(benchmarks) != 4 {
		t.Fatalf("expected 4 unique benchmark rivals after cumulative growth, got %d", len(benchmarks))
	}
}
