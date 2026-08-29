package rlv8

import (
	"context"
	"fmt"
	"net"

	"quoridor/internal/domain"
	pb "quoridor/internal/rlv8/pb"
	"quoridor/internal/rlv8/translator"

	"google.golang.org/grpc"
)

const (
	ActionSpaceSize = translator.ActionSpaceSize
	ObservationSize = translator.ObservationSize
	maxTurns        = translator.MaxTurns
)

type Server struct {
	pb.UnimplementedQuoridorEnvServer
	game      *domain.Game
	turns     int
	done      bool
	winner    int
	lastActor int
}

func NewServer() *Server { return &Server{} }

func (s *Server) Reset(context.Context, *pb.ResetRequest) (*pb.EnvState, error) {
	s.game = domain.NewGame()
	s.game.State = domain.StatePlaying
	s.game.InitBoard([]domain.PlayerConfig{{ID: 1}, {ID: 2}})
	s.turns, s.done, s.winner, s.lastActor = 0, false, 0, 0
	return s.state(0), nil
}

func (s *Server) Step(_ context.Context, request *pb.StepRequest) (*pb.EnvState, error) {
	if s.game == nil || s.done {
		return nil, fmt.Errorf("rlv8 episode is not active")
	}
	if request.GetAction() < 0 || request.GetAction() >= ActionSpaceSize {
		return nil, fmt.Errorf("action %d outside [0,%d)", request.GetAction(), ActionSpaceSize-1)
	}
	actor := s.game.ActivePlayer()
	if actor == nil {
		return nil, fmt.Errorf("rlv8 game has no active player")
	}
	mask := s.actionMask(actor)
	if !mask[request.GetAction()] {
		return nil, fmt.Errorf("action %d is invalid", request.GetAction())
	}
	action := s.realAction(actor, int(request.GetAction()))
	if action < 8 {
		s.applyMove(actor, action)
	} else {
		s.applyWall(actor, action)
	}
	s.lastActor = actor.ID
	s.turns++
	if s.game.CheckWin(actor) {
		s.finish(actor.ID)
	} else if s.turns >= maxTurns {
		s.finish(0)
	} else {
		s.game.NextTurn()
	}
	return s.state(rewardFor(s.winner, actor.ID, s.done)), nil
}

func (s *Server) realAction(player *domain.Player, action int) int {
	return translator.RealAction(player, action)
}

func (s *Server) finish(winner int) {
	s.done, s.winner = true, winner
	s.game.State = domain.StateFinished
	if winner != 0 {
		s.game.State = domain.StateWin
	}
}

func rewardFor(winner, actor int, done bool) float32 {
	if !done {
		return 0
	}
	if winner == actor {
		return 1
	}
	if winner == 0 {
		return -0.1
	}
	return -1
}

func (s *Server) applyMove(player *domain.Player, direction int) {
	var moves [8]domain.Move
	count := s.game.GetValidMovesInto(player, &moves)
	dx, dy := translator.MoveDirection(direction)
	for i := 0; i < count; i++ {
		move := moves[i]
		moveDX, moveDY := move.X-player.GridX, move.Y-player.GridY
		var match bool
		if direction < 4 {
			// Cardinal: coincide por signo (cubre paso simple de 1 casilla
			// y salto recto de 2 casillas sobre el rival).
			match = (dx == 0 && moveDX == 0 && moveDY*dy > 0) || (dy == 0 && moveDY == 0 && moveDX*dx > 0)
		} else {
			// Diagonal: coincide por delta exacto (siempre magnitud 1 en ambos ejes).
			match = moveDX == dx && moveDY == dy
		}
		if match {
			player.GridX, player.GridY = move.X, move.Y
			return
		}
	}
}

func (s *Server) applyWall(player *domain.Player, action int) {
	index := action - 8
	orientation := domain.WallHorizontal
	if index >= 64 {
		orientation, index = domain.WallVertical, index-64
	}
	s.game.PlaceWall(index/8, index%8, orientation, player)
}

func (s *Server) actionMask(player *domain.Player) []bool {
	return translator.BuildActionMask(s.game, player)
}

func (s *Server) state(reward float32) *pb.EnvState {
	current := s.game.ActivePlayer()
	playerID := 0
	// Nota: en el flujo normal de Python, esta función solo se consulta para
	// decidir el reward cuando !done, momento en el cual "current" ya es
	// garantizadamente el turno del agente RL (el wrapper ya jugó los turnos
	// del oponente histórico antes de devolver esta respuesta). Por eso
	// "agent_path_len"/"opp_path_len" son seguros de nombrar así aquí.
	var agentPathLen, oppPathLen float32 = -1, -1
	if current != nil {
		playerID = current.ID
		opponent := s.game.Players[0]
		if opponent.ID == current.ID {
			opponent = s.game.Players[1]
		}
		agentPathLen = float32(s.game.ShortestPathLen(current))
		oppPathLen = float32(s.game.ShortestPathLen(opponent))
	}
	return &pb.EnvState{Observation: s.observation(current), ActionMask: s.actionMaskOrEmpty(current), Reward: reward, Done: s.done, CurrentPlayer: int32(playerID), Info: map[string]float32{
		"turns": float32(s.turns), "walls_placed": float32(len(s.game.Walls)), "winner": float32(s.winner),
		"agent_path_len": agentPathLen, "opp_path_len": oppPathLen,
	}}
}

func (s *Server) actionMaskOrEmpty(player *domain.Player) []bool {
	if s.done || player == nil {
		return make([]bool, ActionSpaceSize)
	}
	return s.actionMask(player)
}

func (s *Server) observation(current *domain.Player) []float32 {
	return translator.BuildObservation(s.game, current, s.turns)
}

func StartGRPCServer(port string) error {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}
	grpcServer := grpc.NewServer()
	pb.RegisterQuoridorEnvServer(grpcServer, NewServer())
	return grpcServer.Serve(listener)
}
