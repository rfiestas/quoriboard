package ebitenui

import (
	"image/color"
	"log"

	"quoridor/assets"
	"quoridor/internal/domain"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	CellSize  = 62
	GapSize   = 10
	BoardLeft = 40
	BoardTop  = 40
)

type GameAdapter struct {
	DomainGame        *domain.Game
	Controllers       map[int]domain.PlayerController
	RLV8PythonEnabled bool
	RLV8Checkpoint    string
	RLV8Python        string
	SelectedTileX     int
	SelectedTileY     int

	// Menu State
	NumPlayers int
	MenuConfig []playerSlot
	Colors     []color.Color

	Winner *domain.Player

	// Playing State
	WallMode        bool                   // Toggle between Move mode and Wall mode
	WallOrientation domain.WallOrientation // Current orientation (Horizontal / Vertical)
	HoverWallX      int                    // Calculated wall intersection X (0-7)
	HoverWallY      int                    // Calculated wall intersection Y (0-7)
	HoverWallValid  bool                   // True if current hovered wall position is legal

	screenWidth  int
	screenHeight int

	assets assetsSources
}

type assetsSources struct {
	fonts  fonts
	images images
}

type playerSlot struct {
	ID             int
	Color          color.Color
	ControllerType ControllerType
}

type fonts struct {
	mainFontBig   text.Face
	mainFontSmall text.Face
}

type images struct {
	backgroundMenuScene  *ebiten.Image
	backgroundBoardScene *ebiten.Image
	winScene             *ebiten.Image
	square               *ebiten.Image
	player1              *ebiten.Image
	player2              *ebiten.Image
	wallV                *ebiten.Image
	wallH                *ebiten.Image
}

func NewGameAdapter(g *domain.Game) *GameAdapter {
	defaultColors := []color.Color{
		color.RGBA{220, 60, 60, 255},  // P1: Red
		color.RGBA{60, 120, 220, 255}, // P2: Blue
		color.RGBA{60, 200, 60, 255},  // P3: Green
		color.RGBA{230, 180, 40, 255}, // P4: Yellow
	}

	adapter := &GameAdapter{
		DomainGame:    g,
		Controllers:   make(map[int]domain.PlayerController),
		NumPlayers:    2,
		Colors:        defaultColors,
		SelectedTileX: -1,
		SelectedTileY: -1,
	}
	adapter.refreshMenuConfigs()
	return adapter
}

func (a *GameAdapter) refreshMenuConfigs() {
	a.MenuConfig = make([]playerSlot, a.NumPlayers)
	for i := 0; i < a.NumPlayers; i++ {
		a.MenuConfig[i] = playerSlot{
			ID:             i + 1,
			ControllerType: ControllerHuman,
			Color:          a.Colors[i],
		}
	}
}

func (a *GameAdapter) LoadResources() {
	// Fonts
	a.assets.fonts.mainFontBig = MustLoadFont(assets.SpaceMonoBytes, 48)
	a.assets.fonts.mainFontSmall = MustLoadFont(assets.SpaceMonoBytes, 24)

	// Images
	a.assets.images.backgroundMenuScene = mustLoadEbitenImage(assets.BackgroundMenuSceneBytes)
	a.assets.images.backgroundBoardScene = mustLoadEbitenImage(assets.BackgroundBoardSceneBytes)
	a.assets.images.winScene = mustLoadEbitenImage(assets.WinSceneBytes)
	a.assets.images.square = mustLoadEbitenImage(assets.SquareBytes)
	a.assets.images.player1 = mustLoadEbitenImage(assets.Player1Bytes)
	a.assets.images.player2 = mustLoadEbitenImage(assets.Player2Bytes)
	a.assets.images.wallV = mustLoadEbitenImage(assets.WallVBytes)
	a.assets.images.wallH = mustLoadEbitenImage(assets.WallHBytes)
}

func (a *GameAdapter) StartGame() {
	configs := make([]domain.PlayerConfig, len(a.MenuConfig))
	for i, slot := range a.MenuConfig {
		configs[i] = domain.PlayerConfig{ID: slot.ID, Color: slot.Color}
	}
	a.DomainGame.InitBoard(configs)

	for _, controller := range a.Controllers {
		if closer, ok := controller.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
	}
	a.Controllers = make(map[int]domain.PlayerController)
	for _, slot := range a.MenuConfig {
		controller, err := a.createController(slot.ControllerType)
		if err != nil {
			panic(err)
		}
		a.Controllers[slot.ID] = controller
	}

	a.DomainGame.State = domain.StatePlaying
}

func (a *GameAdapter) CloseControllers() {
	for _, controller := range a.Controllers {
		if closer, ok := controller.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
	}
}

func (a *GameAdapter) Update() error {
	switch a.DomainGame.State {
	case domain.StateMenu:
		a.updateMenu()
	case domain.StatePlaying:
		a.updatePlaying()
	case domain.StateWin:
		a.updateWin() // NEW
	}
	return nil
}

func (a *GameAdapter) updatePlaying() {
	cx, cy := ebiten.CursorPosition()
	a.SelectedTileX, a.SelectedTileY = a.pixelToGrid(cx, cy)

	// Update hover position every frame
	a.updateHoverWall(cx, cy)

	activePlayer := a.DomainGame.ActivePlayer()
	if activePlayer == nil {
		return
	}

	controller := a.Controllers[activePlayer.ID]
	if controller == nil {
		return
	}

	action := controller.GetAction(a.DomainGame, activePlayer)
	if action.Type == domain.ActionNone {
		if !a.playerCanAct(a.DomainGame, activePlayer) {
			log.Printf("[GameAdapter] jugador %d sin acción válida; avanzando turno", activePlayer.ID)
			a.DomainGame.NextTurn()
			return
		}
		log.Printf("[GameAdapter] jugador %d devolvió ActionNone con acciones disponibles", activePlayer.ID)
		return
	}

	log.Printf("[GameAdapter] acción del jugador %d: tipo=%v x=%d y=%d orient=%v", activePlayer.ID, action.Type, action.TargetX, action.TargetY, action.WallOrientation)

	if action.Type == domain.ActionMove {
		if !a.isValidMoveAction(a.DomainGame, activePlayer, action) {
			log.Printf("[GameAdapter] movimiento inválido del jugador %d: (%d,%d)", activePlayer.ID, action.TargetX, action.TargetY)
			a.DomainGame.NextTurn()
			return
		}
		activePlayer.GridX = action.TargetX
		activePlayer.GridY = action.TargetY

		if (activePlayer.TargetY != -1 && activePlayer.GridY == activePlayer.TargetY) ||
			(activePlayer.TargetX != -1 && activePlayer.GridX == activePlayer.TargetX) {
			a.Winner = activePlayer
			a.DomainGame.State = domain.StateWin
			return
		}
		a.DomainGame.NextTurn()
	} else if action.Type == domain.ActionPlaceWall {
		if !a.isValidWallAction(a.DomainGame, activePlayer, action) {
			log.Printf("[GameAdapter] muro inválido del jugador %d: (%d,%d,%v)", activePlayer.ID, action.TargetX, action.TargetY, action.WallOrientation)
			a.DomainGame.NextTurn()
			return
		}
		if a.DomainGame.PlaceWall(action.TargetX, action.TargetY, action.WallOrientation, activePlayer) {
			a.WallMode = false // Switch back to move mode after placing
			a.HoverWallX = -1
			a.HoverWallY = -1
			a.DomainGame.NextTurn()
		}
	}
}

func (a *GameAdapter) isValidMoveAction(g *domain.Game, p *domain.Player, action domain.PlayerAction) bool {
	if action.Type != domain.ActionMove || p == nil {
		return false
	}
	var validMoveBuf [8]domain.Move
	for i := 0; i < g.GetValidMovesInto(p, &validMoveBuf); i++ {
		move := validMoveBuf[i]
		if move.X == action.TargetX && move.Y == action.TargetY {
			return true
		}
	}
	return false
}

func (a *GameAdapter) isValidWallAction(g *domain.Game, p *domain.Player, action domain.PlayerAction) bool {
	if action.Type != domain.ActionPlaceWall || p == nil {
		return false
	}
	return g.CanPlaceWall(action.TargetX, action.TargetY, action.WallOrientation, p)
}

func (a *GameAdapter) playerCanAct(g *domain.Game, p *domain.Player) bool {
	if p == nil {
		return false
	}

	var moveBuf [8]domain.Move
	if g.GetValidMovesInto(p, &moveBuf) > 0 {
		return true
	}

	if p.WallsLeft <= 0 {
		return false
	}

	for wx := 0; wx < domain.BoardSize-1; wx++ {
		for wy := 0; wy < domain.BoardSize-1; wy++ {
			for _, orient := range []domain.WallOrientation{domain.WallHorizontal, domain.WallVertical} {
				if g.CanPlaceWall(wx, wy, orient, p) {
					return true
				}
			}
		}
	}

	return false
}

func (a *GameAdapter) updateHoverWall(px, py int) {
	a.HoverWallX = -1
	a.HoverWallY = -1
	a.HoverWallValid = false

	if !a.WallMode {
		return
	}

	// Calculate offset from board origin
	relX := px - BoardLeft
	relY := py - BoardTop

	step := CellSize + GapSize

	// Determine nearest tile intersection (0 to 7)
	gx := relX / step
	gy := relY / step

	// Clamp to valid wall intersection range (0 to BoardSize - 2)
	if gx < 0 {
		gx = 0
	}
	if gx >= domain.BoardSize-1 {
		gx = domain.BoardSize - 2
	}
	if gy < 0 {
		gy = 0
	}
	if gy >= domain.BoardSize-1 {
		gy = domain.BoardSize - 2
	}

	// Ensure the cursor is within the general board bounds before enabling hover
	maxBoardCoord := domain.BoardSize*CellSize + (domain.BoardSize-1)*GapSize
	if relX >= 0 && relX <= maxBoardCoord && relY >= 0 && relY <= maxBoardCoord {
		a.HoverWallX = gx
		a.HoverWallY = gy

		activeP := a.DomainGame.ActivePlayer()
		if activeP != nil {
			a.HoverWallValid = a.DomainGame.CanPlaceWall(gx, gy, a.WallOrientation, activeP)
		}
	}
}

func (a *GameAdapter) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{30, 30, 40, 255})

	switch a.DomainGame.State {
	case domain.StateMenu:
		a.drawMenu(screen)
	case domain.StatePlaying:
		a.drawBoard(screen)
		a.drawHUD(screen)
	case domain.StateWin:
		a.drawBoard(screen) // Draw frozen board in background
		a.drawHUD(screen)
		a.drawWinOverlay(screen) // NEW: Draw victory modal over board
	}
}

func (a *GameAdapter) Layout(outsideWidth, outsideHeight int) (int, int) {
	a.screenWidth = outsideWidth
	a.screenHeight = outsideHeight
	return outsideWidth, outsideHeight
}

func (a *GameAdapter) pixelToGrid(px, py int) (int, int) {
	for gx := 0; gx < domain.BoardSize; gx++ {
		for gy := 0; gy < domain.BoardSize; gy++ {
			x, y := a.getTilePos(gx, gy)
			if px >= x && px < x+CellSize && py >= y && py < y+CellSize {
				return gx, gy
			}
		}
	}
	return -1, -1
}

func (a *GameAdapter) getTilePos(gx, gy int) (int, int) {
	x := BoardLeft + gx*(CellSize+GapSize)
	y := BoardTop + gy*(CellSize+GapSize)
	return x, y
}
