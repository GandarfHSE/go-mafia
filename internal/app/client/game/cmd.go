package client

import (
	"context"
	"errors"
	"log"

	"github.com/GandarfHSE/go-mafia/internal/proto"
	"github.com/fatih/color"
)

type GameCommand interface {
	Name() string
	Descr() string
	Run(c *GameClient)
}

type GameCommandPack struct {
	cmds []GameCommand
}

func (pack *GameCommandPack) PrintHelp(w *color.Color) {
	w.Print("Список команд:\n")
	for _, cmd := range pack.cmds {
		w.Printf("%v - %v\n", cmd.Name(), cmd.Descr())
	}
}

func (pack *GameCommandPack) GetCommand(s string) (GameCommand, error) {
	for _, cmd := range pack.cmds {
		if cmd.Name() == s {
			return cmd, nil
		}
	}

	return nil, errors.New("No command found!")
}

func CreateCommandPack(cmds []GameCommand) *GameCommandPack {
	cmdPack := &GameCommandPack{cmds: cmds}
	cmdPack.cmds = append(cmdPack.cmds, &GameCommandHelp{pack: cmdPack})
	return cmdPack
}

func GetDefaultCommandPack() *GameCommandPack {
	return CreateCommandPack([]GameCommand{
		&GameCommandList{},
		&GameCommandRole{},
		&GameCommandExit{},
	})
}

type GameCommandHelp struct {
	pack *GameCommandPack
}

func (cmd *GameCommandHelp) Name() string {
	return "!help"
}

func (cmd *GameCommandHelp) Descr() string {
	return "Вывести список команд"
}

func (cmd *GameCommandHelp) Run(c *GameClient) {
	cmd.pack.PrintHelp(c.Wr)
}

// ===================

type GameCommandList struct {
}

func (cmd *GameCommandList) Name() string {
	return "!list"
}

func (cmd *GameCommandList) Descr() string {
	return "Вывести список всех игроков"
}

func (cmd *GameCommandList) Run(c *GameClient) {
	resp, err := c.Client.MemberList(context.TODO(), &proto.Empty{})
	if err != nil {
		log.Printf("MemberList error: %v\n", err)
		return
	}

	c.Wr.Printf("Игроков в игре: [%v/4]\n", len(resp.PlayerNames))
	for ind, name := range resp.PlayerNames {
		c.Wr.Printf("%v. %v\n", ind+1, name)
	}
}

// ===================

type GameCommandExit struct {
}

func (cmd *GameCommandExit) Name() string {
	return "!exit"
}

func (cmd *GameCommandExit) Descr() string {
	return "Выйти из игры"
}

func (cmd *GameCommandExit) Run(c *GameClient) {
	_, err := c.Client.Exit(context.TODO(), &proto.ExitRequest{Player: c.Player})
	if err != nil {
		log.Printf("Exit err: %v\n", err)
	}
}

// ===================

type GameCommandRole struct {
}

func (cmd *GameCommandRole) Name() string {
	return "!role"
}

func (cmd *GameCommandRole) Descr() string {
	return "Напомнить свою роль"
}

func (cmd *GameCommandRole) Run(c *GameClient) {
	resp, err := c.Client.Role(context.TODO(), &proto.RoleRequest{Player: c.Player})
	PrintRole(resp.Role)
	if err != nil {
		log.Printf("Exit err: %v\n", err)
	}
}
