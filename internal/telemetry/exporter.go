package telemetry

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
)

type BotWeights struct {
	MyDistanceWeight    float64 `json:"my_distance_weight"`
	RivalDistanceWeight float64 `json:"rival_distance_weight"`
	WallReserveWeight   float64 `json:"wall_reserve_weight"`
	CentralityWeight    float64 `json:"centrality_weight"`
	JumpConcededPenalty float64 `json:"jump_conceded_penalty"`
	FunnelingBonus      float64 `json:"funneling_bonus"`
}

type BotConfig struct {
	PanicThreshold int        `json:"panic_threshold"`
	NormalWeights  BotWeights `json:"normal_weights"`
	PanicWeights   BotWeights `json:"panic_weights"`
}

type BotMetric struct {
	ID          string    `json:"id"`
	Rank        int       `json:"rank"`
	Fitness     float64   `json:"fitness"`
	Wins        int       `json:"wins"`
	GamesPlayed int       `json:"games_played"`
	Config      BotConfig `json:"config"`
}

type RivalMetric struct {
	Name     string  `json:"name"`
	Wins     int     `json:"wins"`
	Losses   int     `json:"losses"`
	Draws    int     `json:"draws"`
	Games    int     `json:"games"`
	WinRate  float64 `json:"win_rate"`
	LossRate float64 `json:"loss_rate"`
	DrawRate float64 `json:"draw_rate"`
}

type GenerationSummary struct {
	Epoch             int           `json:"epoch"`
	Generation        int           `json:"generation"`
	MaxFitness        float64       `json:"max_fitness"`
	AvgFitness        float64       `json:"avg_fitness"`
	MinFitness        float64       `json:"min_fitness"`
	TopAverageWinRate float64       `json:"top_avg_winrate"`
	Bots              []BotMetric   `json:"bots"`
	Rivals            []RivalMetric `json:"rivals"`
}

type TrainingReport struct {
	Generations []GenerationSummary `json:"generations"`
}

type DashboardServer struct {
	mu       sync.Mutex
	report   TrainingReport
	jsonPath string
	webDir   string
}

func NewDashboardServer(webDir string, jsonPath string) *DashboardServer {
	return &DashboardServer{
		report: TrainingReport{
			Generations: make([]GenerationSummary, 0),
		},
		jsonPath: jsonPath,
		webDir:   webDir,
	}
}

// Inicia el servidor HTTP de forma no bloqueante
func (s *DashboardServer) StartAsync(port string) {
	go func() {
		fs := http.FileServer(http.Dir(s.webDir))
		http.Handle("/", fs)

		// Handler para entregar el json generado
		http.HandleFunc("/data.json", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			http.ServeFile(w, r, s.jsonPath)
		})

		fmt.Printf("📊 Dashboard de entrenamiento disponible en: http://localhost%s\n", port)
		if err := http.ListenAndServe(port, nil); err != nil {
			fmt.Printf("Error iniciando servidor web: %v\n", err)
		}
	}()
}

// Añade el resumen de una generación y guarda de forma atómica en disco
func (s *DashboardServer) RecordGeneration(genSummary GenerationSummary) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.report.Generations = append(s.report.Generations, genSummary)

	// Guardado Atómico (escribe a .tmp y luego hace rename)
	data, err := json.MarshalIndent(s.report, "", "  ")
	if err != nil {
		fmt.Printf("Error serializando JSON: %v\n", err)
		return
	}

	tmpFile := s.jsonPath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		fmt.Printf("Error escribiendo temporal JSON: %v\n", err)
		return
	}

	if err := os.Rename(tmpFile, s.jsonPath); err != nil {
		fmt.Printf("Error reemplazando JSON atómico: %v\n", err)
	}
}
