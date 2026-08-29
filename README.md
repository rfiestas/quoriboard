```bash
               ___  _   _  ___  ___ ___ ___  ___   _   ___ ___              
 ___ ___ ___ / _ \| | | |/ _ \| _ \_ _| _ )/ _ \ /_\ | _ \   \ ___ ___ ___ 
|___|___|___| (_) | |_| | (_) |   /| || _ \ (_) / _ \|   / |) |___|___|___|
             \__\_\___/ \___/|_|_\___|___/\___/_/ \_\_|_\___/
```

Quoridor implemented in **Go** using the **Ebitengine** library. The project features a full 2D interactive UI, local human-vs-human matches, and a wide array of AI bots ranging from random agents and Minimax with Alpha-Beta pruning, to heuristic bots trained via genetic algorithms and Reinforcement Learning models.

---

## 🚧 Status: Alpha (Work in Progress)

The game is **fully playable and operational**. However, it is currently in **Alpha stage** with several planned enhancements underway:
- [ ] Sound effects (SFX) and audio feedback.
- [ ] Maximum move limit per match (to prevent infinite looping games).
- [ ] Graphical polishes and UI refinements.

---

## 🎮 Game Preview

![Ongoing Quoriboard match](/assets/captures/game-human-vs-human.png)
*Screenshot of a human vs. human match: 9x9 board with cardboard texture, turn/action mode panel on the right, and controls (L-Click to move/place, W to toggle wall mode, R/Space to rotate wall).*

---

## 🚀 Getting Started

### Prerequisites
- [Go 1.22+](https://go.dev/doc/install) installed on your system.

### Running the Game
To launch the game locally, run the following command from the project root:

```bash
go run cmd/game/main.go
```

---

## 🤖 Available AI Bots & Documentation

You can play against several AI algorithms implemented in the project:

- **[Heuristic Bot](./heuristic-bot.md):** Evaluates positions using weighted board features. Includes both a basic handcrafted version and a **genetically trained version**.
- **[Minimax Bot](./minimax-bot.md):** Game-tree search using Alpha-Beta pruning, move ordering, Zobrist hashing, and zero-allocation BFS. Available in **Depth 2**, **Depth 3**, and **Depth 4** difficulties.
- **[Random Bot](./random-bot.md):** Makes random legal moves/wall placements. Used primarily to validate domain interfaces and baseline behavior.
- **[Greedy Path Bot](./greedy-path-bot-en.md):** Pure shortest-path ruleset bot *(Note: currently inactive in the game UI selection)*.
- **RL Bot (v8 Go):** A bot learning via **Reinforcement Learning**. *Note: The model is currently actively training; playing against it uses an early checkpoint, so its moves will appear erratic. Documentation is currently in progress.*

---

## 🧬 AI Training Platforms

- **[Heuristic Training Platform](./heuristic-training-platform.md):** Describes the genetic algorithm framework used to evolve optimal weights for the Heuristic Bot through automated multi-bot tournaments.
- **Reinforcement Learning Training Pipeline:** *(Work in progress — documentation coming soon).*
