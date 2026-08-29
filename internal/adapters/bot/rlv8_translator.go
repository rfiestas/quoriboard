package bot

import (
	"quoridor/internal/domain"
	"quoridor/internal/rlv8/translator"
)

const (
	ActionSpaceSize = translator.ActionSpaceSize
	ObservationSize = translator.ObservationSize
	maxTurns        = translator.MaxTurns
)

// RealAction aplica la simetría autoinversa para el Jugador 2
func RealAction(player *domain.Player, action int) int {
	return translator.RealAction(player, action)
}

// BuildObservation genera el vector de 491 floats exactamente como en el entrenamiento
func BuildObservation(g *domain.Game, current *domain.Player, turns int) []float32 {
	return translator.BuildObservation(g, current, turns)
}

// BuildActionMask genera la máscara de 136 booleanos
func BuildActionMask(g *domain.Game, player *domain.Player) []bool {
	return translator.BuildActionMask(g, player)
}

// DecodeAction convierte el índice del modelo (0-135) a la PlayerAction nativa de Go
func DecodeAction(g *domain.Game, player *domain.Player, action int) domain.PlayerAction {
	return translator.DecodeAction(g, player, action)
}
