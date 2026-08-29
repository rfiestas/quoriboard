package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"quoridor/internal/rlv8league"

	"gopkg.in/yaml.v3"
)

type config struct {
	GRPCTarget   string `yaml:"grpc_target"`
	OpponentMode string `yaml:"opponent_mode"`
}

func main() {
	configPath := flag.String("config", filepath.Join("configs", "ppo_a136_league.yaml"), "Path to the league training config")
	pythonExe := flag.String("python", "", "Python executable to use for the RL trainer")
	serverOnly := flag.Bool("server-only", false, "Only start the parallel Go environment server")
	flag.Parse()

	var cfg config
	raw, err := os.ReadFile(*configPath)
	if err != nil || yaml.Unmarshal(raw, &cfg) != nil {
		cfg = config{GRPCTarget: "localhost:50052", OpponentMode: "minimax2"}
	}
	if cfg.GRPCTarget == "" {
		cfg.GRPCTarget = "localhost:50052"
	}
	if cfg.OpponentMode == "" {
		cfg.OpponentMode = "minimax"
	}
	port := cfg.GRPCTarget
	if index := strings.LastIndex(port, ":"); index >= 0 {
		port = port[index+1:]
	}
	go func() {
		if err := rlv8league.StartGRPCServer(port, cfg.OpponentMode); err != nil {
			fmt.Fprintf(os.Stderr, "failed to start parallel Go server: %v\n", err)
			os.Exit(1)
		}
	}()
	if *serverOnly {
		select {}
	}
	time.Sleep(250 * time.Millisecond)

	py := *pythonExe
	if py == "" {
		if runtime.GOOS == "windows" {
			py = filepath.Join(".venv", "Scripts", "python.exe")
			if _, err := os.Stat(py); err != nil {
				py = "python"
			}
		} else {
			py = filepath.Join(".venv", "bin", "python3")
			if _, err := os.Stat(py); err != nil {
				py = "python3"
			}
		}
	}

	cmd := exec.Command(py, "-m", "python_ppo_a136_league.train", "--config", *configPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	fmt.Printf("Starting ppo_a136_league trainer with config %s\n", *configPath)
	fmt.Printf("Using Python executable: %s\n", py)

	if err := runTrainer(cmd); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start trainer: %v\n", err)
		os.Exit(1)
	}
}

func runTrainer(cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return err
	}

	exit := make(chan error, 1)
	go func() {
		exit <- cmd.Wait()
	}()

	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(interrupts)

	select {
	case err := <-exit:
		return err
	case <-interrupts:
		fmt.Println("Interrupt received; asking Python trainer to save and exit...")
		if err := cmd.Process.Signal(os.Interrupt); err != nil {
			fmt.Fprintf(os.Stderr, "could not signal Python trainer: %v\n", err)
		}
		select {
		case err := <-exit:
			return err
		case <-time.After(30 * time.Second):
			_ = cmd.Process.Kill()
			return fmt.Errorf("Python trainer did not exit after interrupt")
		}
	}
}
