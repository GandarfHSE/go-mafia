package client

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

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
	player   proto.Player
	w        *color.Color
}

func CreateLobbyClient() *LobbyClient {
	// [TODO] Get server port from config
	serverAddr := "localhost:8085"
	grpcConn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to server at addr %v!", serverAddr)
	}
	grpcConn.Connect()
	cli := proto.NewLobbyClient(grpcConn)

	return &LobbyClient{
		client:   cli,
		grpcConn: grpcConn,
		player:   proto.Player{},
		w:        color.New(color.FgHiRed, color.Italic, color.Bold),
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
		c.w.Println(str)
	}
}

func (c *LobbyClient) ConnectToLobby() {
	c.w.Println("Подключаюсь к лобби...")

	rnd := rand.New(rand.NewSource(time.Now().Unix()))
	var err error

	for i := 0; i < 5; i++ {
		// [TODO] Get this from config
		port := 2000 + rnd.Uint32()%2000
		c.player.Addr = fmt.Sprintf("localhost:%v", port)
		c.chatConn, err = net.ListenUDP("udp", &net.UDPAddr{Port: int(port)})
		if err == nil {
			break
		}
	}

	go c.serveChat()

	if err != nil {
		log.Fatal("Can't connect to chat!")
	}

	_, err = c.client.Join(context.TODO(), &proto.JoinRequest{Player: &c.player})
	if err != nil {
		log.Fatal("err in join")
	}

	c.w.Print("Подключение произошло успешно!")
	fmt.Print("\n\n==================================\n\n")
}

func (c *LobbyClient) Close() {
	c.grpcConn.Close()
	c.chatConn.Close()
}

func (c *LobbyClient) Greet() {
	c.w.Println("~ Добро пожаловать в консольное приложение Мафия! ~")
	c.w.Print("Введите своё имя:\n> ")
	fmt.Scan(&c.player.Name)
	c.w.Printf("Здравствуй, %s!\n\n", c.player.Name)
}

func (c *LobbyClient) Run() {
	terminal.ClearScreen()
	c.Greet()
	c.ConnectToLobby()
	for {
		var cmd string
		c.w.Print("> ")
		fmt.Scan(&cmd)
	}
}
