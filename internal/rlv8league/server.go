package rlv8league

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"strings"

	"quoridor/internal/adapters/bot"
	"quoridor/internal/domain"
	pb "quoridor/internal/rlv8/pb"
	"quoridor/internal/rlv8/translator"

	"google.golang.org/grpc"
)

const agentPlayerID = 1

const repetitionLimit = 3 // misma posición exacta N veces en el episodio -> tablas forzadas

const (
	drawReasonNone = iota
	drawReasonTimeout
	drawReasonRepetition
)

type Server struct {
	pb.UnimplementedQuoridorEnvServer
	game       *domain.Game
	turns      int
	done       bool
	winner     int
	mode       string
	opponent   domain.PlayerController
	history    map[string]int
	drawReason int
}

func NewServer(mode string) (*Server, error) {
	mode = strings.ToLower(strings.TrimSpace(mode))
	opponent, err := newOpponent(mode)
	if err != nil {
		return nil, err
	}
	return &Server{mode: mode, opponent: opponent}, nil
}

func randomChoiceConfig(configs ...func() bot.BotConfig) bot.BotConfig {
	return configs[rand.Intn(len(configs))]()
}

func newOpponent(mode string) (domain.PlayerController, error) {
	var opponent domain.PlayerController
	switch mode {
	case "self":
		// Python owns this opponent because the PPO model lives in Python.
		return nil, nil
	case "heuristic_chaotic":
		opponent = bot.NewHeuristicBot(bot.Bot_Chaotic(), 0)
	case "heuristic_balanced":
		opponent = bot.NewHeuristicBot(bot.Bot_Balanced(), 0)
	case "heuristic_aggressive":
		opponent = bot.NewHeuristicBot(bot.Bot_Aggressive(), 0)
	case "heuristic_defensive":
		opponent = bot.NewHeuristicBot(bot.Bot_Defensive(), 0)
	case "heuristic_tier_beats_heuristic":
		opponent = bot.NewHeuristicBot(randomChoiceConfig(
			bot.Bot_E6G5M74, bot.Bot_E6G5M81, bot.Bot_E6G5M41, bot.Bot_E6G5M39, bot.Bot_E6G5M86,
		), 0)
	case "heuristic_tier_beats_minimax3":
		opponent = bot.NewHeuristicBot(randomChoiceConfig(
			bot.Bot_E4G10M52, bot.Bot_E5G4M77, bot.Bot_E5G4M80, bot.Bot_E6G4M92, bot.Bot_E6G4M33,
		), 0)
	case "heuristic_tier_beats_minimax4":
		opponent = bot.NewHeuristicBot(randomChoiceConfig(
			bot.Bot_E6G5M81_b, bot.Bot_E6G3M75, bot.Bot_E6G4M97, bot.Bot_E6G4M71, bot.Bot_E6G4M29,
			bot.Bot_E1G2M90, bot.Bot_E1G3M77,
		), 0)
	case "heuristic":
		opponent = bot.NewHeuristicBot(bot.DefaultBotConfig(), 0)
	case "minimax1":
		opponent = bot.NewMinimaxBot(0, 1)
	case "minimax2":
		opponent = bot.NewMinimaxBot(0, 2)
	case "minimax3":
		opponent = bot.NewMinimaxBot(0, 3)
	case "greedy_path":
		opponent = bot.NewGreedyPathBot()
	case "mixed":
		mixedModes := []string{
			"heuristic",
			"heuristic_chaotic",
			"heuristic_balanced",
			"heuristic_aggressive",
			"heuristic_defensive",
			"greedy_path",
		}
		chosenMode := mixedModes[rand.Intn(len(mixedModes))]

		// Llamamos recursivamente a la misma función con el modo elegido para mantener el switch limpio
		return newOpponent(chosenMode)
	default:
		return nil, fmt.Errorf("unsupported Go opponent mode %q; use self, heuristic, heuristic_chaotic, heuristic_balanced, heuristic_aggressive, heuristic_defensive, heuristic_tier_beats_heuristic, heuristic_tier_beats_minimax3, heuristic_tier_beats_minimax4, minimax1, minimax2, minimax3, or mixed", mode)
	}
	return opponent, nil
}

func (s *Server) Reset(context.Context, *pb.ResetRequest) (*pb.EnvState, error) {
	s.game = domain.NewGame()
	s.game.State = domain.StatePlaying
	s.game.InitBoard([]domain.PlayerConfig{{ID: 1}, {ID: 2}})
	s.turns, s.done, s.winner = 0, false, 0
	s.history = make(map[string]int)
	s.drawReason = drawReasonNone
	var err error
	s.opponent, err = newOpponent(s.mode)
	if err != nil {
		return nil, err
	}
	return s.advanceOpponent(), nil
}

func (s *Server) Step(_ context.Context, request *pb.StepRequest) (*pb.EnvState, error) {
	if s.game == nil || s.done {
		return nil, fmt.Errorf("ppo_a136_league episode is not active")
	}
	if s.game.ActivePlayer() == nil || s.game.ActivePlayer().ID != agentPlayerID {
		return nil, fmt.Errorf("ppo_a136_league is waiting for its Go opponent")
	}
	if request.GetAction() < 0 || request.GetAction() >= translator.ActionSpaceSize {
		return nil, fmt.Errorf("action %d outside [0,%d)", request.GetAction(), translator.ActionSpaceSize-1)
	}
	player := s.game.ActivePlayer()
	actionMask := translator.BuildActionMask(s.game, player)
	if !actionMask[request.GetAction()] {
		return nil, fmt.Errorf("action %d is invalid", request.GetAction())
	}
	s.apply(player, translator.DecodeAction(s.game, player, int(request.GetAction())))
	s.finishOrNext(player)
	return s.advanceOpponent(), nil
}

func (s *Server) advanceOpponent() *pb.EnvState {
	if s.opponent == nil {
		return s.state()
	}
	for !s.done && s.game.ActivePlayer() != nil && s.game.ActivePlayer().ID != agentPlayerID {
		player := s.game.ActivePlayer()
		action := s.opponent.GetAction(s.game, player)
		if action.Type == domain.ActionNone {
			break
		}
		s.apply(player, action)
		s.finishOrNext(player)
	}
	return s.state()
}

func (s *Server) apply(player *domain.Player, action domain.PlayerAction) {
	switch action.Type {
	case domain.ActionMove:
		player.GridX, player.GridY = action.TargetX, action.TargetY
	case domain.ActionPlaceWall:
		s.game.PlaceWall(action.TargetX, action.TargetY, action.WallOrientation, player)
	}
	s.turns++
}

func (s *Server) finishOrNext(player *domain.Player) {
	if s.game.CheckWin(player) {
		s.done, s.winner, s.drawReason = true, player.ID, drawReasonNone
		s.game.State = domain.StateWin
		return
	}
	key := s.stateKey()
	s.history[key]++
	if s.history[key] >= repetitionLimit {
		s.done, s.winner, s.drawReason = true, 0, drawReasonRepetition
		s.game.State = domain.StateFinished
		return
	}
	if s.turns >= translator.MaxTurns {
		s.done, s.winner, s.drawReason = true, 0, drawReasonTimeout
		s.game.State = domain.StateFinished
		return
	}
	s.game.NextTurn()
}

// stateKey identifica una posición exacta: ambos jugadores, a quién le toca
// mover y el conjunto de muros. Como los muros nunca se retiran, una
// posición solo puede repetirse mientras no se coloque un muro nuevo entre
// medio, así que no hace falta ordenar s.game.Walls.
func (s *Server) stateKey() string {
	p1, p2 := s.game.Players[0], s.game.Players[1]
	var sb strings.Builder
	fmt.Fprintf(&sb, "%d,%d|%d,%d|%d", p1.GridX, p1.GridY, p2.GridX, p2.GridY, s.game.TurnIndex)
	for _, w := range s.game.Walls {
		fmt.Fprintf(&sb, "|%d,%d,%d", w.GridX, w.GridY, w.Orientation)
	}
	return sb.String()
}

func (s *Server) state() *pb.EnvState {
	current := s.game.ActivePlayer()
	if current == nil {
		return &pb.EnvState{Done: s.done}
	}
	opponent := s.game.Players[0]
	if opponent.ID == current.ID {
		opponent = s.game.Players[1]
	}
	return &pb.EnvState{
		Observation:   translator.BuildObservation(s.game, current, s.turns),
		ActionMask:    maskOrEmpty(s.game, current, s.done),
		Done:          s.done,
		CurrentPlayer: int32(current.ID),
		Info: map[string]float32{
			"turns":          float32(s.turns),
			"walls_placed":   float32(len(s.game.Walls)),
			"winner":         float32(s.winner),
			"draw_reason":    float32(s.drawReason),
			"agent_path_len": float32(s.game.ShortestPathLen(s.game.Players[agentPlayerID-1])),
			"opp_path_len":   float32(s.game.ShortestPathLen(opponent)),
		},
	}
}

func maskOrEmpty(game *domain.Game, player *domain.Player, done bool) []bool {
	if done {
		return make([]bool, translator.ActionSpaceSize)
	}
	return translator.BuildActionMask(game, player)
}

func StartGRPCServer(port, mode string) error {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}
	server, err := NewServer(mode)
	if err != nil {
		_ = listener.Close()
		return err
	}
	grpcServer := grpc.NewServer()
	pb.RegisterQuoridorEnvServer(grpcServer, server)
	return grpcServer.Serve(listener)
}
