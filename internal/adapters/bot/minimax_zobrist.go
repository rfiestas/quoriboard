package bot

import "quoridor/internal/domain"

type TTFlag uint8

const (
	TTExact      TTFlag = iota
	TTLowerBound        // Fallo en Beta (Corte)
	TTUpperBound        // Fallo en Alpha
)

type TTEntry struct {
	Depth int
	Score float64
	Flag  TTFlag
}

// Tablas de números aleatorios Zobrist (64 bits)
var (
	zobristPlayers [5][domain.BoardSize][domain.BoardSize]uint64
	zobristWalls   [domain.BoardSize][domain.BoardSize][2]uint64
	zobristTurn    [5]uint64
)

func init() {
	// Usamos un generador determinista para llenar las claves Zobrist
	var seed uint64 = 0x123456789ABCDEF0
	nextRand := func() uint64 {
		seed ^= seed << 13
		seed ^= seed >> 7
		seed ^= seed << 17
		return seed
	}

	for p := 0; p < 5; p++ {
		zobristTurn[p] = nextRand()
		for x := 0; x < domain.BoardSize; x++ {
			for y := 0; y < domain.BoardSize; y++ {
				zobristPlayers[p][x][y] = nextRand()
			}
		}
	}

	for x := 0; x < domain.BoardSize; x++ {
		for y := 0; y < domain.BoardSize; y++ {
			zobristWalls[x][y][0] = nextRand()
			zobristWalls[x][y][1] = nextRand()
		}
	}
}
