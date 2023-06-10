package player

import (
	"fmt"
	"net"
)

type Player struct {
	Name string
	Addr string
	Conn net.Conn

	Role  string
	Alive bool
}

// [TODO] make version with color
func (p Player) SendMsg(msg string) {
	_, err := fmt.Fprint(p.Conn, msg)
	if err != nil {
		fmt.Printf("Can't send msg to %v, err: %v", p.Name, err)
	}
}
