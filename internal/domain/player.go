package domain

import "image/color"

type Player struct {
	ID        int
	Color     color.Color
	GridX     int
	GridY     int
	WallsLeft int
	TargetX   int
	TargetY   int
}

type PlayerConfig struct {
	ID    int
	Color color.Color
}

type ActionType int

const (
	ActionNone ActionType = iota
	ActionMove
	ActionPlaceWall
)

type PlayerAction struct {
	Type            ActionType
	TargetX         int
	TargetY         int
	WallOrientation WallOrientation
}

// PlayerController is an Inbound/Driven Port implemented by UI or Bot adapters
type PlayerController interface {
	GetAction(g *Game, p *Player) PlayerAction
}
