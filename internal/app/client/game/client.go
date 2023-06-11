package client

import (
	"bufio"
	"context"
	"io"
	"log"
	"net"
	"os"
	"strings"

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

	Player      *proto.Player
	Wr          *color.Color
	reader      *bufio.Reader
	cmdChan     chan string
	gameEndChan chan struct{}

	CmdPack *GameCommandPack
	Role    string
	Alive   bool
	Day     bool
	lastCmd []string
}

func CreateGameClient(serverAddr string, Player *proto.Player) *GameClient {
	grpcConn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to server at addr %v!", serverAddr)
	}
	grpcConn.Connect()
	cli := proto.NewGameClient(grpcConn)

	return &GameClient{
		Client:      cli,
		grpcConn:    grpcConn,
		Player:      Player,
		Wr:          color.New(color.FgHiRed, color.Italic, color.Bold),
		reader:      bufio.NewReader(os.Stdin),
		Alive:       true,
		cmdChan:     make(chan string),
		gameEndChan: make(chan struct{}),
		Day:         true,
		lastCmd:     make([]string, 0),
	}
}

func (c *GameClient) Close() {
	c.Client.Exit(context.TODO(), &proto.ExitRequest{Player: c.Player})
	c.grpcConn.Close()
	c.chatConn.Close()
	close(c.cmdChan)
}

func (c *GameClient) HandleGameEvents() {
	str, err := c.Client.SubscribeToGameEvent(context.TODO(), &proto.SubscribeToGameRequest{Player: c.Player})
	if err != nil {
		log.Fatalf("Can't subscribe to game events: %v", err)
	}
	for {
		e, err := str.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("HandleGameEvents error: %v", err)
		}

		switch e.Type {
		case "day":
			c.Day = !c.Day
			if c.Alive {
				if c.Day {
					c.CmdPack = GetDayCommandPack()
				} else {
					switch c.Role {
					case "maf":
						c.CmdPack = GetNightMafiaCommandPack()
						c.Wr.Println("Настало время для поиска жертвы! Используйте команду !kill для убийства игрока\n")
					case "com":
						c.CmdPack = GetNightComCommandPack()
						c.Wr.Println("Настало время для поиска мафии! Используйте команду !check для проверки игрока\n")
					case "civ":
						c.CmdPack = GetNightCivCommandPack()
						c.Wr.Println("Полная луна за окном навевает тревогу...\n")
					default:
						log.Fatalf("Unknown role %v", c.Role)
					}
				}
			}
		case "kill":
			c.Wr.Printf("Тело игрока %v утром было найдено в канаве...\n\n", e.GetKilled().Player)
		case "jail":
			c.Wr.Printf("Игрок %v вздёрнут на площади!\n\n", e.GetJailed().Player)
		case "end":
			ev := e.GetEnd()
			c.Wr.Println("Игра окончена!")
			if ev.Won == "maf" {
				c.Wr.Println("Победу одержала мафия!")
			} else {
				c.Wr.Println("Победу одержали мирные жители!")
			}
			c.Wr.Println("Распределение по ролям:")
			for i := range ev.PlayerNames {
				c.Wr.Printf("#%v. %v - ", i+1, ev.PlayerNames[i])
				PrintRole(ev.Roles[i])
				c.Wr.Print("\n")
			}
			close(c.gameEndChan)
		case "dead":
			c.Alive = false
			c.Wr.Println("Вы мертвы! :(\n")
			c.CmdPack = GetDeadCommandPack()
		default:
			log.Printf("Unknown event type: %v", e.Type)
		}
	}
}

func (c *GameClient) PrepareForGame() {
	go c.HandleGameEvents()

	roleResp, err := c.Client.Role(context.TODO(), &proto.RoleRequest{Player: c.Player})
	if err != nil {
		log.Fatal("Can't get Role in PrepareForGame")
	}
	c.Role = roleResp.Role
	c.Wr.Print("Игра начинается!\n\nВаша роль: ")
	PrintRole(c.Role)
	c.Wr.Print("!\n\n")

	c.CmdPack = GetDayCommandPack()
	c.Wr.Print("Для списка доступных команд введите !help\n\n")
}

func (c *GameClient) ReadCmd() (string, bool) {
	go func() {
		c.Wr.Print("Введите команду или сообщение в чат:\n")
		txt, err := c.reader.ReadString('\n')
		if err != nil {
			log.Fatalf("GameClient::ReadCmd error: %v", err)
		}
		c.cmdChan <- strings.Join(strings.Fields(txt), " ")
	}()

	select {
	case cmd := <-c.cmdChan:
		return cmd, false
	case <-c.gameEndChan:
		return "", true
	}
}

func (c *GameClient) Run() bool {
	c.Wr.Println("Введите любую строку для продолжения...")
	c.reader.ReadLine()
	terminal.ClearScreen()
	c.PrepareForGame()

	for {
		cmd, fin := c.ReadCmd()
		if fin {
			// [TODO] ask for continue
			return true
		}

		if len(cmd) == 0 {
			continue
		}
		if cmd[0] == '!' {
			c.lastCmd = strings.Split(cmd, " ")
			command, err := c.CmdPack.GetCommand(c.lastCmd[0])
			if err != nil {
				c.Wr.Print("Неправильная команда! Введите !help для списка команд\n")
			} else {
				command.Run(c)
			}
		} else {
			if !c.Alive {
				c.Wr.Print("Дуновение ветра с кладбища напомнило прохожему о вас...")
				continue
			}
			if !c.Day {
				c.Wr.Print("Вы попытались нарушить ночную тишину, но никто вас не услышал...\n")
				continue
			}

			_, err := c.Client.SendMessage(context.TODO(), &proto.SendMessageRequest{Msg: cmd, Player: c.Player})
			if err != nil {
				log.Printf("Can't send msg, error: %v\n", err)
			}
		}

		c.Wr.Print("\n")
	}
}
