package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	game "github.com/GandarfHSE/go-mafia/internal/app/server/game"
	"github.com/GandarfHSE/go-mafia/internal/proto"
	"github.com/GandarfHSE/go-mafia/internal/utils/algo"
	"github.com/GandarfHSE/go-mafia/internal/utils/player"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

const (
	GamePlayers int = 4
)

type LobbyServer struct {
	proto.UnimplementedLobbyServer

	players []player.Player
	mu      sync.Mutex

	gameWG   sync.WaitGroup
	gameAddr string
}

func CreateLobbyServer() *LobbyServer {
	lobby := &LobbyServer{
		players:  nil,
		gameAddr: "",
	}
	lobby.gameWG.Add(GamePlayers)
	return lobby
}

func (s *LobbyServer) Close() {
	// [TODO] Make destructor
}

func (s *LobbyServer) addPlayer(pbplayer *proto.Player) error {
	for _, p := range s.players {
		if p.Name == pbplayer.Name {
			return errors.New("Игрок с таким именем уже существует!")
		}
	}

	conn, err := net.Dial("udp", pbplayer.Addr)
	if err != nil {
		log.Printf("Can't add player %v with addr %v!\n", pbplayer.Name, pbplayer.Addr)
		return err
	}

	newPlayer := player.Player{
		Name:          pbplayer.Name,
		Addr:          pbplayer.Addr,
		Conn:          conn,
		Role:          "anon",
		Alive:         true,
		GameEventChan: make(chan *proto.GameEvent),
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

func (s *LobbyServer) Join(ctx context.Context, req *proto.JoinRequest) (*proto.Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("Join request from %v with addr %v\n", req.Player.Name, req.Player.Addr)

	p, _ := peer.FromContext(ctx)
	port := 0
	fmt.Sscanf(req.Player.Addr, ":%d", &port)
	switch addr := p.Addr.(type) {
	case *net.UDPAddr:
		addr.Port = port
		req.Player.Addr = addr.String()
	case *net.TCPAddr:
		addr.Port = port
		req.Player.Addr = addr.String()
	}

	err := s.addPlayer(req.Player)
	if err != nil {
		return nil, err
	}

	msg := fmt.Sprintf("Игрок %v успешно присоединился к лобби!", req.Player.Name)
	s.broadcastMsgFromPlayer(msg, req.Player.Addr, req.Player.Name)

	if len(s.players) == GamePlayers {
		// start game
		s.PrepareGame()
	}
	s.gameWG.Done()

	return &proto.Empty{}, nil
}

func (s *LobbyServer) MemberList(ctx context.Context, _ *proto.Empty) (*proto.MemberListResponse, error) {
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
	s.mu.Lock()
	defer s.mu.Unlock()

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

	s.players = algo.Erase(s.players, pind)
	s.gameWG.Add(1)
	s.broadcastMsgFromPlayer(fmt.Sprintf("Игрок %v отключился!", req.Player.Name), req.Player.Addr, req.Player.Name)

	return &proto.Empty{}, nil
}

func (s *LobbyServer) SubscribeToGame(_ context.Context, _ *proto.Empty) (*proto.SubscribeToGameResponse, error) {
	s.gameWG.Wait()
	time.Sleep(time.Second)
	defer s.gameWG.Add(1)
	return &proto.SubscribeToGameResponse{GameAddr: s.gameAddr}, nil
}

func (s *LobbyServer) PrepareGame() {
	log.Println("Preparing game...")

	rnd := rand.New(rand.NewSource(time.Now().Unix()))

	var lis net.Listener
	var err error

	for i := 0; i < 5; i++ {
		// [TODO] Get this from config
		port := 9000 + rnd.Uint32()%100
		s.gameAddr = fmt.Sprintf(":%v", port)
		lis, err = net.Listen("tcp", s.gameAddr)
		if err == nil {
			break
		}
	}
	log.Printf("Start game server at %v\n", s.gameAddr)

	gameServer := game.CreateGameServer(s.players)
	grpcServer := grpc.NewServer()
	proto.RegisterGameServer(grpcServer, gameServer)
	go func() {
		gameServer.Run()
		gameServer.Close()
	}()
	go func() {
		grpcServer.Serve(lis)
	}()
	s.broadcastMsgFromServer("Лобби заполнено...")
	s.players = nil
}
