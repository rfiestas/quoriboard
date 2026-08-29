package ebitenui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"

	"quoridor/internal/domain"
	"quoridor/internal/rlv8/translator"
)

type rlv8PythonInferenceRequest struct {
	Observation []float32 `json:"observation"`
	ActionMask  []bool    `json:"action_mask"`
}

type rlv8PythonInferenceResponse struct {
	Action int    `json:"action"`
	Error  string `json:"error,omitempty"`
}

type RLV8PythonController struct {
	process *exec.Cmd
	stdin   io.WriteCloser
	output  *bufio.Scanner
}

func NewRLV8PythonController(python, checkpoint string) (*RLV8PythonController, error) {
	process := exec.Command(python, "-m", "python_agent_v8.infer", "--checkpoint", checkpoint)
	stdin, err := process.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("open RL V8 Python stdin: %w", err)
	}
	stdout, err := process.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("open RL V8 Python stdout: %w", err)
	}
	process.Stderr = os.Stderr
	if err := process.Start(); err != nil {
		return nil, fmt.Errorf("start RL V8 Python inference: %w", err)
	}
	return &RLV8PythonController{
		process: process,
		stdin:   stdin,
		output:  bufio.NewScanner(stdout),
	}, nil
}

func (c *RLV8PythonController) GetAction(game *domain.Game, player *domain.Player) domain.PlayerAction {
	mask := translator.BuildActionMask(game, player)
	request := rlv8PythonInferenceRequest{
		Observation: translator.BuildObservation(game, player, minRLV8Turns(game.Turns)),
		ActionMask:  mask,
	}
	payload, err := json.Marshal(request)
	if err != nil {
		log.Printf("[RLV8 Python] marshal request failed: %v", err)
		return domain.PlayerAction{}
	}
	if _, err := c.stdin.Write(append(payload, '\n')); err != nil {
		log.Printf("[RLV8 Python] write to stdin failed: %v", err)
		return domain.PlayerAction{}
	}
	if !c.output.Scan() {
		log.Printf("[RLV8 Python] read from stdout failed: %v", c.output.Err())
		return domain.PlayerAction{}
	}

	var response rlv8PythonInferenceResponse
	if err := json.Unmarshal(c.output.Bytes(), &response); err != nil {
		log.Printf("[RLV8 Python] invalid response: %v", err)
		return domain.PlayerAction{}
	}
	if response.Error != "" || response.Action < 0 || response.Action >= len(mask) || !mask[response.Action] {
		if response.Error != "" {
			log.Printf("[RLV8 Python] inference failed: %s", response.Error)
		}
		return domain.PlayerAction{}
	}
	return translator.DecodeAction(game, player, response.Action)
}

func (c *RLV8PythonController) Close() error {
	if c == nil || c.process == nil {
		return nil
	}
	_ = c.stdin.Close()
	return c.process.Process.Kill()
}

func minRLV8Turns(turns int) int {
	if turns > translator.MaxTurns {
		return translator.MaxTurns
	}
	return turns
}
