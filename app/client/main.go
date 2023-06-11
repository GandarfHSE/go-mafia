package main

import (
	client "github.com/GandarfHSE/go-mafia/internal/app/client/lobby"
)

func main() {
	defer func() {
		if recover() != nil {
			// всё хорошо =)
		}
	}()

	cli := client.CreateLobbyClient()
	defer cli.Close()
	cli.Run()
}
