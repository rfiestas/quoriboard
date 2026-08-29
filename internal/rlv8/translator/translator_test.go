package translator

import (
	"testing"

	"quoridor/internal/domain"
)

func TestRealActionPlayer2IsInvolution(t *testing.T) {
	player := &domain.Player{ID: 2}
	for action := 0; action < ActionSpaceSize; action++ {
		if got := RealAction(player, RealAction(player, action)); got != action {
			t.Fatalf("action %d transformed twice = %d", action, got)
		}
	}
}

func TestDecodeActionPlayer2ReflectsWallsVertically(t *testing.T) {
	game := domain.NewGame()
	game.InitBoard([]domain.PlayerConfig{{ID: 1}, {ID: 2}})
	player := game.Players[1]

	for _, test := range []struct {
		action      int
		x, y        int
		orientation domain.WallOrientation
	}{
		{8 + 3*8, 3, 7, domain.WallHorizontal},
		{72 + 5*8 + 2, 5, 5, domain.WallVertical},
	} {
		got := DecodeAction(game, player, test.action)
		if got.Type != domain.ActionPlaceWall || got.TargetX != test.x || got.TargetY != test.y || got.WallOrientation != test.orientation {
			t.Fatalf("action %d decoded as %+v", test.action, got)
		}
	}
}
