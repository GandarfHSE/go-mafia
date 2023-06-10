package client

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/GandarfHSE/go-mafia/internal/proto"
	"github.com/GandarfHSE/go-mafia/internal/utils/terminal"
	"github.com/fatih/color"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GameClient struct {
	Client   proto.GameClient
	grpcConn *grpc.ClientConn
	chatConn *net.UDPConn

	Player *proto.Player
	Wr     *color.Color

	CmdPack *GameCommandPack
	Role    string
	Alive   bool
}

func CreateGameClient(serverAddr string, Player *proto.Player) *GameClient {
	grpcConn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to server at addr %v!", serverAddr)
	}
	grpcConn.Connect()
	cli := proto.NewGameClient(grpcConn)

	return &GameClient{
		Client:   cli,
		grpcConn: grpcConn,
		Player:   Player,
		Wr:       color.New(color.FgHiRed, color.Italic, color.Bold),
		Alive:    true,
	}
}

func (c *GameClient) Close() {
	c.grpcConn.Close()
	c.chatConn.Close()
}

func (c *GameClient) PrepareForGame() {
	// сабы, туда-сюда
	roleResp, err := c.Client.Role(context.TODO(), &proto.RoleRequest{Player: c.Player})
	if err != nil {
		log.Fatal("Can't get Role in PrepareForGame")
	}
	c.Role = roleResp.Role
	c.Wr.Print("Игра начинается!\n\n")
	PrintRole(c.Role)
	c.Wr.Print("\n\n")

	c.CmdPack = GetDefaultCommandPack()
}

func (c *GameClient) Run() bool {
	terminal.ClearScreen()
	c.PrepareForGame()

	for {
		var cmd string
		c.Wr.Print("Введите команду или сообщение в чат:\n> ")
		fmt.Scan(&cmd)

		if len(cmd) == 0 {
			continue
		}
		if cmd[0] == '!' {
			command, err := c.CmdPack.GetCommand(cmd)
			if err != nil {
				c.Wr.Print("Неправильная команда! Введите !help для списка команд\n")
			} else {
				command.Run(c)
			}
		} else {
			_, err := c.Client.SendMessage(context.TODO(), &proto.SendMessageRequest{Msg: cmd, Player: c.Player})
			if err != nil {
				log.Printf("Can't send msg, error: %v\n", err)
			}
		}

		c.Wr.Print("\n")
	}
}
