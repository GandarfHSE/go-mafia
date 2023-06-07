package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/GandarfHSE/go-mafia/internal/app/server"
	"github.com/GandarfHSE/go-mafia/internal/proto"
	"google.golang.org/grpc"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer cancel()

	// [TODO] Get server port from config
	lis, err := net.Listen("tcp", "localhost:8085")
	if err != nil {
		log.Fatal("Failed to listen!")
	}
	log.Printf("Listening on localhost:8085....")

	lobbyServer := server.CreateLobbyServer()
	defer lobbyServer.Close()

	grpcServer := grpc.NewServer()
	proto.RegisterLobbyServer(grpcServer, lobbyServer)
	log.Printf("Serving grpc server...")
	go grpcServer.Serve(lis)

	<-ctx.Done()
}
