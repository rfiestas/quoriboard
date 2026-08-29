package main

import (
	"fmt"
	"time"

	"quoridor/internal/adapters/bot"
	"quoridor/internal/domain"
)

type BenchmarkStats struct {
	TotalGames     int
	TotalTurns     int
	TotalGameTime  time.Duration
	TotalThinkTime time.Duration
	MaxThinkTime   time.Duration
	P1Wins         int
	P2Wins         int
	Draws          int
}

func main() {
	configP1 := bot.DefaultBotConfig()
	configP2 := bot.DefaultBotConfig()

	// Bot 1 usará los pesos por defecto.
	// Bot 2 usará una ligera variación para evitar partidas espejo exactas.
	configP2.NormalWeights.RivalDistanceWeight += 2.0

	b1 := bot.NewHeuristicBot(configP1, 0)
	//b2 := bot.NewHeuristicBot(configP2, 0)
	b2 := bot.NewMinimaxBot(0, 3)

	totalGames := 10
	maxTurns := 120

	fmt.Printf("Iniciando Benchmark: %d partidas (Secuencial)...\n", totalGames)

	stats := runBenchmark(b1, b2, totalGames, maxTurns)

	fmt.Println("\n================ REPORTE DE RENDIMIENTO ================")
	fmt.Printf("Partidas Jugadas: %d\n", stats.TotalGames)
	fmt.Printf("Ganador P1: %d | Ganador P2: %d | Empates: %d\n", stats.P1Wins, stats.P2Wins, stats.Draws)
	fmt.Println("--------------------------------------------------------")
	fmt.Printf("Turnos Totales:   %d\n", stats.TotalTurns)
	fmt.Printf("Tiempo Total:     %v\n", stats.TotalGameTime)
	fmt.Printf("Tiempo/Partida:   %v\n", stats.TotalGameTime/time.Duration(stats.TotalGames))
	fmt.Printf("Tiempo/Turno:     %v (Media de GetAction)\n", stats.TotalThinkTime/time.Duration(stats.TotalTurns))
	fmt.Printf("Turno Más Lento:  %v (Pico máximo de GetAction)\n", stats.MaxThinkTime)
	fmt.Println("========================================================")
}

//field Controllers map[int]domain.PlayerController

func runBenchmark(b1, b2 domain.PlayerController, totalGames, maxTurns int) BenchmarkStats {
	stats := BenchmarkStats{TotalGames: totalGames}
	startAll := time.Now()

	for i := 0; i < totalGames; i++ {
		swapPositions := (i%2 == 1)

		g := domain.NewGame()
		configs := []domain.PlayerConfig{
			{ID: 1},
			{ID: 2},
		}
		g.InitBoard(configs)

		// Mapear qué bot físico (b1 o b2) maneja a qué jugador del dominio (P1 o P2)
		var botP1, botP2 domain.PlayerController
		if !swapPositions {
			botP1, botP2 = b1, b2
		} else {
			botP1, botP2 = b2, b1
		}

		gameEnded := false

		for turn := 0; turn < maxTurns; turn++ {
			stats.TotalTurns++
			p := g.ActivePlayer()
			if p == nil {
				break
			}

			// Identificar si le toca a b1 o b2
			currentBot := botP1
			if p.ID == 2 {
				currentBot = botP2
			}

			// --- MEDIR TIEMPO DE DECISIÓN ---
			startThink := time.Now()
			action := currentBot.GetAction(g, p)
			thinkTime := time.Since(startThink)

			stats.TotalThinkTime += thinkTime
			if thinkTime > stats.MaxThinkTime {
				stats.MaxThinkTime = thinkTime
			}

			// --- APLICAR ACCIÓN ---
			if action.Type == domain.ActionMove {
				p.GridX = action.TargetX
				p.GridY = action.TargetY

				// Comprobación de victoria idéntica a tu adaptador de Ebitengine
				hasWon := (p.TargetY != -1 && p.GridY == p.TargetY) ||
					(p.TargetX != -1 && p.GridX == p.TargetX)

				if hasWon {
					recordWin(&stats, currentBot, b1, b2)
					gameEnded = true
					break
				}

			} else if action.Type == domain.ActionPlaceWall {
				if !g.PlaceWall(action.TargetX, action.TargetY, action.WallOrientation, p) {
					// Si el bot intenta colocar una pared ilegal, pierde la partida
					recordOpponentWin(&stats, currentBot, b1, b2)
					gameEnded = true
					break
				}
			}

			g.NextTurn()
		}

		if !gameEnded {
			stats.Draws++
		}
	}

	stats.TotalGameTime = time.Since(startAll)
	return stats
}

// Mapean la victoria o derrota directamente al puntero del bot (b1 o b2)
func recordWin(stats *BenchmarkStats, winner, b1, b2 domain.PlayerController) {
	if winner == b1 {
		stats.P1Wins++
	} else {
		stats.P2Wins++
	}
}

func recordOpponentWin(stats *BenchmarkStats, loser, b1, b2 domain.PlayerController) {
	if loser == b1 {
		stats.P2Wins++
	} else {
		stats.P1Wins++
	}
}
