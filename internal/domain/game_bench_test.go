package domain

import "testing"

func getValidMoves(g *Game, p *Player) []Move {
	var buf [8]Move
	count := g.GetValidMovesInto(p, &buf)
	return buf[:count]
}

// BenchmarkGetValidMoves measures the cost of generating valid moves for a player.
func BenchmarkGetValidMoves(b *testing.B) {
	g := helperSetupGame(2)
	p := g.Players[0]

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = getValidMoves(g, p)
	}
}

// BenchmarkCanPlaceWall measures the cost of validating a legal wall placement.
func BenchmarkCanPlaceWall(b *testing.B) {
	g := helperSetupGame(2)
	p := g.Players[0]

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = g.CanPlaceWall(3, 3, WallHorizontal, p)
	}
}

// BenchmarkPlaceWall measures the cost of placing a wall and updating game state.
func BenchmarkPlaceWall(b *testing.B) {
	g := helperSetupGame(2)
	p := g.Players[0]
	initialWalls := p.WallsLeft

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		g.Walls = g.Walls[:0]
		p.WallsLeft = initialWalls
		_ = g.PlaceWall(3, 3, WallHorizontal, p)
	}
}

func BenchmarkGetValidMovesParallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		g := helperSetupGame(2)
		p := g.Players[0]

		for pb.Next() {
			_ = getValidMoves(g, p)
		}
	})
}

func BenchmarkCanPlaceWallParallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		g := helperSetupGame(2)
		p := g.Players[0]

		for pb.Next() {
			_ = g.CanPlaceWall(3, 3, WallHorizontal, p)
		}
	})
}

func BenchmarkPlaceWallParallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		g := helperSetupGame(2)
		p := g.Players[0]
		initialWalls := p.WallsLeft

		for pb.Next() {
			g.Walls = g.Walls[:0]
			p.WallsLeft = initialWalls
			_ = g.PlaceWall(3, 3, WallHorizontal, p)
		}
	})
}
