package client

import (
	"context"
	"errors"
	"log"
	"strconv"

	"github.com/GandarfHSE/go-mafia/internal/proto"
	"github.com/fatih/color"
)

type GameCommand interface {
	Name() string
	Args() string
	Descr() string
	Run(c *GameClient)
}

type GameCommandPack struct {
	cmds []GameCommand
}

func (pack *GameCommandPack) PrintHelp(w *color.Color) {
	w.Print("Список команд:\n")
	for _, cmd := range pack.cmds {
		w.Printf("%v %v - %v\n", cmd.Name(), cmd.Args(), cmd.Descr())
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

func GetDayCommandPack() *GameCommandPack {
	return CreateCommandPack([]GameCommand{
		&GameCommandList{},
		&GameCommandRole{},
		&GameCommandExit{},
		&GameCommandAlive{},
		&GameCommandVote{},
	})
}

func GetNightMafiaCommandPack() *GameCommandPack {
	return CreateCommandPack([]GameCommand{
		&GameCommandList{},
		&GameCommandRole{},
		&GameCommandExit{},
		&GameCommandAlive{},
		&GameCommandKill{},
	})
}

func GetNightComCommandPack() *GameCommandPack {
	return CreateCommandPack([]GameCommand{
		&GameCommandList{},
		&GameCommandRole{},
		&GameCommandExit{},
		&GameCommandAlive{},
		&GameCommandCheck{},
	})
}

func GetNightCivCommandPack() *GameCommandPack {
	return CreateCommandPack([]GameCommand{
		&GameCommandList{},
		&GameCommandRole{},
		&GameCommandExit{},
		&GameCommandAlive{},
	})
}

func GetDeadCommandPack() *GameCommandPack {
	return CreateCommandPack([]GameCommand{
		&GameCommandList{},
		&GameCommandRole{},
		&GameCommandExit{},
		&GameCommandAlive{},
	})
}

// ===================

type GameCommandHelp struct {
	pack *GameCommandPack
}

func (cmd *GameCommandHelp) Name() string {
	return "!help"
}

func (cmd *GameCommandHelp) Args() string {
	return ""
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

func (cmd *GameCommandList) Args() string {
	return ""
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

func (cmd *GameCommandExit) Args() string {
	return ""
}

func (cmd *GameCommandExit) Descr() string {
	return "Выйти из игры"
}

func (cmd *GameCommandExit) Run(c *GameClient) {
	_, err := c.Client.Exit(context.TODO(), &proto.ExitRequest{Player: c.Player})
	if err != nil {
		log.Printf("Exit err: %v\n", err)
		return
	}
	c.gameEndChan <- struct{}{}
}

// ===================

type GameCommandRole struct {
}

func (cmd *GameCommandRole) Name() string {
	return "!role"
}

func (cmd *GameCommandRole) Args() string {
	return ""
}

func (cmd *GameCommandRole) Descr() string {
	return "Напомнить свою роль"
}

func (cmd *GameCommandRole) Run(c *GameClient) {
	resp, err := c.Client.Role(context.TODO(), &proto.RoleRequest{Player: c.Player})
	c.Wr.Print("Ваша роль: ")
	PrintRole(resp.Role)
	c.Wr.Print("!\n")
	if err != nil {
		log.Printf("Role err: %v\n", err)
		return
	}
}

// ===================

type GameCommandVote struct {
}

func (cmd *GameCommandVote) Name() string {
	return "!vote"
}

func (cmd *GameCommandVote) Args() string {
	return "<pid>"
}

func (cmd *GameCommandVote) Descr() string {
	return "Проголосовать за посадку игрока pid (pid = 0 означает пропуск голосования)"
}

func (cmd *GameCommandVote) Run(c *GameClient) {
	if len(c.lastCmd) < 2 {
		c.Wr.Print("Слишком мало аргументов для команды kill!\n")
		return
	}

	pid, err := strconv.Atoi(c.lastCmd[1])
	_, err = c.Client.Vote(context.TODO(), &proto.VoteRequest{Player: c.Player, Voting: int32(pid - 1)})
	if err != nil {
		c.Wr.Printf("Произошла ошибка при обработке голосования: %v\n", err)
		return
	}
}

// ===================

type GameCommandKill struct {
}

func (cmd *GameCommandKill) Name() string {
	return "!kill"
}

func (cmd *GameCommandKill) Args() string {
	return "<pid>"
}

func (cmd *GameCommandKill) Descr() string {
	return "Убить игрока pid"
}

func (cmd *GameCommandKill) Run(c *GameClient) {
	if len(c.lastCmd) < 2 {
		c.Wr.Print("Слишком мало аргументов для команды kill!\n")
		return
	}

	pid, err := strconv.Atoi(c.lastCmd[1])
	_, err = c.Client.Kill(context.TODO(), &proto.KillRequest{Player: c.Player, Killing: int32(pid - 1)})
	if err != nil {
		c.Wr.Printf("Произошла ошибка при убийстве игрока: %v\n", err)
		return
	}
	c.Wr.Printf("Кажется, игрок #%v не доживёт до утра...\n", pid)
}

// ===================

type GameCommandCheck struct {
}

func (cmd *GameCommandCheck) Name() string {
	return "!check"
}

func (cmd *GameCommandCheck) Args() string {
	return "<pid>"
}

func (cmd *GameCommandCheck) Descr() string {
	return "Узнать роль игрока pid"
}

func (cmd *GameCommandCheck) Run(c *GameClient) {
	if len(c.lastCmd) < 2 {
		c.Wr.Print("Слишком мало аргументов для команды check!\n")
		return
	}

	pid, err := strconv.Atoi(c.lastCmd[1])
	resp, err := c.Client.Check(context.TODO(), &proto.CheckRequest{Player: c.Player, Checking: int32(pid - 1)})
	if err != nil {
		c.Wr.Printf("Произошла ошибка при проверке игрока: %v\n", err)
		return
	}
	c.Wr.Printf("Прошерстив архивные документы, вы выяснили, что роль игрока #%v - ", pid)
	PrintRole(resp.Role)
	c.Wr.Print("!\n")
}

// ===================

type GameCommandAlive struct {
}

func (cmd *GameCommandAlive) Name() string {
	return "!alive"
}

func (cmd *GameCommandAlive) Args() string {
	return ""
}

func (cmd *GameCommandAlive) Descr() string {
	return "Вывести список живых игроков"
}

func (cmd *GameCommandAlive) Run(c *GameClient) {
	resp, err := c.Client.AliveList(context.TODO(), &proto.Empty{})
	if err != nil {
		c.Wr.Printf("Произошла ошибка при получении списка живых игроков: %v\n", err)
		return
	}
	c.Wr.Print("Оставшиеся игроки:\n")
	for i := range resp.PlayerNames {
		c.Wr.Printf("%v. %v\n", resp.Pids[i]+1, resp.PlayerNames[i])
	}
}
