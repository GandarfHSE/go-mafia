package client

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"

	client "github.com/GandarfHSE/go-mafia/internal/app/client/game"
	"github.com/GandarfHSE/go-mafia/internal/proto"
	"github.com/GandarfHSE/go-mafia/internal/utils/terminal"
	"github.com/fatih/color"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type LobbyClient struct {
	client   proto.LobbyClient
	grpcConn *grpc.ClientConn
	chatConn *net.UDPConn

	player proto.Player
	w      *color.Color
	reader *bufio.Reader

	gameClient *client.GameClient
	gameChan   chan struct{}
	cmdChan    chan string
}

func CreateLobbyClient() *LobbyClient {
	// [TODO] Get server port from config
	serverAddr := ":8085"
	grpcConn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to server at addr %v!", serverAddr)
	}
	grpcConn.Connect()
	cli := proto.NewLobbyClient(grpcConn)

	return &LobbyClient{
		client:     cli,
		grpcConn:   grpcConn,
		player:     proto.Player{},
		w:          color.New(color.FgHiRed, color.Italic, color.Bold),
		gameClient: nil,
		gameChan:   make(chan struct{}),
		cmdChan:    make(chan string),
		reader:     bufio.NewReader(os.Stdin),
	}
}

func (c *LobbyClient) serveChat() {
	var buf [2048]byte
	for {
		msgLen, _, err := c.chatConn.ReadFrom(buf[:])
		if err != nil {
			break
		}
		str := string(buf[0:msgLen])
		splittedMsg := strings.Split(str, "##")

		if len(splittedMsg) < 2 {
			fmt.Printf("len(splittedMsg) < 2!")
			continue
		}

		writer := color.New(color.FgHiWhite)
		switch splittedMsg[0] {
		case "server":
			writer = color.New(color.FgWhite)
		case c.player.Addr:
			continue
		}
		writer.Println(splittedMsg[1])
	}
}

func (c *LobbyClient) WaitForGame() {
	resp, err := c.client.SubscribeToGame(context.TODO(), &proto.Empty{})
	if err != nil {
		log.Fatal("Can't connect to game!")
	}

	c.gameClient = client.CreateGameClient(resp.GameAddr, &c.player)
	c.gameChan <- struct{}{}
}

func (c *LobbyClient) ConnectToLobby() {
	c.w.Println("Подключаюсь к лобби...")

	rnd := rand.New(rand.NewSource(time.Now().Unix()))
	var err error

	for i := 0; i < 10; i++ {
		// [TODO] Get this from config
		port := 2000 + rnd.Uint32()%2000
		c.player.Addr = fmt.Sprintf(":%v", port)
		c.chatConn, err = net.ListenUDP("udp", &net.UDPAddr{Port: int(port)})
		if err == nil {
			break
		}
	}

	go c.serveChat()

	if err != nil {
		log.Fatal("Can't connect to chat!")
	}

	go c.WaitForGame()
	_, err = c.client.Join(context.TODO(), &proto.JoinRequest{Player: &c.player})
	if err != nil {
		log.Fatalf("err in join, err = %v", err)
	}

	c.w.Print("Подключение произошло успешно!")
	fmt.Print("\n\n==================================\n\n")
}

func (c *LobbyClient) Close() {
	c.grpcConn.Close()
	c.chatConn.Close()
	close(c.cmdChan)
	close(c.gameChan)
}

func (c *LobbyClient) Greet() {
	c.w.Println("~ Добро пожаловать в консольное приложение Мафия! ~")
	c.w.Print("Введите своё имя:\n> ")
	fmt.Scan(&c.player.Name)
	c.w.Printf("Здравствуй, %s!\n\n", c.player.Name)
}

func (c *LobbyClient) ReadCmd() (string, bool) {
	go func() {
		c.w.Print("Введите команду или сообщение в чат:\n")
		txt, err := c.reader.ReadString('\n')
		if err != nil {
			log.Fatalf("LobbyClient::ReadCmd error: %v", err)
		}
		c.cmdChan <- strings.Join(strings.Fields(txt), " ")
	}()

	select {
	case cmd := <-c.cmdChan:
		return cmd, false
	case <-c.gameChan:
		return "", true
	}
}

func (c *LobbyClient) Run() {
	terminal.ClearScreen()
	c.Greet()
	c.ConnectToLobby()

	for {
		cmd, _ := c.ReadCmd()

		if c.gameClient != nil {
			f := c.gameClient.Run()
			c.gameClient.Close()
			c.gameClient = nil
			if f {
				return
			}

			_, err := c.client.Join(context.TODO(), &proto.JoinRequest{Player: &c.player})
			if err != nil {
				log.Fatal("err in join")
			}
		}

		// [TODO]: Make command pack
		switch cmd {
		case "!help":
			c.w.Print("Список команд:\n" +
				"!help - Вывести это сообщение\n" +
				"!list - Вывести список всех игроков\n" +
				"!exit - Выйти из игры\n")
		case "!list":
			resp, err := c.client.MemberList(context.TODO(), &proto.Empty{})
			if err != nil {
				log.Printf("MemberList error: %v\n", err)
				continue
			}

			c.w.Printf("Игроков в лобби: [%v/4]\n", len(resp.PlayerNames))
			for ind, name := range resp.PlayerNames {
				c.w.Printf("%v. %v\n", ind+1, name)
			}
		case "!exit":
			_, err := c.client.Exit(context.TODO(), &proto.ExitRequest{Player: &c.player})
			if err != nil {
				log.Printf("Exit err: %v\n", err)
			}
			return
		default:
			if len(cmd) == 0 {
				continue
			}
			if cmd[0] == '!' {
				c.w.Print("Неправильная команда! Введите !help для списка команд\n\n")
				continue
			}

			_, err := c.client.SendMessage(context.TODO(), &proto.SendMessageRequest{Msg: cmd, Player: &c.player})
			if err != nil {
				log.Printf("Can't send msg, error: %v\n", err)
				continue
			}
		}

		c.w.Print("\n")
	}
}
