package assets

import (
	_ "embed"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// FONT
//
//go:embed fonts/SpaceMono-Regular.ttf
var SpaceMonoBytes []byte
var MainFontBig text.Face
var MainFontSmall text.Face

// Images
//
//go:embed images/menu_scene.png
var BackgroundMenuSceneBytes []byte
var BackgroundMenuScene *ebiten.Image

//go:embed images/board_scene.png
var BackgroundBoardSceneBytes []byte
var BackgroundBoardScene *ebiten.Image

//go:embed images/square.png
var SquareBytes []byte
var Square *ebiten.Image

//go:embed images/player1.png
var Player1Bytes []byte
var Player1 *ebiten.Image

//go:embed images/player2.png
var Player2Bytes []byte
var Player2 *ebiten.Image

//go:embed images/wall_v.png
var WallVBytes []byte
var WallV *ebiten.Image

//go:embed images/wall_h.png
var WallHBytes []byte
var WallH *ebiten.Image

//go:embed images/win_scene.png
var WinSceneBytes []byte
var WinScene *ebiten.Image

//go:embed model/quoridor_rl_model.json
var RLJSONWeights []byte
