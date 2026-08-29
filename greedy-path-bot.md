# Greedy Path Bot

The Greedy Path Bot uses a pure heuristic, greedy decision-making approach based on Breadth-First Search (BFS) pathfinding. Unlike the Minimax bot, it does not explore future game states using a search tree. Instead, it relies on rigid, immediate rules to evaluate the current board state.

## Action Selection Logic

Every turn, the bot calculates the exact number of steps (shortest path) required for both itself and the opponent to reach their respective targets. Based on these distances, it follows a strict priority ruleset:

1. **RULE 1: Sprint to Goal (End Game)**
   If the bot is 3 steps or fewer from winning, it ignores all other strategies and always moves its pawn along the shortest path toward the goal.

2. **RULE 2: Defensive Wall Placement**
   If the opponent is closer to winning (meaning the opponent's shortest path is smaller than the bot's) and the bot still has walls left, it attempts to place a defensive wall. 
   - It simulates placing a wall in every legal position on the board.
   - It evaluates how each placement affects the opponent's shortest path.
   - It selects the wall that **maximizes the opponent's delay** (makes their path as long as possible) while ensuring neither player is completely trapped.

3. **RULE 3: Default Move**
   If the bot is leading the race (no defensive wall is needed) or if no useful wall placement was found, it defaults to advancing its pawn along the current shortest path.

## Pathfinding Optimization

Because distance calculations are the core of this bot's logic, it includes technical optimizations to maintain high performance:

- **Zero-Allocation BFS:** Just like the highly optimized Minimax bot, the Greedy Path Bot calculates distances without triggering the Garbage Collector. It uses a static `9x9` array for the `visited` map and a fixed-size array for the BFS queue. This means finding the shortest path consumes exactly `0` extra memory allocations per call, keeping the bot extremely fast and lightweight.