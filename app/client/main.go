package main

import client "github.com/GandarfHSE/go-mafia/internal/app/client/lobby"

func main() {
	cli := client.CreateLobbyClient()
	defer cli.Close()
	cli.Run()
}
