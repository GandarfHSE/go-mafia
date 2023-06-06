package main

import "github.com/GandarfHSE/go-mafia/internal/app/client"

func main() {
	cli := client.CreateLobbyClient()
	defer cli.Close()
	cli.Run()
}
