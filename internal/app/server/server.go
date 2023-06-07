package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/GandarfHSE/go-mafia/internal/proto"
	"github.com/GandarfHSE/go-mafia/internal/utils/player"
	"github.com/GandarfHSE/go-mafia/internal/utils/role"
)

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

func (s *LobbyServer) Close() {
	// [TODO] Make destructor
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
	log.Printf("Broadcast message: %v\n", msg)
	for _, p := range s.players {
		p.SendMsg(msg)
	}
}

func (s *LobbyServer) broadcastMsgFromPlayer(msg string, addr string, name string) {
	s.broadcastMsg(fmt.Sprintf("%v##[%v] %v", addr, name, msg))
}

func (s *LobbyServer) broadcastMsgFromServer(msg string) {
	s.broadcastMsg(fmt.Sprintf("server##[server] %v", msg))
}

func (s *LobbyServer) Join(_ context.Context, req *proto.JoinRequest) (*proto.Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.addPlayer(req.Player)
	if err != nil {
		return nil, err
	}

	msg := fmt.Sprintf("Игрок %v успешно присоединился к лобби!", req.Player.Name)
	s.broadcastMsgFromPlayer(msg, req.Player.Addr, req.Player.Name)
	return &proto.Empty{}, nil
}

func (s *LobbyServer) MemberList(_ context.Context, _ *proto.Empty) (*proto.MemberListResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	playerNames := make([]string, 0)
	for _, pl := range s.players {
		playerNames = append(playerNames, pl.Name)
	}
	return &proto.MemberListResponse{PlayerNames: playerNames}, nil
}

func (s *LobbyServer) SendMessage(_ context.Context, req *proto.SendMessageRequest) (*proto.Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.broadcastMsgFromPlayer(req.Msg, req.Player.Addr, req.Player.Name)
	return &proto.Empty{}, nil
}

func (s *LobbyServer) Exit(_ context.Context, req *proto.ExitRequest) (*proto.Empty, error) {
	if len(s.players) <= 0 {
		return nil, errors.New("Weird shit")
	}

	pind := -1
	for i, p := range s.players {
		if p.Addr == req.Player.Addr && p.Name == req.Player.Name {
			pind = i
			break
		}
	}

	if pind == -1 {
		return nil, errors.New("Player is not found!")
	}

	s.players[pind] = s.players[len(s.players)-1]
	s.players = s.players[:len(s.players)-1]

	s.broadcastMsgFromPlayer(fmt.Sprintf("Игрок %v отключился!", req.Player.Name), req.Player.Addr, req.Player.Name)

	return &proto.Empty{}, nil
}
