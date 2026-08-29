package ebitenui

import (
	"fmt"

	"quoridor/internal/adapters/bot"
	"quoridor/internal/domain"
)

type ControllerType string

const (
	ControllerHuman        ControllerType = "human"
	ControllerHeuristic    ControllerType = "heuristic"
	ControllerHeuristicTop ControllerType = "heuristic-top-e4g3m81"
	ControllerMiniMax2     ControllerType = "minimax2"
	ControllerMiniMax3     ControllerType = "minimax3"
	ControllerMiniMax4     ControllerType = "minimax4"
	ControllerRandom       ControllerType = "random"
	ControllerRLV8Python   ControllerType = "rlv8-python"
	// ControllerRLV8Go is the Go-native RL V8 policy bot. Disabled for now, re-enable later.
	ControllerRLV8Go ControllerType = "rlv8-go"
)

type controllerDefinition struct {
	Type       ControllerType
	Label      string
	Selectable bool
	Create     func(*GameAdapter) (domain.PlayerController, error)
}

var controllerCatalog = []controllerDefinition{
	{
		Type:       ControllerHuman,
		Label:      "Human",
		Selectable: true,
		Create: func(adapter *GameAdapter) (domain.PlayerController, error) {
			return &EbitenHumanController{UI: adapter}, nil
		},
	},
	{
		Type:       ControllerHeuristic,
		Label:      "Heuristic",
		Selectable: true,
		Create: func(*GameAdapter) (domain.PlayerController, error) {
			return bot.NewHeuristicBot(bot.DefaultBotConfig(), 0), nil
		},
	},
	{
		Type:       ControllerHeuristicTop,
		Label:      "Heuristic Top5",
		Selectable: true,
		Create: func(*GameAdapter) (domain.PlayerController, error) {
			return bot.NewHeuristicBot(bot.Bot_E4G3M81(), 0), nil
		},
	},
	{
		Type:       ControllerMiniMax2,
		Label:      "MiniMax 2",
		Selectable: true,
		Create: func(*GameAdapter) (domain.PlayerController, error) {
			return bot.NewMinimaxBot(0, 2), nil
		},
	},
	{
		Type:       ControllerMiniMax3,
		Label:      "MiniMax 3",
		Selectable: true,
		Create: func(*GameAdapter) (domain.PlayerController, error) {
			return bot.NewMinimaxBot(0, 3), nil
		},
	},
	{
		Type:       ControllerMiniMax4,
		Label:      "MiniMax 4",
		Selectable: true,
		Create: func(*GameAdapter) (domain.PlayerController, error) {
			return bot.NewMinimaxBot(0, 4), nil
		},
	},
	{
		Type:       ControllerRandom,
		Label:      "Random",
		Selectable: true,
		Create: func(*GameAdapter) (domain.PlayerController, error) {
			return bot.NewRandomBot(0.3), nil
		},
	},
	{
		Type:       ControllerRLV8Python,
		Label:      "RL V8 Python",
		Selectable: true,
		Create: func(adapter *GameAdapter) (domain.PlayerController, error) {
			if !adapter.RLV8PythonEnabled {
				return nil, fmt.Errorf("RL V8 Python is not enabled; start the game with --rl-v8")
			}
			return NewRLV8PythonController(adapter.RLV8Python, adapter.RLV8Checkpoint)
		},
	},
	//RL V8 Go controller disabled for now; re-enable later.
	{
		Type:       ControllerRLV8Go,
		Label:      "RL V8 Go",
		Selectable: true,
		Create: func(*GameAdapter) (domain.PlayerController, error) {
			return bot.NewRLJSONBot()
		},
	},
}

func controllerDefinitionFor(controllerType ControllerType) (controllerDefinition, bool) {
	for _, definition := range controllerCatalog {
		if definition.Type == controllerType {
			return definition, true
		}
	}
	return controllerDefinition{}, false
}

func nextSelectableControllerType(current ControllerType, rlv8PythonEnabled bool) ControllerType {
	currentIndex := -1
	selectable := make([]controllerDefinition, 0, len(controllerCatalog))
	for _, definition := range controllerCatalog {
		if !definition.Selectable || (definition.Type == ControllerRLV8Python && !rlv8PythonEnabled) {
			continue
		}
		if definition.Type == current {
			currentIndex = len(selectable)
		}
		selectable = append(selectable, definition)
	}

	if len(selectable) == 0 {
		return current
	}
	return selectable[(currentIndex+1)%len(selectable)].Type
}

func controllerLabel(controllerType ControllerType) string {
	if definition, ok := controllerDefinitionFor(controllerType); ok {
		return definition.Label
	}
	return string(controllerType)
}

func (a *GameAdapter) createController(controllerType ControllerType) (domain.PlayerController, error) {
	definition, ok := controllerDefinitionFor(controllerType)
	if !ok {
		return nil, fmt.Errorf("tipo de controlador no soportado: %q", controllerType)
	}
	return definition.Create(a)
}

func (a *GameAdapter) controllerLabelForPlayer(playerID int) string {
	for _, slot := range a.MenuConfig {
		if slot.ID == playerID {
			return controllerLabel(slot.ControllerType)
		}
	}
	return "Unknown"
}
