package ebitenui

import (
	"image/color"
	"testing"

	"quoridor/internal/domain"
)

func TestGameAdapter_RejectsInvalidBotWallActionAndAdvancesTurn(t *testing.T) {
	g := domain.NewGame()
	g.InitBoard([]domain.PlayerConfig{
		{ID: 1, Color: color.RGBA{255, 0, 0, 255}},
		{ID: 2, Color: color.RGBA{0, 0, 255, 255}},
	})
	g.TurnIndex = 1
	g.State = domain.StatePlaying

	adapter := NewGameAdapter(g)
	adapter.Controllers = map[int]domain.PlayerController{
		2: &stubController{action: domain.PlayerAction{Type: domain.ActionPlaceWall, TargetX: 8, TargetY: 2, WallOrientation: domain.WallHorizontal}},
	}

	beforeTurn := g.TurnIndex
	adapter.updatePlaying()

	if g.TurnIndex == beforeTurn {
		t.Fatalf("expected turn to advance after invalid wall action, before=%d after=%d", beforeTurn, g.TurnIndex)
	}
}

func TestHumanController_IgnoresInputWhenNotActivePlayer(t *testing.T) {
	g := domain.NewGame()
	g.InitBoard([]domain.PlayerConfig{
		{ID: 1, Color: color.RGBA{255, 0, 0, 255}},
		{ID: 2, Color: color.RGBA{0, 0, 255, 255}},
	})
	g.TurnIndex = 1

	adapter := NewGameAdapter(g)
	controller := &EbitenHumanController{UI: adapter}
	adapter.Controllers = map[int]domain.PlayerController{1: controller, 2: controller}

	player1 := g.Players[0]
	player2 := g.Players[1]
	if g.ActivePlayer().ID != player2.ID {
		t.Fatalf("test setup expected player 2 to be active")
	}

	action := controller.GetAction(g, player1)
	if action.Type != domain.ActionNone {
		t.Fatalf("expected human input to be ignored when not active, got %+v", action)
	}
}

type stubController struct {
	action domain.PlayerAction
}

func (s *stubController) GetAction(g *domain.Game, p *domain.Player) domain.PlayerAction {
	return s.action
}
