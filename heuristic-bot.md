# Heuristic Bot

The heuristic bot evaluates every possible move or wall placement using a score function based on weights. It does not explore a game tree (unlike minimax): it simply scores the resulting state of each candidate move and chooses the highest-scoring one.

The weights are trained using the [heuristic training platform](./heuristic-training-platform.md), which runs a genetic algorithm to find the best-performing combinations. The already trained bots (with their final weights) are defined in: [internal/adapters/bot/heuristic_weights.go](./internal/adapters/bot/heuristic_weights.go)


Among others, you will find `aggressive`, `defensive`, `balanced`, and `chaotic` (the 4 manual rivals used in training) and the best bots resulting from the various training rounds.

## Weights

| Weight | Typical Sign | What it rewards/penalizes |
|---|---|---|
| `MyDistanceWeight` | Negative (e.g., `-10`) | Moving closer to your own goal. Being negative means less distance = better score. |
| `RivalDistanceWeight` | Positive (e.g., `8`) | Increasing the rival's distance to their goal. |
| `WallReserveWeight` | Positive | Conserving available walls. |
| `CentralityWeight` | Positive | Occupying positions close to the center of the board (more movement options). |
| `JumpConcededPenalty` | Negative (e.g., `-30`) | Allowing the rival a forward jump. |
| `FunnelingBonus` | Positive | Blocking paths around the rival. |

There are two sets of weights:

- **`NormalWeights`**: Used by default.
- **`PanicWeights`**: Used when the bot enters "panic mode" because it is falling behind the rival, prioritizing heavier defense or accelerating its own advance.

### Score Formula

The score of each candidate move is a weighted sum of the 6 factors:

```
score = MyDistanceWeight    * my_distance
      + RivalDistanceWeight * rival_distance
      + WallReserveWeight   * remaining_walls
      + CentralityWeight    * centrality
      + FunnelingBonus      * funneling
```

`JumpConcededPenalty` is added as an extra term (not multiplied by any variable) when the evaluated move enables a favorable jump for the opponent.

**Tie-breaking**: If two moves get the exact same score, there is no explicit rule — the first one found during evaluation is kept. This is not an engineered mechanism, just the default behavior when ties aren't broken.

### Panic Mode

The switch to `PanicWeights` is decided by comparing the distance to the goal of both players. The threshold of how many cells of difference trigger panic (1, 2...) **is not fixed**: it is itself a parameter trained alongside the rest of the weights within the same genetic process. In other words, the bot not only learns how to play in each mode, but also when to switch modes.

## Seed Bots

These are the 4 manual profiles used as seeds and initial rivals in the [heuristic training platform](./heuristic-training-platform.md). Each also carries its own `PanicThreshold` (number of cells difference with the rival that triggers panic mode).

```go
// Bot_Aggressive prioritizes penalizing the rival over its own path.
// High positive RivalDistanceWeight combined with negative MyDistanceWeight makes it act aggressively.
func Bot_Aggressive() BotConfig {
	return BotConfig{
		PanicThreshold: 1,
		NormalWeights: BotWeights{
			MyDistanceWeight:    -10.0,
			RivalDistanceWeight: 9.98,
			WallReserveWeight:   0.74,
			CentralityWeight:    2.85,
			JumpConcededPenalty: -30.94,
			FunnelingBonus:      14.33,
		},
		PanicWeights: BotWeights{
			MyDistanceWeight:    2.22,
			RivalDistanceWeight: 20.0,
			WallReserveWeight:   2.6,
			CentralityWeight:    1.47,
			JumpConcededPenalty: -49.26,
			FunnelingBonus:      25.49,
		},
	}
}

// Bot_Defensive prioritizes its own progression to the goal line while conserving walls.
// High positive MyDistanceWeight keeps it focused on its own path, minimizing risky offensive moves.
func Bot_Defensive() BotConfig {
	return BotConfig{
		PanicThreshold: 2,
		NormalWeights: BotWeights{
			MyDistanceWeight:    15.0,
			RivalDistanceWeight: -2.0,
			WallReserveWeight:   10.0,
			CentralityWeight:    1.5,
			JumpConcededPenalty: -10.0,
			FunnelingBonus:      5.0,
		},
		PanicWeights: BotWeights{
			MyDistanceWeight:    25.0,
			RivalDistanceWeight: -5.0,
			WallReserveWeight:   15.0,
			CentralityWeight:    0.5,
			JumpConcededPenalty: -20.0,
			FunnelingBonus:      2.0,
		},
	}
}

// Bot_Balanced keeps an even weight distribution between advancing and obstructing the opponent.
// Balanced values for MyDistanceWeight and RivalDistanceWeight allow it to adapt dynamically to game state.
func Bot_Balanced() BotConfig {
	return BotConfig{
		PanicThreshold: 2,
		NormalWeights: BotWeights{
			MyDistanceWeight:    5.0,
			RivalDistanceWeight: 5.0,
			WallReserveWeight:   3.0,
			CentralityWeight:    3.0,
			JumpConcededPenalty: -25.0,
			FunnelingBonus:      10.0,
		},
		PanicWeights: BotWeights{
			MyDistanceWeight:    10.0,
			RivalDistanceWeight: 10.0,
			WallReserveWeight:   5.0,
			CentralityWeight:    2.0,
			JumpConcededPenalty: -35.0,
			FunnelingBonus:      15.0,
		},
	}
}

// Bot_Chaotic uses unconventional weight distributions to create unpredictable positional dynamics.
// Zero or inverted distance weights force the decision engine to prioritize secondary factors like centrality and funneling.
func Bot_Chaotic() BotConfig {
	return BotConfig{
		PanicThreshold: 3,
		NormalWeights: BotWeights{
			MyDistanceWeight:    0.0,
			RivalDistanceWeight: 0.0,
			WallReserveWeight:   20.0,
			CentralityWeight:    15.0,
			JumpConcededPenalty: -50.0,
			FunnelingBonus:      30.0,
		},
		PanicWeights: BotWeights{
			MyDistanceWeight:    -15.0,
			RivalDistanceWeight: -15.0,
			WallReserveWeight:   0.0,
			CentralityWeight:    0.0,
			JumpConcededPenalty: 0.0,
			FunnelingBonus:      50.0,
		},
	}
}
```

As seen in the [heuristic training results](./heuristic-training-platform.md#results), out of these four, only `aggressive` turned out to be truly competitive; `defensive` and `balanced` failed to win any matches against the trained population, and `chaotic` fell somewhere in the middle.

## Detailed Calculation of Each Weight

### MyDistanceWeight / RivalDistanceWeight

The distance to the goal is **not** Manhattan or Euclidean: it is the real distance of a valid path calculated via BFS on the board, taking placed walls into account (`getFastShortestPath` over the `WallGrid`). This is important because a wall can significantly lengthen a player's real path without that being reflected in a straight-line distance metric.

### WallReserveWeight

Direct calculation without extra normalization: `remaining_walls × WallReserveWeight`. It only values having walls in reserve as a positional advantage; it neither penalizes nor rewards the act of placing a wall itself.

### CentralityWeight

Calculated from the lateral displacement relative to the center of the board:

```go
boardCenter := 4.0
lateralOffset := abs(lateral_position - boardCenter)
centrality := boardCenter - lateralOffset
```

Note: it only accounts for the lateral position (column), not the distance along the forward axis toward the goal.

### FunnelingBonus

Counts the free directions (no wall) around the rival's current position, looking directly at the `WallGrid` in all four directions:

```go
funneling = 4 - oppFreePaths
```

Important details:
- It does not use `GetValidMovesInto`, so it doesn't check real legal moves, only the presence of walls.
- It has no normalization or additional limit besides the natural range `0`–`4`.
- At the edges of the board, it may count directions as "free" that actually lead off the board, because the `WallGrid` does not mark boundaries as blocked. This is a known, uncorrected inaccuracy.

### JumpConcededPenalty

Triggered **before** the rival jumps, not after. The function reviews the rival's possible moves and looks for displacements greater than one cell (jumps). If it finds a legal jump that also reduces the rival's distance to their goal, it applies the penalty immediately — it doesn't wait for the rival to actually execute the jump.

### Candidate Move Generation

The bot does not crudely evaluate all possible wall positions (up to 128 orientation/position combinations). Instead:

- All legal pawn movements (including jumps) are considered.
- For walls, **spatial filtering** is applied: only placements close to the players' positions are tested, rather than the entire board.
- On top of that reduced set, **quick-ranking pruning** is also applied, discarding low-priority candidates before scoring them with the full formula.

This keeps the evaluation cost per turn constrained, at the cost of not guaranteeing 100% exploration of the possible move space on every turn.

## Highlighted Bots from Trainings

Selection of relevant bots emerging from the two training series (see [heuristic training platform](./heuristic-training-platform.md) for context on each). "Role" indicates whether the bot stood out as a top player or as a rival promoted to the benchmark pool.

| Bot | Training | Role | Panic Threshold | Normal Weights (MyDist / RivalDist / WallRes / Centrality / JumpPen / Funneling) | Panic Weights (MyDist / RivalDist / WallRes / Centrality / JumpPen / Funneling) |
| :--- | :--- | :--- | :---: | :--- | :--- |
| **Bot_E6G5M74** | vs Heuristic | Player | 1 | -11.89 / 9.76 / -4.22 / 7.96 / -28.82 / 15.73 | 1.91 / 22.28 / 5.05 / 6.26 / -49.93 / 19.8 |
| **Bot_E6G5M81** | vs Heuristic | Player | 1 | -12.55 / 11.48 / -4.13 / 9.32 / -28.82 / 15.51 | 2.49 / 23.69 / 5.2 / 4.21 / -49.48 / 20.31 |
| **Bot_E6G5M41** | vs Heuristic | Player | 1 | -12.55 / 11.48 / -4.22 / 9.32 / -28.82 / 15.68 | 3.33 / 23.41 / 5.2 / 5.77 / -49.48 / 20.31 |
| **Bot_E6G5M39** | vs Heuristic | Player | 1 | -11.89 / 9.87 / -2.99 / 9.5 / -29.67 / 16.13 | 1.93 / 22.35 / 5.2 / 7.29 / -50.0 / 20.71 |
| **Bot_E6G5M86** | vs Heuristic | Player | 1 | -11.89 / 8.91 / -4.73 / 8.66 / -28.82 / 16.13 | 0.97 / 22.35 / 5.55 / 5.66 / -50.0 / 20.85 |
| **Bot_E1G3M41** | vs Heuristic | Rival | 1 | -9.39 / 6.13 / 7.37 / 6.17 / -29.05 / 7.6 | 2.22 / 17.48 / 4.0 / 3.41 / -50.0 / 25.49 |
| **Bot_E2G3M99** | vs Heuristic | Rival | 0 | -11.92 / 9.8 / -1.08 / 7.2 / -29.54 / 21.29 | -6.04 / 26.31 / 2.53 / 4.02 / -48.71 / 17.44 |
| **Bot_E1G1M78** | vs Heuristic | Rival | 1 | -12.58 / 13.91 / -0.91 / 5.27 / -26.98 / 18.33 | 2.22 / 20.0 / 3.02 / 3.37 / -49.59 / 22.91 |
| **Bot_E2G4M40** | vs Heuristic | Rival | 0 | -14.83 / 7.99 / -1.08 / 7.2 / -29.54 / 21.45 | -6.04 / 26.31 / 3.44 / 5.2 / -48.71 / 17.44 |
| **Bot_E5G8M26** | vs Heuristic | Rival | -2 | -13.96 / 4.73 / 6.02 / 12.88 / -28.42 / 22.62 | -3.45 / 24.54 / 8.64 / 6.94 / -48.15 / 21.35 |
| **Bot_E4G3M81** | vs Minimax3 | Player | 2 | -11.44 / 7.97 / 1.59 / 2.23 / -33.32 / 20.59 | -10.13 / 19.85 / -4.71 / -6.79 / -50.0 / 24.77 |
| **Bot_E4G4M55** | vs Minimax3 | Player | 2 | -11.44 / 7.97 / 3.9 / 2.23 / -33.32 / 21.16 | -10.55 / 22.49 / -4.71 / -6.84 / -50.0 / 25.8 |
| **Bot_E4G4M45** | vs Minimax3 | Player | 2 | -11.44 / 7.97 / 1.59 / 2.23 / -35.16 / 20.59 | -10.13 / 19.85 / -2.87 / -6.79 / -50.0 / 24.77 |
| **Bot_E4G4M78** | vs Minimax3 | Player | 2 | -11.44 / 7.97 / 1.59 / 2.23 / -33.32 / 20.59 | -10.13 / 19.85 / -4.71 / -6.79 / -50.0 / 23.9 |
| **Bot_E4G4M79** | vs Minimax3 | Player | 2 | -12.43 / 7.32 / 1.35 / 0.0 / -33.99 / 20.15 | -7.64 / 21.76 / -4.69 / -4.82 / -50.0 / 22.95 |
| **Bot_E4G10M52** | vs Minimax3 | Rival | 2 | -11.44 / 7.97 / -0.37 / 2.23 / -33.32 / 20.59 | -10.13 / 19.24 / -4.71 / -8.25 / -50.0 / 23.05 |
| **Bot_E5G4M77** | vs Minimax3 | Rival | 2 | -13.25 / 7.97 / 1.59 / 2.01 / -35.16 / 21.88 | -10.13 / 21.0 / -2.87 / -6.79 / -50.0 / 23.8 |
| **Bot_E5G4M80** | vs Minimax3 | Rival | 2 | -12.93 / 8.61 / 3.31 / 1.13 / -30.65 / 18.58 | -4.57 / 21.28 / -1.91 / -4.35 / -48.49 / 21.21 |
| **Bot_E6G4M92** | vs Minimax3 | Rival | 2 | -11.44 / 7.97 / 1.16 / 2.52 / -33.32 / 21.71 | -10.55 / 21.94 / -4.71 / -6.84 / -50.0 / 25.8 |
| **Bot_E6G4M33** | vs Minimax3 | Rival | 2 | -11.44 / 7.97 / 1.59 / 1.83 / -35.16 / 20.59 | -10.13 / 19.17 / -2.87 / -6.79 / -50.0 / 24.77 |
| **Bot_E6G5M81_b** |vs Minimax4 | Player | 1 | -14.3 / 9.44 / -4.48 / 6.79 / -30.23 / 18.38 | 1.91 / 20.61 / 4.17 / 9.53 / -48.35 / 22.28 |
| **Bot_E6G3M75** |vs Minimax4 | Player | 1 | -15.43 / 12.37 / -5.69 / 8.33 / -29.22 / 18.38 | 3.2 / 20.36 / 5.88 / 8.41 / -48.98 / 22.28 |
| **Bot_E6G4M97** |vs Minimax4 | Player | 1 | -15.43 / 12.37 / -5.69 / 8.33 / -29.22 / 18.56 | 3.2 / 20.36 / 6.01 / 8.41 / -49.63 / 20.69 |
| **Bot_E6G4M71** |vs Minimax4 | Player | 1 | -15.43 / 12.37 / -5.69 / 8.33 / -29.22 / 18.38 | 3.2 / 20.36 / 5.88 / 8.41 / -48.98 / 22.28 |
| **Bot_E6G4M29** |vs Minimax4 | Parameter | 1 | -15.43 / 10.77 / -6.29 / 8.38 / -27.58 / 18.38 | 1.91 / 19.16 / 5.88 / 9.69 / -48.98 / 22.28 |
| **Bot_E1G2M90** |vs Minimax4 | Rival | 1 | -12.54 / 10.44 / -0.99 / 3.59 / -33.32 / 21.72 | -10.13 / 19.85 / -4.71 / -5.01 / -50.0 / 25.4 |
| **Bot_E1G3M77** |vs Minimax4 | Rival | 0 | -15.4 / 7.35 / 1.12 / 5.29 / -35.0 / 23.72 | -9.44 / 20.99 / -2.65 / -5.01 / -48.53 / 26.9 |
*Nomenclature: `E{epoch}G{generation}M{mutation_no}` — identifies the exact point in training where each bot emerged.*

## Notes / Learnings

- The random bot (no intelligence, random moves) served as a baseline to define the minimum interface any game bot must fulfill before complicating things with heuristics.
- Although the heuristic bot does not use deep learning, it is still a "trained bot": both its weights and the panic threshold come from the genetic process, not manual tuning.
- Comparing the bots in the table, `JumpConcededPenalty` converges in almost all good bots to values close to the limit `-50.0`/`-49.x` in panic mode — it appears to be the most "non-negotiable" weight of all once training converges.
