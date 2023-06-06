package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/GandarfHSE/go-mafia/internal/proto"
	"github.com/GandarfHSE/go-mafia/internal/utils/player"
	"github.com/GandarfHSE/go-mafia/internal/utils/role"
)

// [TODO] Make destructor
type LobbyServer struct {
	proto.UnimplementedLobbyServer

	players []player.Player
	mu      sync.Mutex
}

func CreateLobbyServer() *LobbyServer {
	return &LobbyServer{
		players: nil,
	}
}

func (s *LobbyServer) addPlayer(pbplayer *proto.Player) error {
	conn, err := net.Dial("udp", pbplayer.Addr)
	if err != nil {
		log.Printf("Can't add player %v with addr %v!\n", pbplayer.Name, pbplayer.Addr)
		return err
	}

	newPlayer := player.Player{
		Name: pbplayer.Name,
		Addr: pbplayer.Addr,
		Conn: conn,
		Role: role.CreateRole("roleless"),
	}
	s.players = append(s.players, newPlayer)
	return nil
}

// [TODO] make version with color
func (s *LobbyServer) broadcastMsg(msg string) {
	for _, p := range s.players {
		p.SendMsg(msg)
	}
}

func (s *LobbyServer) Join(_ context.Context, req *proto.JoinRequest) (*proto.JoinResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.addPlayer(req.Player)
	if err != nil {
		return nil, err
	}

	log.Printf("Player with name %v and addr %v has connected!\n", req.Player.Name, req.Player.Addr)
	msg := fmt.Sprintf("[server] Игрок %v успешно присоединился к лобби!\n", req.Player.Name)
	go s.broadcastMsg(msg)
	return &proto.JoinResponse{Resp: msg}, nil
}

func (s *LobbyServer) MemberList(_ context.Context, req *proto.MemberListRequest) (*proto.MemberListResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	playerNames := make([]string, 0)
	for _, pl := range s.players {
		playerNames = append(playerNames, pl.Name)
	}
	return &proto.MemberListResponse{PlayerNames: playerNames}, nil
}
