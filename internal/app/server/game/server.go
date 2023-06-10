package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/GandarfHSE/go-mafia/internal/proto"
	"github.com/GandarfHSE/go-mafia/internal/utils/algo"
	"github.com/GandarfHSE/go-mafia/internal/utils/player"
)

type GameServer struct {
	proto.UnimplementedGameServer

	players []player.Player
	mu      sync.Mutex

	Day bool
}

func CreateGameServer(Players []player.Player) *GameServer {
	roles := algo.Shuffle([]string{
		"civ", "civ", "maf", "com",
	})
	if len(roles) != len(Players) {
		log.Fatal("Число ролей не совпадает с числом игроков!")
	}
	for i, role := range roles {
		Players[i].Role = role
	}

	return &GameServer{
		players: Players,
	}
}

func (s *GameServer) Close() {
	// [TODO] Make destructor
}

// [TODO] make version with color
func (s *GameServer) broadcastMsg(msg string) {
	log.Printf("Broadcast message: %v\n", msg)
	for _, p := range s.players {
		p.SendMsg(msg)
	}
}

func (s *GameServer) broadcastMsgFromPlayer(msg string, addr string, name string) {
	s.broadcastMsg(fmt.Sprintf("%v##[%v] %v", addr, name, msg))
}

func (s *GameServer) broadcastMsgFromServer(msg string) {
	s.broadcastMsg(fmt.Sprintf("server##[server] %v", msg))
}

func (s *GameServer) MemberList(ctx context.Context, _ *proto.Empty) (*proto.MemberListResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	playerNames := make([]string, 0)
	for _, pl := range s.players {
		playerNames = append(playerNames, pl.Name)
	}
	return &proto.MemberListResponse{PlayerNames: playerNames}, nil
}

func (s *GameServer) SendMessage(_ context.Context, req *proto.SendMessageRequest) (*proto.Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.broadcastMsgFromPlayer(req.Msg, req.Player.Addr, req.Player.Name)
	return &proto.Empty{}, nil
}

func (s *GameServer) Exit(_ context.Context, req *proto.ExitRequest) (*proto.Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.killPlayer(req.Player.Addr, req.Player.Name)
	s.broadcastMsgFromPlayer(fmt.Sprintf("Игрок %v отключился!", req.Player.Name), req.Player.Addr, req.Player.Name)

	return &proto.Empty{}, nil
}

func (s *GameServer) killPlayer(addr string, name string) {
	log.Printf("Убиваем игрока %v...\n", name)
	for _, p := range s.players {
		if p.Addr == addr && p.Name == name {
			if !p.Alive {
				log.Println("Убиваем мертвого!")
			}
			p.Alive = false
			return
		}
	}
}

func (s *GameServer) Role(_ context.Context, req *proto.RoleRequest) (*proto.RoleResponse, error) {
	for _, p := range s.players {
		if p.Addr == req.Player.Addr && p.Name == req.Player.Name {
			return &proto.RoleResponse{Role: p.Role}, nil
		}
	}
	return nil, errors.New("Игрок не найден!")
}
