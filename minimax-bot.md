# Minimax Bot

The Minimax bot uses a classic game theory algorithm to explore the game tree. Unlike the Heuristic bot, it does not rely on heavily weighted formulas to guess the best move; instead, it looks ahead several turns (`MaxDepth`) into the future, assuming both players will play perfectly, and calculates the mathematically safest move.

Because the game tree for Quoridor is massive (due to the large number of possible wall placements), a standard Minimax approach is too slow. Therefore, this bot heavily relies on **performance, CPU, and memory optimizations** rather than machine learning training.

## Evaluation (Leaf Nodes)

When the bot reaches the maximum search depth, it evaluates the board state using a much simpler formula compared to the heuristic bot:

```text
score = (opponent_distance - bot_distance) * 10.0
      + (bot_walls_left - opponent_walls_left) * 2.0
      + centrality * 0.5
```

It prioritizes being closer to the goal than the opponent, followed by having a wall advantage, and slightly favors staying near the center of the board.

## Performance & Search Optimizations

To evaluate the game tree deep enough within a reasonable time, the bot implements several advanced optimizations:

### 1. Alpha-Beta Pruning & Move Ordering
The bot uses Alpha-Beta pruning to discard branches of the game tree that are mathematically proven to be worse than a previously evaluated move.

To make this pruning as aggressive and efficient as possible, **Move Ordering** is applied. The bot scores and sorts candidate actions *before* evaluating them:
- Forward moves toward the goal (highest priority).
- Lateral or backward moves (medium priority).
- Wall placements (lowest priority).

By evaluating the best moves first, the Alpha-Beta algorithm can prune the rest of the tree much earlier.

### 2. Transposition Table & Zobrist Hashing
In a game like Quoridor, players can reach the exact same board state through different sequences of moves (transpositions). To avoid evaluating the same board twice, the bot uses a **Transposition Table (TT)**.

- **Zobrist Hashing:** A highly optimized technique that assigns a unique 64-bit integer (`uint64`) to every possible board state. It uses pre-generated random numbers for player positions, wall placements, and turn ownership.
- When the bot evaluates a node, it saves the result (Score and Depth) in the TT using the Zobrist hash as the key. If it encounters the same hash later, it instantly returns the cached score.

### 3. Search Space Reduction (Wall Filtering)
Generating up to 128 wall placements per node explodes the game tree. The bot aggressively filters wall generation:
- **Tactical Window:** Walls are only evaluated if they are placed within a bounding box (± 2 cells) around the current positions of the players. Walls placed far away at the edges of the board are ignored.
- **Wall Search Depth Cutoff:** At the very bottom of the search tree (when remaining depth is small, e.g., `1`), the bot stops generating wall placements entirely and only calculates pawn moves. This drastically reduces the branching factor at the widest part of the tree.

### 4. Zero-Allocation BFS (Memory Optimization)
Calculating the real distance to the goal requires a Breadth-First Search (BFS) algorithm. Because this is called millions of times per turn, standard map/slice allocations would crash the performance by triggering the Garbage Collector (GC).
The bot uses a **Zero-Allocation BFS**:
- A static `9x9` array is used for the `visited` map.
- A fixed-size array is used for the BFS queue.
- This ensures that calculating distances consumes `0` extra memory allocations per call (`0 allocs/op`).

## Notes / Learnings

- **No Machine Learning:** This bot was not trained. Its intelligence comes from sheer brute-force search depth and standard game-theory algorithms.
- **Anti-looping Mechanism:** Just like the heuristic bot, this bot tracks its last 6 actual moves and applies a penalty (`getRepetitionPenalty`) to candidate moves that revisit recent squares. However, this penalty is applied *only at the root level* (the actual move being decided), not deep inside the Minimax hypothetical tree, preventing the bot from needlessly stalling itself in its own imagination.
- **Depth Progression & Performance:** The initial, unoptimized version of this bot running at `MaxDepth = 3` took around 11 seconds per move. After implementing the optimizations (especially the zero-allocation BFS and Zobrist hashing), response times plummeted to around 1 second or less. 
- **Role in the Game:** This massive performance gain allowed us to increase the search depth to 4 in real games, and even run experimental tests at depths 5 and 6. In the actual game, players can choose to play against Minimax 3 or Minimax 4. Furthermore, these optimized versions were used as benchmark rivals during the genetic training of the Heuristic bot (see [Heuristic Training Platform](./heuristic-training-platform.md)).

### Final Benchmarks
The following benchmark suite showcases the final optimized performance, highlighting the speed of the sequential depth evaluations and the zero memory allocations achieved in the shortest path calculation:

```text
goos: windows
goarch: amd64
pkg: quoridor/internal/adapters/bot
cpu: Intel(R) Core(TM) Ultra 7 155H
BenchmarkHeuristicBotGetAction-22                                     5526        220454 ns/op        1231 B/op          5 allocs/op
BenchmarkHeuristicBotGetActionParallel-22                           9783        175286 ns/op        1238 B/op          5 allocs/op
BenchmarkHeuristicBotEvaluateState-22                             691024          1632 ns/op           0 B/op          0 allocs/op
BenchmarkMinimaxBotGetActionDepth2-22                               1129        993743 ns/op      189663 B/op        506 allocs/op
BenchmarkMinimaxBotGetActionDepthSweepSequential/depth_1-22        31885         33422 ns/op        2280 B/op          7 allocs/op
BenchmarkMinimaxBotGetActionDepthSweepSequential/depth_2-22         1660        773190 ns/op      189664 B/op        507 allocs/op
BenchmarkMinimaxBotGetActionDepthSweepSequential/depth_3-22           68      18909222 ns/op     1477977 B/op       3240 allocs/op
BenchmarkMinimaxBotGetActionDepthSweepSequential/depth_4-22           15      96787573 ns/op    10995293 B/op      28619 allocs/op
BenchmarkMinimaxBotShortestPathLength-22                          314391          3657 ns/op           0 B/op          0 allocs/op
PASS
coverage: 59.4% of statements
ok      quoridor/internal/adapters/bot  23.300s
```