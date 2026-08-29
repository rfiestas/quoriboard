package translator

import "quoridor/internal/domain"

const (
	ActionSpaceSize = 136
	ObservationSize = 491
	MaxTurns        = 150
)

var moveDirections = [8][2]int{
	{0, -1}, {0, 1}, {-1, 0}, {1, 0},
	{-1, -1}, {1, -1}, {-1, 1}, {1, 1},
}

func MoveDirection(action int) (int, int) {
	return moveDirections[action][0], moveDirections[action][1]
}

// RealAction converts a model action into the board coordinates for player 2.
// The transform is its own inverse.
func RealAction(player *domain.Player, action int) int {
	if player.ID != 2 {
		return action
	}
	switch action {
	case 0:
		return 1
	case 1:
		return 0
	case 4:
		return 6
	case 5:
		return 7
	case 6:
		return 4
	case 7:
		return 5
	}
	if action < 8 {
		return action
	}
	index := action - 8
	if index < 64 {
		return 8 + (index/8)*8 + (7 - index%8)
	}
	index -= 64
	return 72 + (index/8)*8 + (7 - index%8)
}

func BuildObservation(game *domain.Game, current *domain.Player, turns int) []float32 {
	values := make([]float32, ObservationSize)
	if current == nil {
		return values
	}
	other := game.Players[0]
	if other.ID == current.ID {
		other = game.Players[1]
	}
	currentY, otherY := current.GridY, other.GridY
	if current.ID == 2 {
		currentY = 8 - currentY
		otherY = 8 - otherY
	}
	values[current.GridX*9+currentY] = 1
	values[81+other.GridX*9+otherY] = 1
	for _, wall := range game.Walls {
		base := 162
		if wall.Orientation == domain.WallVertical {
			base = 243
		}
		wallY := wall.GridY
		if current.ID == 2 {
			wallY = 7 - wallY
		}
		values[base+wall.GridX*9+wallY] = 1
	}
	values[486] = float32(current.WallsLeft) / 10
	values[487] = float32(other.WallsLeft) / 10
	values[488] = float32(turns) / MaxTurns
	return values
}

func BuildActionMask(game *domain.Game, player *domain.Player) []bool {
	mask := make([]bool, ActionSpaceSize)
	var moves [8]domain.Move
	count := game.GetValidMovesInto(player, &moves)
	for i := 0; i < count; i++ {
		dx, dy := moves[i].X-player.GridX, moves[i].Y-player.GridY
		var direction int
		switch {
		case dx == 0 && dy < 0:
			direction = 0
		case dx == 0 && dy > 0:
			direction = 1
		case dy == 0 && dx < 0:
			direction = 2
		case dy == 0 && dx > 0:
			direction = 3
		case dx < 0 && dy < 0:
			direction = 4
		case dx > 0 && dy < 0:
			direction = 5
		case dx < 0 && dy > 0:
			direction = 6
		default:
			direction = 7
		}
		mask[RealAction(player, direction)] = true
	}
	if player.WallsLeft == 0 {
		return mask
	}
	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			if game.CanPlaceWall(x, y, domain.WallHorizontal, player) {
				mask[RealAction(player, 8+x*8+y)] = true
			}
			if game.CanPlaceWall(x, y, domain.WallVertical, player) {
				mask[RealAction(player, 72+x*8+y)] = true
			}
		}
	}
	return mask
}

func DecodeAction(game *domain.Game, player *domain.Player, action int) domain.PlayerAction {
	realAction := RealAction(player, action)
	if realAction < 8 {
		var moves [8]domain.Move
		count := game.GetValidMovesInto(player, &moves)
		dx, dy := moveDirections[realAction][0], moveDirections[realAction][1]
		for i := 0; i < count; i++ {
			move := moves[i]
			moveDX, moveDY := move.X-player.GridX, move.Y-player.GridY
			match := false
			if realAction < 4 {
				match = (dx == 0 && moveDX == 0 && moveDY*dy > 0) || (dy == 0 && moveDY == 0 && moveDX*dx > 0)
			} else {
				match = moveDX == dx && moveDY == dy
			}
			if match {
				return domain.PlayerAction{Type: domain.ActionMove, TargetX: move.X, TargetY: move.Y}
			}
		}
		return domain.PlayerAction{}
	}
	index := realAction - 8
	orientation := domain.WallHorizontal
	if index >= 64 {
		orientation, index = domain.WallVertical, index-64
	}
	return domain.PlayerAction{Type: domain.ActionPlaceWall, TargetX: index / 8, TargetY: index % 8, WallOrientation: orientation}
}
