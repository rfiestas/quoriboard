package main

import (
	"flag"
	"log"

	"quoridor/internal/rlv8"
)

func main() {
	port := flag.String("port", "50052", "RL V8 gRPC port")
	flag.Parse()
	log.Printf("Starting isolated Quoridor RL V8 server on port %s", *port)
	if err := rlv8.StartGRPCServer(*port); err != nil {
		log.Fatal(err)
	}
}
