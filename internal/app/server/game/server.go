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

	state  *GameState
	closed bool
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

	playersCopy := make([]player.Player, len(Players))
	copy(playersCopy, Players)
	s := &GameServer{
		players: playersCopy,
		state:   nil,
	}
	s.state = CreateGameState(s)
	return s
}

func (s *GameServer) CheckVictory() bool {
	mafAlive := 0
	civAlive := 0

	for _, p := range s.players {
		if p.Alive {
			if p.Role == "maf" {
				mafAlive += 1
			} else {
				civAlive += 1
			}
		}
	}

	if mafAlive == 0 {
		s.broadcastEvent(&proto.GameEvent{Type: "end", Event: &proto.GameEvent_End{End: &proto.GameEnd{Won: "civ", PlayerNames: s.getPlayerNames(), Roles: s.getPlayerRoles()}}})
		return true
	}
	if mafAlive >= civAlive {
		s.broadcastEvent(&proto.GameEvent{Type: "end", Event: &proto.GameEvent_End{End: &proto.GameEnd{Won: "maf", PlayerNames: s.getPlayerNames(), Roles: s.getPlayerRoles()}}})
		return true
	}
	return false
}

func (s *GameServer) Run() {
	for {
		s.broadcastMsgFromServer("Новый день - новое голосование!\n")
		s.state.SetupNewDay()
		jailed := <-s.state.VoteChan
		if s.state.VotesCount[jailed] > 0 {
			s.killPlayer(jailed)
			s.broadcastEvent(&proto.GameEvent{Type: "jail", Event: &proto.GameEvent_Jailed{Jailed: &proto.PlayerJailed{Player: s.players[jailed].Name}}})
		} else {
			s.broadcastMsgFromServer("Сегодня виновных не нашлось, попробуем завтра...")
		}
		if s.CheckVictory() {
			return
		}
		s.broadcastEvent(&proto.GameEvent{Type: "day", Event: &proto.GameEvent_Day{Day: &proto.DayChange{}}})

		s.broadcastMsgFromServer("Город засыпает...\n")
		s.state.SetupNewNight()
		killed := <-s.state.KillChan
		<-s.state.CheckChan
		s.killPlayer(killed)
		s.broadcastEvent(&proto.GameEvent{Type: "kill", Event: &proto.GameEvent_Killed{Killed: &proto.PlayerKilled{Player: s.players[killed].Name}}})
		if s.CheckVictory() {
			return
		}
		s.broadcastEvent(&proto.GameEvent{Type: "day", Event: &proto.GameEvent_Day{Day: &proto.DayChange{}}})
	}
}

func (s *GameServer) Close() {
	if s.closed {
		return
	}

	log.Println("Closing game server...")
	s.closed = true
	for _, p := range s.players {
		close(p.GameEventChan)
	}
	s.state.Close()
}

func (s *GameServer) SubscribeToGameEvent(req *proto.SubscribeToGameRequest, event_stream proto.Game_SubscribeToGameEventServer) error {
	pind := s.getPid(req.Player)
	if pind == -1 {
		return errors.New("SubscribeToGameEvent: player not found!")
	}

	for {
		e, ok := <-s.players[pind].GameEventChan
		if !ok {
			return nil
		}
		event_stream.Send(e)
	}
}

func (s *GameServer) getPid(pl *proto.Player) int {
	for i, p := range s.players {
		if pl.Name == p.Name {
			return i
		}
	}
	return -1
}

func (s *GameServer) broadcastEvent(e *proto.GameEvent) {
	// Here we can write into closed chan, if server is closed, so we need to handle panic
	defer func() {
		if recover() != nil {
			log.Print("Panic catched in broadcastEvent")
		}
	}()

	log.Printf("Broadcast event: %v\n", e)
	for _, p := range s.players {
		p.GameEventChan <- e
	}
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

func (s *GameServer) getPlayerNames() []string {
	playerNames := make([]string, 0)
	for _, pl := range s.players {
		playerNames = append(playerNames, pl.Name)
	}
	return playerNames
}

func (s *GameServer) getPlayerRoles() []string {
	playerRoles := make([]string, 0)
	for _, pl := range s.players {
		playerRoles = append(playerRoles, pl.Role)
	}
	return playerRoles
}

func (s *GameServer) MemberList(ctx context.Context, _ *proto.Empty) (*proto.MemberListResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return &proto.MemberListResponse{PlayerNames: s.getPlayerNames()}, nil
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

	if !s.closed {
		s.killPlayer(s.getPid(req.Player))
		s.broadcastMsgFromPlayer(fmt.Sprintf("Игрок %v отключился!", req.Player.Name), req.Player.Addr, req.Player.Name)
	}

	return &proto.Empty{}, nil
}

func (s *GameServer) killPlayer(pid int) bool {
	log.Printf("Убиваем игрока %v...\n", s.players[pid].Name)
	if pid == -1 {
		log.Printf("Trying to kill unknown player: %v", pid)
		return false
	}
	if !s.players[pid].Alive {
		log.Println("Убиваем мертвого!")
		return false
	}
	s.players[pid].Alive = false
	s.state.TryFinishVote()
	s.players[pid].GameEventChan <- &proto.GameEvent{Type: "dead", Event: &proto.GameEvent_Dead{}}
	if s.players[pid].Role == "com" {
		s.state.CheckChanClosed = true
		close(s.state.CheckChan)
	}
	return true
}

func (s *GameServer) Role(_ context.Context, req *proto.RoleRequest) (*proto.RoleResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pid := s.getPid(req.Player)
	if pid == -1 {
		return nil, errors.New("Игрок не найден!")
	}
	return &proto.RoleResponse{Role: s.players[pid].Role}, nil
}

func (s *GameServer) Vote(_ context.Context, req *proto.VoteRequest) (*proto.Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pid := s.getPid(req.Player)
	if !s.players[pid].Alive {
		return nil, errors.New("Vote: голос от мертвеца")
	}

	vid := int(req.Voting)
	err := s.state.Vote(pid, vid)
	if err == nil {
		if vid != -1 {
			s.broadcastMsgFromServer(fmt.Sprintf("Игрок #%v проголосовал против игрока #%v!", pid+1, vid+1))
		} else {
			s.broadcastMsgFromServer(fmt.Sprintf("Игрок #%v решил не голосовать!", pid+1))
		}
	}

	return &proto.Empty{}, err
}

func (s *GameServer) Kill(_ context.Context, req *proto.KillRequest) (*proto.Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pid := s.getPid(req.Player)
	kid := int(req.Killing)
	if !s.players[pid].Alive {
		return nil, errors.New("Kill: голос от мертвеца")
	}
	if s.players[pid].Role != "maf" {
		return nil, errors.New("Kill: голос от мирного")
	}
	if !s.players[kid].Alive {
		return nil, errors.New("Kill: игрок уже мёртв!")
	}

	err := s.state.Kill(pid, kid)
	if err != nil {
		return nil, err
	}
	if !s.killPlayer(kid) {
		return nil, errors.New("Kill: weird shit")
	}
	return &proto.Empty{}, nil
}

func (s *GameServer) Check(_ context.Context, req *proto.CheckRequest) (*proto.CheckResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Println("Check...")

	pid := s.getPid(req.Player)
	cid := int(req.Checking)
	if !s.players[pid].Alive {
		return nil, errors.New("Check: проверка от мертвеца")
	}
	if s.players[pid].Role != "com" {
		return nil, errors.New("Check: проверка не от копа")
	}

	err := s.state.Check(pid, cid)
	if err != nil {
		return nil, err
	}

	log.Println("Check end...")

	return &proto.CheckResponse{Role: s.players[cid].Role}, nil
}

func (s *GameServer) AliveList(_ context.Context, _ *proto.Empty) (*proto.AliveListResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	playerNames := make([]string, 0)
	pids := make([]int32, 0)
	for i, pl := range s.players {
		if pl.Alive {
			playerNames = append(playerNames, pl.Name)
			pids = append(pids, int32(i))
		}
	}
	return &proto.AliveListResponse{PlayerNames: playerNames, Pids: pids}, nil
}
