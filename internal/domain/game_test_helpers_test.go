package domain

import "image/color"

// helperSetupGame creates a game with the requested number of players
// in a deterministic initial configuration.
//
// It is intended only for tests.
func helperSetupGame(numPlayers int) *Game {
	g := &Game{
		State:     StatePlaying,
		TurnIndex: 0,
		Players:   make([]*Player, 0, numPlayers),
		Walls:     make([]Wall, 0),
	}

	// Standard Quoridor starting positions.
	//
	// Player 1 starts at the bottom center and targets the top row.
	// Player 2 starts at the top center and targets the bottom row.
	//
	// Additional players, if requested, are placed in deterministic
	// positions that do not interfere with the standard two-player setup.
	for i := 0; i < numPlayers; i++ {
		p := &Player{
			ID:        i + 1,
			Color:     color.White,
			WallsLeft: 10,
			TargetX:   -1,
			TargetY:   -1,
		}

		switch i {
		case 0:
			p.GridX = 4
			p.GridY = 8
			p.TargetY = 0

		case 1:
			p.GridX = 4
			p.GridY = 0
			p.TargetY = 8

		default:
			// Deterministic fallback position for tests that request
			// more than two players.
			p.GridX = i % BoardSize
			p.GridY = i % BoardSize
			p.TargetY = 0
		}

		g.Players = append(g.Players, p)
	}

	return g
}
