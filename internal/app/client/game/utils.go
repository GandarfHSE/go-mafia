package client

import (
	"log"

	"github.com/fatih/color"
)

func PrintRole(role string) {
	var wr *color.Color
	switch role {
	case "civ":
		wr = color.New(color.FgHiGreen, color.Bold)
		wr.Printf("Ваша роль: Мирный житель!\n")
	case "maf":
		wr = color.New(color.FgHiBlack, color.BgHiRed, color.Bold)
		wr.Printf("Ваша роль: Мафия!\n")
	case "com":
		wr = color.New(color.FgHiCyan, color.Bold)
		wr.Printf("Ваша роль: Комиссар!\n")
	default:
		log.Printf("Unknown role: %v", role)
	}
}
