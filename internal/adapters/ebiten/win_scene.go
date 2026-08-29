package ebitenui

import (
	"fmt"

	"quoridor/internal/domain"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

func (a *GameAdapter) updateWin() {
	screenWidth := a.screenWidth
	screenHeight := a.screenHeight

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		cx, cy := ebiten.CursorPosition()
		// "Main Menu" button bounds (x: 300, y: 340, w: 200, h: 45)
		if cx >= screenWidth-470 && cx <= screenWidth-90 && cy >= screenHeight-180 && cy <= screenHeight-50 {
			a.Winner = nil
			a.DomainGame.State = domain.StateMenu
		}
		//vector.FillRect(screen, float32(screenWidth-470), float32(screenHeight-180), 380, 130, color.RGBA{0, 0, 0, 180}, true)

	}
}

func (a *GameAdapter) drawWinOverlay(screen *ebiten.Image) {
	screenWidth := float64(a.screenWidth)
	screenHeight := float64(a.screenHeight)

	// Dark semi-transparent background overlay
	//vector.FillRect(screen, 0, 0, float32(screenWidth), float32(a.screenHeight), color.RGBA{0, 0, 0, 180}, true)

	if a.Winner != nil {
		// Winner color circle badge
		//vector.FillCircle(screen, 400, 210, 20, a.Winner.Color, true)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(screenWidth/2+60, 0)
		screen.DrawImage(a.assets.images.winScene, op)

		// Victory text
		drawRealisticText(screen, fmt.Sprintf("PLAYER %d WINS!", a.Winner.ID), screenWidth-480, 70, a.assets.fonts.mainFontBig, ColorBlack)

		drawRealisticText(screen, fmt.Sprintf("(%s)", a.controllerLabelForPlayer(a.Winner.ID)), screenWidth-350, 140, a.assets.fonts.mainFontSmall, ColorBlue)
	}

	// Main Menu Button
	drawRealisticText(screen, "MAIN MENU", screenWidth-410, screenHeight-160, a.assets.fonts.mainFontBig, ColorBlack)
}
