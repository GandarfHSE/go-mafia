package player

import (
	"fmt"
	"net"

	"github.com/GandarfHSE/go-mafia/internal/proto"
)

type Player struct {
	Name string
	Addr string
	Conn net.Conn

	Role  string
	Alive bool

	GameEventChan chan *proto.GameEvent
}

// [TODO] make version with color
func (p Player) SendMsg(msg string) {
	_, err := fmt.Fprint(p.Conn, msg)
	if err != nil {
		fmt.Printf("Can't send msg to %v, err: %v", p.Name, err)
	}
}
