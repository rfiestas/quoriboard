package main

import (
	"bufio"
	"flag"
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
	"os"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gopkg.in/yaml.v3"

	"quoridor/internal/adapters/bot"
	"quoridor/internal/domain"
	"quoridor/internal/telemetry"
)

// TrainingConfig contains the training settings loaded from YAML or defaults.
type TrainingConfig struct {
	Epochs               int      `yaml:"epochs"`
	Generations          int      `yaml:"generations"`
	PopulationSize       int      `yaml:"population_size"`
	TopKeep              int      `yaml:"top_keep"`
	BenchmarkAddPerEpoch int      `yaml:"benchmark_add_per_epoch"`
	BenchmarkMaxSize     int      `yaml:"benchmark_max_size"`
	MutationRateStart    float64  `yaml:"mutation_rate_start"`
	MutationRateEnd      float64  `yaml:"mutation_rate_end"`
	MutationAmountStart  float64  `yaml:"mutation_amount_start"`
	MutationAmountEnd    float64  `yaml:"mutation_amount_end"`
	MaxTurns             int      `yaml:"max_turns"`
	GamesPerEval         int      `yaml:"games_per_evaluation"`
	SeedBots             []string `yaml:"seed_bots"`
	BenchmarkBots        []string `yaml:"benchmark_bots"`
	WebListenAddr        string   `yaml:"web_listen_addr"`
	DashboardPath        string   `yaml:"dashboard_path"`
	DataJSONPath         string   `yaml:"data_json_path"`
}

type Individual struct {
	ID          string
	Config      bot.BotConfig
	Wins        int
	GamesPlayed int
	Fitness     float64
}

type BotFactory func() domain.PlayerController

type BenchmarkRival struct {
	Name    string
	Factory BotFactory
}

type promotedBenchmarkCandidate struct {
	Name   string
	Config bot.BotConfig
	Rank   int
}

var benchmarkFactoryRegistry = map[string]struct{}{}

func makeHeuristicBotFactory(cfg bot.BotConfig) BotFactory {
	return func() domain.PlayerController {
		return bot.NewHeuristicBot(cfg, 0)
	}
}

func makeNamedHeuristicRival(name string, cfg bot.BotConfig) BenchmarkRival {
	return BenchmarkRival{Name: name, Factory: makeHeuristicBotFactory(cfg)}
}

func benchmarkConfigKey(cfg bot.BotConfig) string {
	return fmt.Sprintf("%#v", cfg)
}

func benchmarkConfigHash(cfg bot.BotConfig) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(benchmarkConfigKey(cfg)))
	return fmt.Sprintf("%016x", h.Sum64())
}

func makeHofRivalName(cfg bot.BotConfig) string {
	return "hof_" + benchmarkConfigHash(cfg)
}

func makeHofRivalNameWithContext(epoch, generation, rank int, cfg bot.BotConfig) string {
	hash := benchmarkConfigHash(cfg)
	if len(hash) > 8 {
		hash = hash[:8]
	}
	return fmt.Sprintf("hof_e%02d_g%02d_r%02d_%s", epoch, generation, rank, hash)
}

func promotedRivalName(candidate promotedBenchmarkCandidate, epoch, generation int) string {
	name := strings.TrimSpace(candidate.Name)
	if name != "" {
		return name
	}
	return makeHofRivalNameWithContext(epoch, generation, candidate.Rank, candidate.Config)
}

func appendUniqueBenchmarkFactories(existing []BenchmarkRival, newConfigs []bot.BotConfig) []BenchmarkRival {
	for _, cfg := range newConfigs {
		key := benchmarkConfigKey(cfg)
		if _, exists := benchmarkFactoryRegistry[key]; exists {
			continue
		}
		benchmarkFactoryRegistry[key] = struct{}{}
		existing = append(existing, makeNamedHeuristicRival(makeHofRivalName(cfg), cfg))
	}
	return existing
}

// DefaultConfig returns the baseline configuration used when no YAML file is supplied.
func DefaultConfig() TrainingConfig {
	return TrainingConfig{
		Epochs:               5,
		Generations:          10,
		PopulationSize:       100,
		TopKeep:              20,
		BenchmarkAddPerEpoch: 5,
		BenchmarkMaxSize:     30,
		MutationRateStart:    0.3,
		MutationRateEnd:      0.08,
		MutationAmountStart:  2.0,
		MutationAmountEnd:    0.6,
		MaxTurns:             180,
		GamesPerEval:         0,
		SeedBots:             []string{"e2g2m69", "e2g3m38"},
		BenchmarkBots:        []string{"default_heuristic"},
		WebListenAddr:        ":8080",
		DashboardPath:        "./web",
		DataJSONPath:         "./web/data.json",
	}
}

// ApplyDefaults fills in any missing values so the trainer can boot safely with partial YAML input.
func ApplyDefaults(cfg *TrainingConfig) {
	if cfg == nil {
		return
	}

	defaults := DefaultConfig()

	if cfg.Epochs <= 0 {
		cfg.Epochs = defaults.Epochs
	}
	if cfg.Generations <= 0 {
		cfg.Generations = defaults.Generations
	}
	if cfg.PopulationSize <= 0 {
		cfg.PopulationSize = defaults.PopulationSize
	}
	if cfg.TopKeep <= 0 {
		cfg.TopKeep = defaults.TopKeep
	}
	if cfg.BenchmarkAddPerEpoch <= 0 {
		cfg.BenchmarkAddPerEpoch = defaults.BenchmarkAddPerEpoch
	}
	if cfg.BenchmarkMaxSize <= 0 {
		cfg.BenchmarkMaxSize = defaults.BenchmarkMaxSize
	}
	if cfg.MutationRateStart <= 0 {
		cfg.MutationRateStart = defaults.MutationRateStart
	}
	if cfg.MutationRateEnd <= 0 {
		cfg.MutationRateEnd = defaults.MutationRateEnd
	}
	if cfg.MutationAmountStart <= 0 {
		cfg.MutationAmountStart = defaults.MutationAmountStart
	}
	if cfg.MutationAmountEnd <= 0 {
		cfg.MutationAmountEnd = defaults.MutationAmountEnd
	}
	if cfg.MaxTurns <= 0 {
		cfg.MaxTurns = defaults.MaxTurns
	}
	if cfg.WebListenAddr == "" {
		cfg.WebListenAddr = defaults.WebListenAddr
	}
	if cfg.DashboardPath == "" {
		cfg.DashboardPath = defaults.DashboardPath
	}
	if cfg.DataJSONPath == "" {
		cfg.DataJSONPath = defaults.DataJSONPath
	}
	if len(cfg.SeedBots) == 0 {
		cfg.SeedBots = defaults.SeedBots
	}
	if len(cfg.BenchmarkBots) == 0 {
		cfg.BenchmarkBots = defaults.BenchmarkBots
	}
	if cfg.TopKeep > cfg.PopulationSize {
		cfg.TopKeep = cfg.PopulationSize
	}
}

// mutationValueForEpoch interpolates a value between its start and end points across the configured epoch span.
func mutationValueForEpoch(start, end float64, epoch, totalEpochs int) float64 {
	if totalEpochs <= 1 {
		return end
	}
	if epoch <= 1 {
		return start
	}
	if epoch >= totalEpochs {
		return end
	}
	t := float64(epoch-1) / float64(totalEpochs-1)
	return start + (end-start)*t
}

func (cfg TrainingConfig) mutationRateForEpoch(epoch int) float64 {
	return mutationValueForEpoch(cfg.MutationRateStart, cfg.MutationRateEnd, epoch, cfg.Epochs)
}

func (cfg TrainingConfig) mutationAmountForEpoch(epoch int) float64 {
	return mutationValueForEpoch(cfg.MutationAmountStart, cfg.MutationAmountEnd, epoch, cfg.Epochs)
}

func resolveGamesPerEval(cfg TrainingConfig, benchmarks []BenchmarkRival) int {
	if cfg.GamesPerEval > 0 {
		return cfg.GamesPerEval
	}
	return len(benchmarks) * 2
}

func applyEpochBenchmarkPromotion(benchmarks []BenchmarkRival, cfg TrainingConfig, epoch, generation int, candidates []promotedBenchmarkCandidate) []BenchmarkRival {
	if len(candidates) == 0 {
		return benchmarks
	}

	added := 0
	for _, candidate := range candidates {
		if added >= cfg.BenchmarkAddPerEpoch {
			break
		}
		cfgCandidate := candidate.Config
		key := benchmarkConfigKey(cfgCandidate)
		if _, exists := benchmarkFactoryRegistry[key]; exists {
			continue
		}
		benchmarkFactoryRegistry[key] = struct{}{}
		name := promotedRivalName(candidate, epoch, generation)
		benchmarks = append(benchmarks, makeNamedHeuristicRival(name, cfgCandidate))
		added++
	}

	if cfg.BenchmarkMaxSize > 0 && len(benchmarks) > cfg.BenchmarkMaxSize {
		benchmarks = benchmarks[:cfg.BenchmarkMaxSize]
	}
	return benchmarks
}

func (cfg TrainingConfig) gamesPerEvalForEpoch(epoch int, benchmarks []BenchmarkRival) int {
	if cfg.GamesPerEval > 0 {
		return cfg.GamesPerEval
	}
	baseGames := len(benchmarks) * 2
	if baseGames == 0 {
		return 0
	}
	if cfg.Epochs <= 1 {
		return baseGames
	}
	progress := float64(epoch-1) / float64(cfg.Epochs-1)
	rivalBoost := int(math.Round(float64(len(benchmarks)) * progress))
	return (len(benchmarks) + rivalBoost) * 2
}

// LoadConfig reads the YAML trainer configuration and returns a validated runtime configuration.
func LoadConfig(path string) (TrainingConfig, error) {
	cfg := DefaultConfig()
	if path == "" {
		ApplyDefaults(&cfg)
		return cfg, nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return TrainingConfig{}, err
	}

	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return TrainingConfig{}, err
	}

	ApplyDefaults(&cfg)
	return cfg, nil
}

// heuristicSeedConfigs enumerates the known handcrafted heuristics that can be used as base seeds.
func heuristicSeedConfigs() map[string]func() bot.BotConfig {
	return map[string]func() bot.BotConfig{
		"aggressive": bot.Bot_Aggressive,
		"defensive":  bot.Bot_Defensive,
		"balanced":   bot.Bot_Balanced,
		"chaotic":    bot.Bot_Chaotic,
		"e6g5m74":    bot.Bot_E6G5M74,
		"e6g5m81":    bot.Bot_E6G5M81,
		"e6g5m41":    bot.Bot_E6G5M41,
		"e6g5m39":    bot.Bot_E6G5M39,
		"e6g5m86":    bot.Bot_E6G5M86,
		"e1g3m41":    bot.Bot_E1G3M41,
		"e2g3m99":    bot.Bot_E2G3M99,
		"e1g1m78":    bot.Bot_E1G1M78,
		"e2g4m40":    bot.Bot_E2G4M40,
		"e5g8m26":    bot.Bot_E5G8M26,
		"e4g3m81":    bot.Bot_E4G3M81,
		"e4g4m55":    bot.Bot_E4G4M55,
		"e4g4m45":    bot.Bot_E4G4M45,
		"e4g4m78":    bot.Bot_E4G4M78,
		"e4g4m79":    bot.Bot_E4G4M79,
		"e4g10m52":   bot.Bot_E4G10M52,
		"e5g4m77":    bot.Bot_E5G4M77,
		"e5g4m80":    bot.Bot_E5G4M80,
		"e6g4m92":    bot.Bot_E6G4M92,
		"e6g4m33":    bot.Bot_E6G4M33,
	}
}

// resolveNamedConfig converts a YAML bot name into a concrete bot configuration.
func resolveNamedConfig(name string) (bot.BotConfig, bool) {
	key := strings.ToLower(strings.TrimSpace(name))
	if key == "" {
		return bot.BotConfig{}, false
	}
	if key == "default_heuristic" {
		return bot.DefaultBotConfig(), true
	}
	if factory, ok := heuristicSeedConfigs()[key]; ok {
		return factory(), true
	}
	return bot.BotConfig{}, false
}

// buildSeedPopulation creates the initial population from the configured base heuristics.
func buildSeedPopulation(seedNames []string) ([]bot.BotConfig, []string) {
	seedConfigs := make([]bot.BotConfig, 0, len(seedNames))
	seedIDs := make([]string, 0, len(seedNames))

	for _, name := range seedNames {
		cfg, ok := resolveNamedConfig(name)
		if !ok {
			fmt.Printf("Warning: seed bot '%s' is not recognized; using default heuristic config.\n", name)
			cfg = bot.DefaultBotConfig()
		}
		seedConfigs = append(seedConfigs, cfg)
		seedIDs = append(seedIDs, strings.TrimSpace(name))
	}

	if len(seedConfigs) == 0 {
		seedConfigs = append(seedConfigs, bot.DefaultBotConfig())
		seedIDs = append(seedIDs, "default_heuristic")
	}

	return seedConfigs, seedIDs
}

// buildBenchmarkFactories resolves benchmark names into bot factories used for evaluation.
func buildInitialPopulation(seedConfigs []bot.BotConfig, seedIDs []string, popSize int, mutationRate float64, mutationAmount float64) []*Individual {
	population := make([]*Individual, popSize)
	seedCount := len(seedConfigs)
	if seedCount == 0 {
		seedConfigs = []bot.BotConfig{bot.DefaultBotConfig()}
		seedIDs = []string{"default_heuristic"}
		seedCount = 1
	}

	for i := 0; i < seedCount && i < popSize; i++ {
		seedID := ""
		if i < len(seedIDs) {
			seedID = strings.TrimSpace(seedIDs[i])
		}
		if seedID == "" {
			seedID = fmt.Sprintf("seed_%d", i+1)
		}

		population[i] = &Individual{
			ID:     seedID,
			Config: seedConfigs[i],
		}
	}

	for i := seedCount; i < popSize; i++ {
		parentSeed := seedConfigs[rand.Intn(seedCount)]
		mutatedConfig := mutateConfig(parentSeed, mutationRate, mutationAmount)
		population[i] = &Individual{
			ID:     fmt.Sprintf("E1G1M%d", i+1),
			Config: mutatedConfig,
		}
	}

	return population
}

// buildBenchmarkFactories resolves benchmark names into bot factories used for evaluation.
func buildBenchmarkFactories(names []string) []BenchmarkRival {
	benchmarks := make([]BenchmarkRival, 0, len(names))
	benchmarkFactoryRegistry = make(map[string]struct{})

	for _, name := range names {
		n := strings.ToLower(strings.TrimSpace(name))
		switch n {
		case "default_heuristic":
			cfg := bot.DefaultBotConfig()
			benchmarkFactoryRegistry[benchmarkConfigKey(cfg)] = struct{}{}
			benchmarks = append(benchmarks, makeNamedHeuristicRival("default_heuristic", cfg))
		case "random":
			benchmarks = append(benchmarks, BenchmarkRival{Name: "random", Factory: func() domain.PlayerController {
				return bot.NewRandomBot(0.3)
			}})
		case "minimax_1":
			benchmarks = append(benchmarks, BenchmarkRival{Name: "minimax_1", Factory: func() domain.PlayerController {
				return bot.NewMinimaxBot(0, 1)
			}})
		case "minimax_3":
			benchmarks = append(benchmarks, BenchmarkRival{Name: "minimax_3", Factory: func() domain.PlayerController {
				return bot.NewMinimaxBot(0, 3)
			}})
		case "minimax_4":
			benchmarks = append(benchmarks, BenchmarkRival{Name: "minimax_4", Factory: func() domain.PlayerController {
				return bot.NewMinimaxBot(0, 4)
			}})
		default:
			if cfg, ok := resolveNamedConfig(n); ok {
				benchmarkFactoryRegistry[benchmarkConfigKey(cfg)] = struct{}{}
				benchmarks = append(benchmarks, makeNamedHeuristicRival(n, cfg))
				continue
			}
			fmt.Printf("Warning: benchmark '%s' is not recognized; using default heuristic.\n", n)
			cfg := bot.DefaultBotConfig()
			benchmarkFactoryRegistry[benchmarkConfigKey(cfg)] = struct{}{}
			benchmarks = append(benchmarks, makeNamedHeuristicRival("default_heuristic", cfg))
		}
	}

	if len(benchmarks) == 0 {
		cfg := bot.DefaultBotConfig()
		benchmarkFactoryRegistry[benchmarkConfigKey(cfg)] = struct{}{}
		benchmarks = []BenchmarkRival{makeNamedHeuristicRival("default_heuristic", cfg)}
	}
	return benchmarks
}

// mutateConfig applies small random changes to a bot genome while keeping values within safe bounds.
func mutateConfig(c bot.BotConfig, mutationRate float64, mutationAmount float64) bot.BotConfig {
	newConfig := c

	mutateStruct := func(s interface{}) {
		v := reflect.ValueOf(s).Elem()
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			if field.Kind() == reflect.Float64 {
				if rand.Float64() < mutationRate {
					delta := (rand.Float64()*2 - 1) * mutationAmount
					newVal := field.Float() + delta

					if newVal < -50 {
						newVal = -50
					} else if newVal > 50 {
						newVal = 50
					}

					roundedVal := math.Round(newVal*100) / 100
					field.SetFloat(roundedVal)
				}
			}
		}
	}

	mutateStruct(&newConfig.NormalWeights)
	mutateStruct(&newConfig.PanicWeights)

	if rand.Float64() < mutationRate {
		newConfig.PanicThreshold += rand.Intn(3) - 1
		if newConfig.PanicThreshold < -3 {
			newConfig.PanicThreshold = -3
		} else if newConfig.PanicThreshold > 8 {
			newConfig.PanicThreshold = 8
		}
	}

	return newConfig
}

// main parses the runtime arguments and starts the genetic training process.
func main() {
	flagConfigPath := flag.String("config", "", "Path to the YAML trainer configuration file.")
	flag.Parse()

	cfg, err := LoadConfig(*flagConfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load trainer configuration: %v\n", err)
		os.Exit(1)
	}

	benchmarks := buildBenchmarkFactories(cfg.BenchmarkBots)
	_ = benchmarks

	runTraining(cfg)
}

// runTraining executes the evolutionary loop over epochs and generations using the configured population.
func runTraining(cfg TrainingConfig) {
	rand.Seed(time.Now().UnixNano())

	seedConfigs, seedIDs := buildSeedPopulation(cfg.SeedBots)
	hallOfFameConfigs := make([]bot.BotConfig, 0, len(seedConfigs))
	hallOfFameConfigs = append(hallOfFameConfigs, seedConfigs...)
	benchmarks := buildBenchmarkFactories(cfg.BenchmarkBots)
	gamesPerEval := resolveGamesPerEval(cfg, benchmarks)

	epochs := cfg.Epochs
	generations := cfg.Generations
	popSize := cfg.PopulationSize
	topKeep := cfg.TopKeep
	maxTurns := cfg.MaxTurns

	population := buildInitialPopulation(seedConfigs, seedIDs, popSize, cfg.MutationRateStart, cfg.MutationAmountStart)

	if len(population) == 0 {
		population = []*Individual{{ID: "E1G1D1", Config: seedConfigs[0]}}
	}

	dash := telemetry.NewDashboardServer(cfg.DashboardPath, cfg.DataJSONPath)
	dash.StartAsync(cfg.WebListenAddr)

	numCPUs := runtime.NumCPU()
	fmt.Printf("=== STARTING GENETIC TRAINING (%d CPUs) ===\n", numCPUs)
	fmt.Printf("Population: %d | Epochs: %d | Generations: %d | Strategy: CPU worker pool\n\n", popSize, epochs, generations)

	for epoch := 1; epoch <= epochs; epoch++ {
		epochMutationRate := cfg.mutationRateForEpoch(epoch)
		epochMutationAmount := cfg.mutationAmountForEpoch(epoch)
		gamesPerEval = resolveGamesPerEval(cfg, benchmarks)
		fmt.Printf("[Epoch %02d] active benchmark count: %d | games per evaluation: %d\n", epoch, len(benchmarks), gamesPerEval)
		var bestAvgFitnessInEpoch float64 = 0.0
		consecutiveDropCount := 0
		lastCompletedGeneration := 1

		for gen := 1; gen <= generations; gen++ {
			genStartTime := time.Now()
			fmt.Printf("[Gen %02d] Evaluating %d bots in parallel...\n", gen, popSize)
			rivalMetrics := evaluatePopulationParallel(population, benchmarks, numCPUs, gamesPerEval, maxTurns)

			sort.Slice(population, func(i, j int) bool {
				return population[i].Fitness > population[j].Fitness
			})

			var sumFitness float64
			minFit := population[0].Fitness
			maxFit := population[0].Fitness
			botMetrics := make([]telemetry.BotMetric, len(population))

			for c := 0; c < popSize; c++ {
				fit := population[c].Fitness
				sumFitness += fit
				if fit > maxFit {
					maxFit = fit
				}
				if fit < minFit {
					minFit = fit
				}

				best := population[c]
				botMetrics[c] = telemetry.BotMetric{
					ID:          best.ID,
					Rank:        c + 1,
					Fitness:     best.Fitness,
					Wins:        best.Wins,
					GamesPlayed: best.GamesPlayed,
					Config: telemetry.BotConfig{
						PanicThreshold: best.Config.PanicThreshold,
						NormalWeights:  telemetry.BotWeights(best.Config.NormalWeights),
						PanicWeights:   telemetry.BotWeights(best.Config.PanicWeights),
					},
				}
			}

			avgFit := 0.0
			if len(population) > 0 {
				avgFit = sumFitness / float64(len(population))
			}

			var sumTopFit float64
			topCount := min(topKeep, len(population))
			for i := 0; i < topCount; i++ {
				sumTopFit += population[i].Fitness
			}

			topAverageWinRate := 0.0
			if topCount > 0 {
				topAverageWinRate = sumTopFit / float64(topCount)
			}

			genSummary := telemetry.GenerationSummary{
				Epoch:             epoch,
				Generation:        gen,
				MaxFitness:        maxFit,
				AvgFitness:        avgFit,
				MinFitness:        minFit,
				TopAverageWinRate: topAverageWinRate,
				Bots:              botMetrics,
				Rivals:            rivalMetrics,
			}
			dash.RecordGeneration(genSummary)

			fmt.Printf("[Gen %02d completed in %v]\n",
				gen,
				time.Since(genStartTime).Round(time.Millisecond),
			)
			lastCompletedGeneration = gen

			if gen == generations {
				break
			}

			nextGen := make([]*Individual, popSize)
			for i := 0; i < topKeep; i++ {
				nextGen[i] = &Individual{
					ID:     population[i].ID,
					Config: population[i].Config,
				}
			}

			for i := topKeep; i < popSize; i++ {
				parent := population[rand.Intn(topKeep)]
				mutatedConfig := mutateConfig(parent.Config, epochMutationRate*0.83, epochMutationAmount*0.83)
				nextGen[i] = &Individual{
					ID:     fmt.Sprintf("E%dG%dM%d", epoch, gen+1, i+1),
					Config: mutatedConfig,
				}
			}

			lastTopWinrate := population[topKeep-1].Fitness

			if gen > 2 && lastTopWinrate >= 1.0 {
				fmt.Printf("🚀 [Epoch %d] Top %d reached 100%% in generation %d; promoting to next epoch.\n", epoch, topKeep, gen)
				break
			}

			if avgFit > bestAvgFitnessInEpoch {
				bestAvgFitnessInEpoch = avgFit
				consecutiveDropCount = 0
			} else {
				consecutiveDropCount++
				if consecutiveDropCount >= 2 {
					fmt.Printf("⚠️ [Epoch %d] Fitness degraded in generation %d (average fell %d times). Stopping the epoch.\n", epoch, gen, consecutiveDropCount)
					break
				}
			}

			population = nextGen
		}

		fmt.Printf("\n🏆 === EPOCH %d FINISHED ===\n", epoch)

		promotedCount := min(topKeep, len(population))
		promotedCandidates := make([]promotedBenchmarkCandidate, 0, promotedCount)
		for i := 0; i < promotedCount; i++ {
			topConfig := population[i].Config
			hallOfFameConfigs = append(hallOfFameConfigs, topConfig)
			promotedCandidates = append(promotedCandidates, promotedBenchmarkCandidate{
				Name:   population[i].ID,
				Config: topConfig,
				Rank:   i + 1,
			})
		}
		benchmarks = applyEpochBenchmarkPromotion(benchmarks, cfg, epoch, lastCompletedGeneration, promotedCandidates)

		newPopulation := make([]*Individual, popSize)
		for i := 0; i < topKeep && i < popSize; i++ {
			newPopulation[i] = &Individual{
				ID:     population[i].ID,
				Config: population[i].Config,
			}
		}

		for i := topKeep; i < popSize; i++ {
			parentSeed := hallOfFameConfigs[rand.Intn(len(hallOfFameConfigs))]
			mutatedConfig := mutateConfig(parentSeed, epochMutationRate, epochMutationAmount)
			newPopulation[i] = &Individual{
				ID:     fmt.Sprintf("E%dG%dM%d", epoch+1, 1, i+1),
				Config: mutatedConfig,
			}
		}

		population = newPopulation
	}

	fmt.Println("\n=======================================================")
	fmt.Println("🎉 Training finished successfully!")
	fmt.Println("🌐 The web dashboard remains available at http://localhost:8080")
	fmt.Println("👉 Press ENTER in this console to close the server...")
	fmt.Println("=======================================================")

	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

// evaluatePopulationParallel measures every individual in parallel against the configured benchmark bots.
func evaluatePopulationParallel(pop []*Individual, benchmarks []BenchmarkRival, numWorkers, gamesPerEval, maxTurns int) []telemetry.RivalMetric {
	jobs := make(chan *Individual, len(pop))
	var wg sync.WaitGroup

	type rivalTally struct {
		wins   int
		losses int
		draws  int
		games  int
	}

	mergedRivalStats := make([]rivalTally, len(benchmarks))
	var statsMu sync.Mutex

	totalInds := len(pop)
	var completedInds int64

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ind := range jobs {
				rivalStats := evalIndividual(ind, benchmarks, gamesPerEval, maxTurns)

				statsMu.Lock()
				for i := range mergedRivalStats {
					mergedRivalStats[i].wins += rivalStats[i].wins
					mergedRivalStats[i].losses += rivalStats[i].losses
					mergedRivalStats[i].draws += rivalStats[i].draws
					mergedRivalStats[i].games += rivalStats[i].games
				}
				statsMu.Unlock()

				current := atomic.AddInt64(&completedInds, 1)
				percent := float64(current) / float64(totalInds) * 100

				fmt.Printf("\r ⚡ Evaluating population: [%2.0f%%] (%d/%d bots processed)", percent, current, totalInds)
			}
		}()
	}

	for _, ind := range pop {
		jobs <- ind
	}
	close(jobs)

	wg.Wait()
	fmt.Println()

	rivalMetrics := make([]telemetry.RivalMetric, 0, len(benchmarks))
	for i, rival := range benchmarks {
		stats := mergedRivalStats[i]
		metric := telemetry.RivalMetric{
			Name:   rival.Name,
			Wins:   stats.wins,
			Losses: stats.losses,
			Draws:  stats.draws,
			Games:  stats.games,
		}
		if stats.games > 0 {
			metric.WinRate = float64(stats.wins) / float64(stats.games)
			metric.LossRate = float64(stats.losses) / float64(stats.games)
			metric.DrawRate = float64(stats.draws) / float64(stats.games)
		}
		rivalMetrics = append(rivalMetrics, metric)
	}

	sort.Slice(rivalMetrics, func(i, j int) bool {
		if rivalMetrics[i].WinRate == rivalMetrics[j].WinRate {
			return rivalMetrics[i].Games > rivalMetrics[j].Games
		}
		return rivalMetrics[i].WinRate > rivalMetrics[j].WinRate
	})

	return rivalMetrics
}

// evalIndividual evaluates one candidate against the benchmark set and stores its fitness.
func evalIndividual(ind *Individual, benchmarks []BenchmarkRival, totalGames, maxTurns int) []struct {
	wins   int
	losses int
	draws  int
	games  int
} {
	ind.Wins = 0
	ind.GamesPlayed = totalGames
	rivalStats := make([]struct {
		wins   int
		losses int
		draws  int
		games  int
	}, len(benchmarks))

	botTested := bot.NewHeuristicBot(ind.Config, 0)

	for g := 0; g < totalGames; g++ {
		numBenchmarks := len(benchmarks)
		rivalIndex := (g / 2) % numBenchmarks
		rivalBot := benchmarks[rivalIndex].Factory()
		rivalStats[rivalIndex].games++

		swapPositions := (g%2 == 1)
		winner := simulateGame(botTested, rivalBot, maxTurns, swapPositions)

		candidateWon := (winner == 1 && !swapPositions) || (winner == 2 && swapPositions)
		candidateLost := (winner == 2 && !swapPositions) || (winner == 1 && swapPositions)

		if candidateWon {
			ind.Wins++
			rivalStats[rivalIndex].losses++
		} else if candidateLost {
			rivalStats[rivalIndex].wins++
		} else {
			rivalStats[rivalIndex].draws++
		}
	}

	ind.Fitness = float64(ind.Wins) / float64(ind.GamesPlayed)
	return rivalStats
}

// simulateGame runs a single match between two controllers and returns the winning player id.
func simulateGame(b1, b2 domain.PlayerController, maxTurns int, swapPositions bool) int {
	g := domain.NewGame()
	configs := []domain.PlayerConfig{
		{ID: 1},
		{ID: 2},
	}
	g.InitBoard(configs)

	var botP1, botP2 domain.PlayerController
	if !swapPositions {
		botP1, botP2 = b1, b2
	} else {
		botP1, botP2 = b2, b1
	}

	for turn := 0; turn < maxTurns; turn++ {
		p := g.ActivePlayer()
		if p == nil {
			break
		}

		var activeBot domain.PlayerController
		if p.ID == 1 {
			activeBot = botP1
		} else {
			activeBot = botP2
		}

		action := activeBot.GetAction(g, p)

		if action.Type == domain.ActionMove {
			p.GridX = action.TargetX
			p.GridY = action.TargetY
		} else if action.Type == domain.ActionPlaceWall {
			if !g.PlaceWall(action.TargetX, action.TargetY, action.WallOrientation, p) {
				if p.ID == 1 {
					return 2
				}
				return 1
			}
		}

		if g.CheckWin(p) {
			return p.ID
		}

		g.NextTurn()
	}

	return 0
}
