package ebitenui

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

func (a *GameAdapter) updateMenu() {
	screenWidth := a.screenWidth

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		cx, cy := ebiten.CursorPosition()

		// 2 Players Button
		if cx >= 230 && cx <= 370 && cy >= 170 && cy <= 210 {
			if a.NumPlayers != 2 {
				a.NumPlayers = 2
				a.refreshMenuConfigs()
			}
		}

		// 4 Players Button
		if cx >= 390 && cx <= 530 && cy >= 170 && cy <= 210 {
			if a.NumPlayers != 4 {
				a.NumPlayers = 4
				a.refreshMenuConfigs()
			}
		}

		// Toggle Human/Bot
		for i := range a.MenuConfig {
			btnY := 250 + (i * 165)
			if cx >= screenWidth/2+10 && cx <= screenWidth/2+290 && cy >= btnY && cy <= btnY+65 {
				a.MenuConfig[i].ControllerType = nextSelectableControllerType(a.MenuConfig[i].ControllerType, a.RLV8PythonEnabled)
			}
		}

		// Start Game Button
		if cx >= screenWidth/2-185 && cx <= screenWidth/2+190 && cy >= 545 && cy <= 675 {
			a.StartGame()
		}
	}
}

func (a *GameAdapter) drawMenu(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	screenWidth := float64(a.screenWidth)

	// Background
	screen.DrawImage(a.assets.images.backgroundMenuScene, op)

	//Panel
	drawRealisticText(screen, "---QUORIBOARD---", screenWidth/2-225, 90, a.assets.fonts.mainFontBig, ColorBlack)
	for i, p := range a.MenuConfig {
		y := float64(265 + (i * 165))
		drawRealisticText(screen, fmt.Sprintf("Player %d:", p.ID), screenWidth/2-190, y, a.assets.fonts.mainFontSmall, ColorBlack)
		drawRealisticText(screen, controllerLabel(p.ControllerType), screenWidth/2+50, y, a.assets.fonts.mainFontSmall, ColorBlue)
	}
	drawRealisticText(screen, "START GAME", screenWidth/2-135, 565, a.assets.fonts.mainFontBig, ColorBlue)
}

/*
rect := ebiten.NewImage(int(280), int(65))
		rect.Fill(color.RGBA{100, 100, 100, 100})

		op1 := &ebiten.DrawImageOptions{}
		op1.GeoM.Translate(screenWidth/2+10, y-15)
		screen.DrawImage(rect, op1)

*/
