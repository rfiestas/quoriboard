package ebitenui

import (
	"quoridor/internal/domain"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type EbitenHumanController struct {
	UI *GameAdapter
}

func (h *EbitenHumanController) GetAction(g *domain.Game, p *domain.Player) domain.PlayerAction {
	if g == nil || p == nil || g.ActivePlayer() == nil || g.ActivePlayer().ID != p.ID {
		return domain.PlayerAction{Type: domain.ActionNone}
	}

	// Toggle Wall placement mode with 'W' key
	if inpututil.IsKeyJustPressed(ebiten.KeyW) {
		h.UI.WallMode = !h.UI.WallMode
	}

	// Rotate Wall orientation with 'R' or Space
	if inpututil.IsKeyJustPressed(ebiten.KeyR) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		if h.UI.WallOrientation == domain.WallHorizontal {
			h.UI.WallOrientation = domain.WallVertical
		} else {
			h.UI.WallOrientation = domain.WallHorizontal
		}
	}
	// Handle Action based on current mode
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if h.UI.WallMode {
			if h.UI.HoverWallValid {
				return domain.PlayerAction{
					Type:            domain.ActionPlaceWall,
					TargetX:         h.UI.HoverWallX,
					TargetY:         h.UI.HoverWallY,
					WallOrientation: h.UI.WallOrientation,
				}
			}
		} else {
			sx, sy := h.UI.SelectedTileX, h.UI.SelectedTileY
			if sx >= 0 && sx < domain.BoardSize && sy >= 0 && sy < domain.BoardSize {
				var validMoveBuf [8]domain.Move
				validMoveCount := g.GetValidMovesInto(p, &validMoveBuf)
				for i := 0; i < validMoveCount; i++ {
					move := validMoveBuf[i]
					if move.X == sx && move.Y == sy {
						return domain.PlayerAction{
							Type:    domain.ActionMove,
							TargetX: sx,
							TargetY: sy,
						}
					}
				}
			}
		}
	}
	return domain.PlayerAction{Type: domain.ActionNone}
}
