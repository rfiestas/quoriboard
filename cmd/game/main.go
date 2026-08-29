package main

import (
	"flag"
	"log"

	ebitenui "quoridor/internal/adapters/ebiten"
	"quoridor/internal/domain"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	rlV8 := flag.Bool("rl-v8", false, "Enable the trained RL V8 Python bot in the game menu")
	checkpoint := flag.String("rl-v8-checkpoint", "models/ppo_a136_league/checkpoints/ppo_a136_league_13007080_steps.zip", "RL V8 checkpoint")
	python := flag.String("rl-v8-python", ".venv/Scripts/python.exe", "Python executable for RL V8 inference")
	flag.Parse()

	// Initialize Domain
	domainGame := domain.NewGame()

	// Initialize Ebiten UI Adapter
	uiAdapter := ebitenui.NewGameAdapter(domainGame)
	defer uiAdapter.CloseControllers()
	uiAdapter.RLV8PythonEnabled = *rlV8
	uiAdapter.RLV8Checkpoint = *checkpoint
	uiAdapter.RLV8Python = *python
	uiAdapter.LoadResources()

	ebiten.SetWindowSize(1280, 720)
	ebiten.SetWindowTitle("Quoriboard")

	if err := ebiten.RunGame(uiAdapter); err != nil {
		log.Fatal(err)
	}
}
