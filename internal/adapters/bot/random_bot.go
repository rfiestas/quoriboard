package bot

import (
	"math/rand"

	"quoridor/internal/domain"
)

/*

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

type RandomBot struct {
	thinkingStart time.Time
	isThinking    bool
	delay         time.Duration
}

func NewRandomBot(delay time.Duration) *RandomBot {
	return &RandomBot{delay: delay}
}

func (b *RandomBot) GetAction(g *domain.Game, p *domain.Player) domain.PlayerAction {
	if !b.isThinking {
		b.isThinking = true
		b.thinkingStart = time.Now()
		return domain.PlayerAction{Type: domain.ActionNone}
	}

	if time.Since(b.thinkingStart) < b.delay {
		return domain.PlayerAction{Type: domain.ActionNone}
	}

	b.isThinking = false

	if p.WallsLeft > 0 && rand.Float32() < 0.5 {
		// Intentar varias posiciones al azar hasta encontrar una legal
		for i := 0; i < 20; i++ {
			wx := rand.Intn(domain.BoardSize - 1)
			wy := rand.Intn(domain.BoardSize - 1)
			orient := domain.WallHorizontal
			if rand.Float32() < 0.5 {
				orient = domain.WallVertical
			}

			if g.CanPlaceWall(wx, wy, orient, p) {
				return domain.PlayerAction{
					Type:            domain.ActionPlaceWall,
					TargetX:         wx,
					TargetY:         wy,
					WallOrientation: orient,
				}
			}
		}
	}

	var validMoveBuf [8]domain.Move
	validMoveCount := g.GetValidMovesInto(p, &validMoveBuf)
	if validMoveCount == 0 {
		return domain.PlayerAction{Type: domain.ActionNone}
	}

	chosen := validMoveBuf[rng.Intn(validMoveCount)]
	return domain.PlayerAction{
		Type:    domain.ActionMove,
		TargetX: chosen.X,
		TargetY: chosen.Y,
	}
}
*/

type RandomBot struct {
	wallProbability float32 // Probabilidad de intentar poner muro (ej: 0.3 = 30%)
}

func NewRandomBot(wallProb float32) *RandomBot {
	return &RandomBot{wallProbability: wallProb}
}

func (b *RandomBot) GetAction(g *domain.Game, p *domain.Player) domain.PlayerAction {
	// 1. Intentar colocar un muro aleatorio según la probabilidad configurada
	if p.WallsLeft > 0 && rand.Float32() < b.wallProbability {
		for i := 0; i < 30; i++ {
			wx := rand.Intn(domain.BoardSize - 1)
			wy := rand.Intn(domain.BoardSize - 1)
			orient := domain.WallHorizontal
			if rand.Float32() < 0.5 {
				orient = domain.WallVertical
			}

			if g.CanPlaceWall(wx, wy, orient, p) {
				return domain.PlayerAction{
					Type:            domain.ActionPlaceWall,
					TargetX:         wx,
					TargetY:         wy,
					WallOrientation: orient,
				}
			}
		}
	}

	// 2. Si no puso muro (o no encontró lugar legal tras 30 intentos), mueve el peón
	var validMoveBuf [8]domain.Move
	validMoveCount := g.GetValidMovesInto(p, &validMoveBuf)
	if validMoveCount > 0 {
		chosen := validMoveBuf[rand.Intn(validMoveCount)]
		return domain.PlayerAction{
			Type:    domain.ActionMove,
			TargetX: chosen.X,
			TargetY: chosen.Y,
		}
	}

	return domain.PlayerAction{Type: domain.ActionNone}
}
