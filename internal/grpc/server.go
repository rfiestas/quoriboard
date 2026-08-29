package server

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"sort"
	"strings"
	"time"

	"quoridor/internal/adapters/bot"
	"quoridor/internal/domain"
	pb "quoridor/internal/pb"

	"google.golang.org/grpc"
	"gopkg.in/yaml.v3"
)

const ActionSpaceSize = 140
const StateTensorSize = 491

var moveOffsets = [12][2]int{
	{0, -1}, {0, 1}, {-1, 0}, {1, 0},
	{0, -2}, {0, 2}, {-2, 0}, {2, 0},
	{-1, -1}, {1, -1}, {-1, 1}, {1, 1},
}

type QuoridorServer struct {
	pb.UnimplementedQuoridorEnvServer
	game           *domain.Game
	agentPlayer    *domain.Player
	opponentPlayer *domain.Player
	activeOpponent domain.PlayerController
	lastMyDist     int
	lastOppDist    int
	totalSteps     int
	maxSteps       int
	stagePools     map[int][]weightedOpponent
	rewardCfg      rewardConfig
}

type weightedOpponent struct {
	botName string
	weight  float64
}

type serverYAMLConfig struct {
	RLServer rlServerConfig `yaml:"rl_server"`
}

type rlServerConfig struct {
	MaxSteps int                         `yaml:"max_steps"`
	Stages   map[int][]weightedOpponentY `yaml:"stages"`
	Reward   rewardConfigY               `yaml:"reward"`
}

type rewardConfig struct {
	WinReward         float32
	LoseReward        float32
	TimeoutReward     float32
	ProgressMyWeight  float32
	ProgressOppWeight float32
	WallBonus2Plus    float32
	WallBonus1        float32
	WallPenalty0      float32
}

type rewardConfigY struct {
	WinReward         float32 `yaml:"win_reward"`
	LoseReward        float32 `yaml:"lose_reward"`
	TimeoutReward     float32 `yaml:"timeout_reward"`
	ProgressMyWeight  float32 `yaml:"progress_my_weight"`
	ProgressOppWeight float32 `yaml:"progress_opp_weight"`
	WallBonus2Plus    float32 `yaml:"wall_bonus_2plus"`
	WallBonus1        float32 `yaml:"wall_bonus_1"`
	WallPenalty0      float32 `yaml:"wall_penalty_0"`
}

type weightedOpponentY struct {
	Bot    string  `yaml:"bot"`
	Weight float64 `yaml:"weight"`
}

func defaultStagePools() map[int][]weightedOpponent {
	return map[int][]weightedOpponent{
		1: {{botName: "random", weight: 1.0}},
		2: {{botName: "minimax1", weight: 0.7}, {botName: "random", weight: 0.3}},
		3: {{botName: "minimax2", weight: 0.7}, {botName: "minimax1", weight: 0.2}, {botName: "random", weight: 0.1}},
		4: {{botName: "minimax3", weight: 0.4}, {botName: "minimax2", weight: 0.2}, {botName: "minimax1", weight: 0.3}, {botName: "random", weight: 0.1}},
	}
}

func defaultRLServerConfig() rlServerConfig {
	return rlServerConfig{
		MaxSteps: 150,
		Reward: rewardConfigY{
			WinReward:         2.0,
			LoseReward:        -2.0,
			TimeoutReward:     -1.0,
			ProgressMyWeight:  0.02,
			ProgressOppWeight: -0.02,
			WallBonus2Plus:    0.05,
			WallBonus1:        0.01,
			WallPenalty0:      -0.02,
		},
		Stages: map[int][]weightedOpponentY{
			1: {{Bot: "random", Weight: 1.0}},
			2: {{Bot: "minimax1", Weight: 0.7}, {Bot: "random", Weight: 0.3}},
			3: {{Bot: "minimax2", Weight: 0.7}, {Bot: "minimax1", Weight: 0.2}, {Bot: "random", Weight: 0.1}},
			4: {{Bot: "minimax3", Weight: 0.4}, {Bot: "minimax2", Weight: 0.2}, {Bot: "minimax1", Weight: 0.3}, {Bot: "random", Weight: 0.1}},
		},
	}
}

func rewardConfigFromYAML(y rewardConfigY) rewardConfig {
	def := defaultRLServerConfig().Reward

	r := rewardConfig{
		WinReward:         y.WinReward,
		LoseReward:        y.LoseReward,
		TimeoutReward:     y.TimeoutReward,
		ProgressMyWeight:  y.ProgressMyWeight,
		ProgressOppWeight: y.ProgressOppWeight,
		WallBonus2Plus:    y.WallBonus2Plus,
		WallBonus1:        y.WallBonus1,
		WallPenalty0:      y.WallPenalty0,
	}

	if r.WinReward == 0 {
		r.WinReward = def.WinReward
	}
	if r.LoseReward == 0 {
		r.LoseReward = def.LoseReward
	}
	if r.TimeoutReward == 0 {
		r.TimeoutReward = def.TimeoutReward
	}
	if r.ProgressMyWeight == 0 {
		r.ProgressMyWeight = def.ProgressMyWeight
	}
	if r.ProgressOppWeight == 0 {
		r.ProgressOppWeight = def.ProgressOppWeight
	}
	if r.WallBonus2Plus == 0 {
		r.WallBonus2Plus = def.WallBonus2Plus
	}
	if r.WallBonus1 == 0 {
		r.WallBonus1 = def.WallBonus1
	}
	if r.WallPenalty0 == 0 {
		r.WallPenalty0 = def.WallPenalty0
	}

	return r
}

func normalizeServerConfig(cfg rlServerConfig) rlServerConfig {
	def := defaultRLServerConfig()
	if cfg.MaxSteps <= 0 {
		cfg.MaxSteps = def.MaxSteps
	}
	if len(cfg.Stages) == 0 {
		cfg.Stages = def.Stages
	}
	return cfg
}

func loadRLServerConfig(path string) rlServerConfig {
	if strings.TrimSpace(path) == "" {
		return defaultRLServerConfig()
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		log.Printf("[GO SERVER] No se pudo leer config YAML '%s': %v. Usando defaults.", path, err)
		return defaultRLServerConfig()
	}

	var cfg serverYAMLConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		log.Printf("[GO SERVER] YAML inválido en '%s': %v. Usando defaults.", path, err)
		return defaultRLServerConfig()
	}

	return normalizeServerConfig(cfg.RLServer)
}

func buildStagePoolsFromConfig(cfg rlServerConfig) map[int][]weightedOpponent {
	pools := make(map[int][]weightedOpponent, len(cfg.Stages))
	for stage, items := range cfg.Stages {
		for _, it := range items {
			name := strings.ToLower(strings.TrimSpace(it.Bot))
			if name == "" {
				continue
			}
			w := it.Weight
			if w <= 0 {
				w = 1.0
			}
			pools[stage] = append(pools[stage], weightedOpponent{botName: name, weight: w})
		}
	}
	if len(pools) == 0 {
		return defaultStagePools()
	}
	return pools
}

func NewQuoridorServer(configPath string) *QuoridorServer {
	rand.Seed(time.Now().UnixNano())
	cfg := loadRLServerConfig(configPath)
	stagePools := buildStagePoolsFromConfig(cfg)
	return &QuoridorServer{
		totalSteps: 0,
		maxSteps:   cfg.MaxSteps,
		stagePools: stagePools,
		rewardCfg:  rewardConfigFromYAML(cfg.Reward),
	}
}

func (s *QuoridorServer) newBotByName(name string) domain.PlayerController {
	switch name {
	case "random":
		return bot.NewRandomBot(0.3)
	case "minimax1":
		return bot.NewMinimaxBot(0, 0)
		//fmt.Printf("[GO SERVER] Usando HeuristicBot en lugar de Minimax1 para '%s'\n", name)
		//return bot.NewHeuristicBot(bot.Bot_Defensive(), 0)
	case "minimax2":
		//return bot.NewMinimaxBot(0, 2)
		return bot.NewHeuristicBot(bot.Bot_Chaotic(), 0)
	case "minimax3":
		//return bot.NewMinimaxBot(0, 3)
		return bot.NewHeuristicBot(bot.Bot_Balanced(), 0)
	case "heuristic":
		//return bot.NewHeuristicBot(bot.Bot_E1G4M38(), 0)
		return bot.NewHeuristicBot(bot.Bot_Aggressive(), 0)
	default:
		return bot.NewRandomBot(0.3)
	}
}

func (s *QuoridorServer) resolveStage(stageID int) int {
	if len(s.stagePools) == 0 {
		return stageID
	}

	if _, ok := s.stagePools[stageID]; ok {
		return stageID
	}

	keys := make([]int, 0, len(s.stagePools))
	for k := range s.stagePools {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	if stageID <= keys[0] {
		return keys[0]
	}
	for _, k := range keys {
		if stageID <= k {
			return k
		}
	}
	return keys[len(keys)-1]
}

func (s *QuoridorServer) selectBot(stageID int32) domain.PlayerController {
	resolvedStage := s.resolveStage(int(stageID))
	pool := s.stagePools[resolvedStage]
	if len(pool) == 0 {
		return bot.NewRandomBot(0.3)
	}

	totalWeight := 0.0
	for _, p := range pool {
		totalWeight += p.weight
	}
	if totalWeight <= 0 {
		return s.newBotByName(pool[rand.Intn(len(pool))].botName)
	}

	r := rand.Float64() * totalWeight
	acc := 0.0
	for _, p := range pool {
		acc += p.weight
		if r <= acc {
			return s.newBotByName(p.botName)
		}
	}

	return s.newBotByName(pool[len(pool)-1].botName)
}

func (s *QuoridorServer) Reset(ctx context.Context, req *pb.ResetRequest) (*pb.StepResponse, error) {
	log.Printf("[GO SERVER] Reset recibido. Tipo Oponente: %d", req.GetOpponentType())

	s.game = domain.NewGame()
	s.game.State = domain.StatePlaying
	s.totalSteps = 0

	p1Cfg := domain.PlayerConfig{ID: 1}
	p2Cfg := domain.PlayerConfig{ID: 2}
	s.game.InitBoard([]domain.PlayerConfig{p1Cfg, p2Cfg})

	s.agentPlayer = s.game.Players[0]
	s.opponentPlayer = s.game.Players[1]
	s.activeOpponent = s.selectBot(req.GetOpponentType())

	if s.game.TurnIndex == 1 {
		oppAction := s.activeOpponent.GetAction(s.game, s.opponentPlayer)
		s.applyAction(s.opponentPlayer, oppAction)
		s.game.NextTurn()
	}

	s.lastMyDist = s.getBFSPath(s.agentPlayer)
	s.lastOppDist = s.getBFSPath(s.opponentPlayer)

	return &pb.StepResponse{
		StateTensor: s.encodeStateTensor(),
		ActionMask:  s.getActionMask(s.agentPlayer),
		Reward:      0.0,
		Done:        false,
		Winner:      0,
	}, nil
}

func (s *QuoridorServer) Step(ctx context.Context, req *pb.StepRequest) (*pb.StepResponse, error) {
	if s.game == nil || s.game.State != domain.StatePlaying {
		return nil, fmt.Errorf("la partida ha terminado o no está activa")
	}
	s.totalSteps++

	// 1. TURNO DEL AGENTE
	agentAction := s.decodeAction(req.ActionId, s.agentPlayer)
	s.applyAction(s.agentPlayer, agentAction)

	if s.game.CheckWin(s.agentPlayer) {
		s.game.State = domain.StateWin
		return &pb.StepResponse{
			StateTensor: s.encodeStateTensor(),
			ActionMask:  make([]bool, ActionSpaceSize),
			Reward:      s.rewardCfg.WinReward,
			Done:        true,
			Winner:      int32(s.agentPlayer.ID),
		}, nil
	}

	if s.totalSteps >= s.maxSteps {
		s.game.State = domain.StateFinished
		return &pb.StepResponse{
			StateTensor: s.encodeStateTensor(),
			ActionMask:  make([]bool, ActionSpaceSize),
			Reward:      s.rewardCfg.TimeoutReward,
			Done:        true,
			Winner:      0,
		}, nil
	}

	s.game.NextTurn()
	s.totalSteps++

	// 2. TURNO DEL OPONENTE
	oppAction := s.activeOpponent.GetAction(s.game, s.opponentPlayer)
	s.applyAction(s.opponentPlayer, oppAction)

	if s.game.CheckWin(s.opponentPlayer) {
		s.game.State = domain.StateWin
		return &pb.StepResponse{
			StateTensor: s.encodeStateTensor(),
			ActionMask:  make([]bool, ActionSpaceSize),
			Reward:      s.rewardCfg.LoseReward,
			Done:        true,
			Winner:      int32(s.opponentPlayer.ID),
		}, nil
	}

	if s.totalSteps >= s.maxSteps {
		s.game.State = domain.StateFinished
		return &pb.StepResponse{
			StateTensor: s.encodeStateTensor(),
			ActionMask:  make([]bool, ActionSpaceSize),
			Reward:      s.rewardCfg.TimeoutReward,
			Done:        true,
			Winner:      0,
		}, nil
	}

	s.game.NextTurn()

	// 3. RECOMPENSA EVALUADA TRAS AMBOS MOVIMIENTOS
	myDistAfter := s.getBFSPath(s.agentPlayer)
	oppDistAfter := s.getBFSPath(s.opponentPlayer)

	// En server.go (Paso 3 de Step)
	deltaMyDist := float32(s.lastMyDist - myDistAfter)
	deltaOppDist := float32(oppDistAfter - s.lastOppDist)

	// AQUI: Empezamos asi pero se estanco en EP27500
	// Subir el factor de avance de 0.01 a 0.10
	//reward := (0.10 * deltaMyDist) + (0.05 * deltaOppDist)
	/*reward := (0.10 * deltaMyDist) + (0.10 * deltaOppDist)

	if req.ActionId >= 12 && deltaOppDist <= 0 {
		reward -= 0.02 // Penalizar muro inútil
	}*/

	reward := (s.rewardCfg.ProgressMyWeight * deltaMyDist) + (s.rewardCfg.ProgressOppWeight * deltaOppDist)
	// if req.ActionId >= 12 {
	// 	if deltaOppDist >= 2 {
	// 		reward += s.rewardCfg.WallBonus2Plus
	// 	} else if deltaOppDist == 1 {
	// 		reward += s.rewardCfg.WallBonus1
	// 	} else {
	// 		reward += s.rewardCfg.WallPenalty0
	// 	}
	// }

	s.lastMyDist = myDistAfter
	s.lastOppDist = oppDistAfter

	return &pb.StepResponse{
		StateTensor: s.encodeStateTensor(),
		ActionMask:  s.getActionMask(s.agentPlayer),
		Reward:      reward,
		Done:        false,
		Winner:      0,
	}, nil
}

// Helper para valor absoluto con enteros
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (s *QuoridorServer) getActionMask(p *domain.Player) []bool {
	mask := make([]bool, ActionSpaceSize) // Tamaño 140

	// 1. Movimientos de Peón (Índices 0 a 11)
	var validMoveBuf [8]domain.Move
	validMoveCount := s.game.GetValidMovesInto(p, &validMoveBuf)
	for i := 0; i < validMoveCount; i++ {
		m := validMoveBuf[i]
		dx := m.X - p.GridX
		dy := m.Y - p.GridY

		for i, offset := range moveOffsets {
			if offset[0] == dx && offset[1] == dy {
				mask[i] = true
				break
			}
		}
	}

	// 2. Muros (Índices 12 a 139)
	if p.WallsLeft > 0 {
		for wx := 0; wx < domain.BoardSize-1; wx++ {
			for wy := 0; wy < domain.BoardSize-1; wy++ {
				if s.game.CanPlaceWall(wx, wy, domain.WallHorizontal, p) {
					mask[12+(wx*8)+wy] = true // Muros horizontales (12..75)
				}
				if s.game.CanPlaceWall(wx, wy, domain.WallVertical, p) {
					mask[76+(wx*8)+wy] = true // Muros verticales (76..139)
				}
			}
		}
	}

	return mask
}

func (s *QuoridorServer) decodeAction(actionID int32, p *domain.Player) domain.PlayerAction {
	if actionID < 12 {
		offset := moveOffsets[actionID]
		return domain.PlayerAction{
			Type:    domain.ActionMove,
			TargetX: p.GridX + offset[0],
			TargetY: p.GridY + offset[1],
		}
	} else if actionID < 76 {
		idx := actionID - 12
		return domain.PlayerAction{
			Type:            domain.ActionPlaceWall,
			TargetX:         int(idx / 8),
			TargetY:         int(idx % 8),
			WallOrientation: domain.WallHorizontal,
		}
	} else {
		idx := actionID - 76
		return domain.PlayerAction{
			Type:            domain.ActionPlaceWall,
			TargetX:         int(idx / 8),
			TargetY:         int(idx % 8),
			WallOrientation: domain.WallVertical,
		}
	}
}

func (s *QuoridorServer) applyAction(p *domain.Player, action domain.PlayerAction) {
	// En server.go, dentro de Step(), justo después de applyAction:
	//log.Printf("[DEBUG] Turno %d | Agente WallsLeft: %d", s.totalSteps, s.agentPlayer.WallsLeft)

	if action.Type == domain.ActionMove {
		p.GridX = action.TargetX
		p.GridY = action.TargetY
	} else if action.Type == domain.ActionPlaceWall {
		s.game.PlaceWall(action.TargetX, action.TargetY, action.WallOrientation, p)
	}

}

func (s *QuoridorServer) encodeStateTensor() []float32 {
	tensor := make([]float32, StateTensorSize)

	tensor[s.agentPlayer.GridX*9+s.agentPlayer.GridY] = 1.0
	tensor[81+s.opponentPlayer.GridX*9+s.opponentPlayer.GridY] = 1.0

	for _, w := range s.game.Walls {
		idx := w.GridX*9 + w.GridY
		if w.Orientation == domain.WallHorizontal {
			tensor[162+idx] = 1.0
		} else {
			tensor[243+idx] = 1.0
		}
	}

	agentDistNorm := float32(s.getBFSPath(s.agentPlayer)) / 16.0
	oppDistNorm := float32(s.getBFSPath(s.opponentPlayer)) / 16.0
	for i := 0; i < 81; i++ {
		tensor[324+i] = agentDistNorm
		tensor[405+i] = oppDistNorm
	}

	// Scalar features appended after the 6 board planes.
	agentWallsNorm := float32(s.agentPlayer.WallsLeft) / 10.0
	oppWallsNorm := float32(s.opponentPlayer.WallsLeft) / 10.0
	stepNorm := float32(s.totalSteps) / float32(s.maxSteps)
	if stepNorm > 1.0 {
		stepNorm = 1.0
	}

	tensor[486] = agentWallsNorm
	tensor[487] = oppWallsNorm
	tensor[488] = stepNorm
	tensor[489] = agentDistNorm
	tensor[490] = oppDistNorm

	return tensor
}

func (s *QuoridorServer) getBFSPath(p *domain.Player) int {
	var visited [domain.BoardSize][domain.BoardSize]bool
	type node struct{ x, y, d int }
	queue := make([]node, 0, 81)

	queue = append(queue, node{p.GridX, p.GridY, 0})
	visited[p.GridX][p.GridY] = true

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if (p.TargetY != -1 && curr.y == p.TargetY) || (p.TargetX != -1 && curr.x == p.TargetX) {
			return curr.d
		}

		dirs := [4][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}
		for _, d := range dirs {
			nx, ny := curr.x+d[0], curr.y+d[1]
			if nx >= 0 && nx < domain.BoardSize && ny >= 0 && ny < domain.BoardSize {
				if !visited[nx][ny] && !s.isWallBlocking(curr.x, curr.y, nx, ny) {
					visited[nx][ny] = true
					queue = append(queue, node{nx, ny, curr.d + 1})
				}
			}
		}
	}
	return 999
}

func (s *QuoridorServer) isWallBlocking(x1, y1, x2, y2 int) bool {
	for _, w := range s.game.Walls {
		if w.Orientation == domain.WallHorizontal {
			if (y1 == w.GridY && y2 == w.GridY+1) || (y1 == w.GridY+1 && y2 == w.GridY) {
				if x1 == w.GridX || x1 == w.GridX+1 {
					if x1 == x2 {
						return true
					}
				}
			}
		} else {
			if (x1 == w.GridX && x2 == w.GridX+1) || (x1 == w.GridX+1 && x2 == w.GridX) {
				if y1 == w.GridY || y1 == w.GridY+1 {
					if y1 == y2 {
						return true
					}
				}
			}
		}
	}
	return false
}

func StartGRPCServer(port string, configPath string) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	srv := NewQuoridorServer(configPath)
	pb.RegisterQuoridorEnvServer(grpcServer, srv)

	fmt.Printf("Servidor Quoridor gRPC escuchando en puerto %s...\n", port)
	return grpcServer.Serve(lis)
}
