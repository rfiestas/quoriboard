package main

import (
	"flag"
	"log"

	server "quoridor/internal/grpc"
)

func main() {
	port := flag.String("port", "50051", "Puerto del servidor gRPC")
	configPath := flag.String("config", "configs/rl_trainer.yaml", "Ruta al YAML de entrenamiento RL")
	flag.Parse()

	log.Printf("Iniciando servidor de entrenamiento Quoridor RL en el puerto %s...", *port)

	if err := server.StartGRPCServer(*port, *configPath); err != nil {
		log.Fatalf("Error crítico al arrancar el servidor gRPC: %v", err)
	}
}
