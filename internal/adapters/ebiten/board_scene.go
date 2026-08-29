package ebitenui

import (
	"fmt"
	"image/color"

	"quoridor/internal/domain"

	"github.com/hajimehoshi/ebiten/v2"
)

func (a *GameAdapter) drawBoard(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}

	// Background
	screen.DrawImage(a.assets.images.backgroundBoardScene, op)

	// Grid
	for gx := 0; gx < domain.BoardSize; gx++ {
		for gy := 0; gy < domain.BoardSize; gy++ {
			op := &ebiten.DrawImageOptions{}
			x, y := a.getTilePos(gx, gy)
			op.GeoM.Translate(float64(x-6), float64(y-5))

			if gx == a.SelectedTileX && gy == a.SelectedTileY { //Hover
				op.ColorScale.ScaleWithColor(color.RGBA{255, 255, 255, 180})
			}
			screen.DrawImage(a.assets.images.square, op)

		}
	}

	// DRAW PLACED WALLS
	for _, w := range a.DomainGame.Walls {
		a.renderWall(screen, w.GridX, w.GridY, w.Orientation, true)
	}

	// DRAW HOVER PREVIEW WALL
	if a.WallMode && a.HoverWallX != -1 && a.HoverWallY != -1 {
		a.renderWall(screen, a.HoverWallX, a.HoverWallY, a.WallOrientation, a.HoverWallValid)
	}

	player := a.assets.images.player1
	for i, p := range a.DomainGame.Players {
		px, py := a.getTilePos(p.GridX, p.GridY)
		centerX := float32(px - 10)
		centerY := float32(py - 15)
		op := &ebiten.DrawImageOptions{}
		if a.DomainGame.ActivePlayer().ID == p.ID {
			//op.GeoM.Scale(1.2, 1.2)
			//centerX = centerX - 3
			//centerY = centerY - 3
			//op.ColorScale.ScaleWithColor(color.RGBA{255, 255, 255, 180})
		}
		op.GeoM.Translate(float64(centerX), float64(centerY))
		if i > 0 {
			player = a.assets.images.player2
		}

		screen.DrawImage(player, op)
	}
}

func (a *GameAdapter) renderWall(screen *ebiten.Image, gx, gy int, orient domain.WallOrientation, valid bool) {
	op := &ebiten.DrawImageOptions{}
	var wall *ebiten.Image

	x, y := a.getTilePos(gx, gy)
	if orient == domain.WallHorizontal {
		wx := float32(x - 7)
		wy := float32(y + CellSize - 5)
		op.GeoM.Translate(float64(wx), float64(wy))
		wall = a.assets.images.wallH

	} else {
		wx := float32(x + CellSize - 4)
		wy := float32(y - 4)
		op.GeoM.Translate(float64(wx), float64(wy))
		wall = a.assets.images.wallV
	}
	if !valid {
		op.ColorScale.ScaleWithColor(color.RGBA{20, 20, 20, 180})
	}
	screen.DrawImage(wall, op)

}

func (a *GameAdapter) drawHUD(screen *ebiten.Image) {
	screenWidth := float64(a.screenWidth)
	drawRealisticText(screen, "--- TURN ---", screenWidth/2+215, 32, a.assets.fonts.mainFontSmall, ColorBlack)

	activeP := a.DomainGame.ActivePlayer()
	if activeP != nil {
		drawRealisticText(screen, fmt.Sprintf("Player %d (%s)", activeP.ID, a.controllerLabelForPlayer(activeP.ID)), screenWidth/2+150, 90, a.assets.fonts.mainFontSmall, ColorBlue)
	}
	drawRealisticText(screen, "--- ACTION MODE ---", screenWidth/2+175, 155, a.assets.fonts.mainFontSmall, ColorBlack)

	modeText := "MOVE PAWN"
	if a.WallMode {
		orientStr := "HORIZONTAL"
		if a.WallOrientation == domain.WallVertical {
			orientStr = "VERTICAL"
		}
		modeText = fmt.Sprintf("WALL (%s)", orientStr)
	}
	drawRealisticText(screen, modeText, screenWidth/2+150, 235, a.assets.fonts.mainFontSmall, ColorBlue)
	drawRealisticText(screen, "--- CONTROLS ---", screenWidth/2+230, 555, a.assets.fonts.mainFontSmall, ColorBlack)
	drawRealisticText(screen, "[L-Click] Move / Place", screenWidth/2+175, 590, a.assets.fonts.mainFontSmall, ColorBlue)
	drawRealisticText(screen, "[W]       Toggle Wall Mode", screenWidth/2+175, 620, a.assets.fonts.mainFontSmall, ColorBlue)
	drawRealisticText(screen, "[R/Space] Rotate Wall", screenWidth/2+175, 655, a.assets.fonts.mainFontSmall, ColorBlue)
	for i, p := range a.DomainGame.Players {
		y := float64(365 + (i * 100))
		txt := fmt.Sprintf("P%d (%s) - Walls: %d", p.ID, a.controllerLabelForPlayer(p.ID), p.WallsLeft)
		drawRealisticText(screen, txt, screenWidth/2+220, y, a.assets.fonts.mainFontSmall, ColorBlack)

	}
}
