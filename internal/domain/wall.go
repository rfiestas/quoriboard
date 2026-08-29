package domain

type WallOrientation int

const (
	WallHorizontal WallOrientation = iota
	WallVertical
)

type Wall struct {
	GridX       int             // Top-left intersection (0 to 7)
	GridY       int             // Top-left intersection (0 to 7)
	Orientation WallOrientation // Horizontal or Vertical
	OwnerID     int
}
