# Random Bot

The **Random Bot** is a lightweight, non-deterministic bot designed primarily to **validate the domain bot interface** (`GetAction(g *domain.Game, p *domain.Player) domain.PlayerAction`). It serves as a baseline dummy opponent for testing, engine integration checks, and benchmarking.

## Purpose

- **Interface Validation:** Ensures that the core game domain can interact seamlessly with any bot fulfilling the player action contract, regardless of internal decision-making complexity.
- **Baseline Testing:** Provides a zero-intelligence opponent to test game flow, rule enforcement, turn switching, and wall placement legality under chaotic inputs.

## Decision Logic

The bot operates on a pure pseudo-random decision flow with a configurable wall placement probability (`wallProbability`):

1. **Wall Placement Attempt:**
   - If the player has remaining walls (`WallsLeft > 0`), the bot checks against its configured `wallProbability` threshold.
   - If triggered, it makes up to **30 random attempts** to select a board coordinate `(wx, wy)` and orientation (`WallHorizontal` or `WallVertical`).
   - If `g.CanPlaceWall(...)` confirms the placement is valid, it immediately returns a `domain.ActionPlaceWall` action.

2. **Pawn Movement Fallback:**
   - If no wall was placed (either due to the probability check, having 0 walls left, or failing 30 legal placement attempts), the bot queries the game engine for valid pawn moves using `g.GetValidMovesInto(p, &validMoveBuf)`.
   - If valid moves exist, it picks one uniformly at random and returns a `domain.ActionMove` action.
   - If no valid moves are available, it returns `domain.ActionNone`.