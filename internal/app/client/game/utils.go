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
		wr.Print("Мирный житель")
	case "maf":
		wr = color.New(color.FgHiBlack, color.BgHiRed, color.Bold)
		wr.Print("Мафия")
	case "com":
		wr = color.New(color.FgHiCyan, color.Bold)
		wr.Print("Комиссар")
	default:
		log.Printf("Unknown role: %v", role)
	}
}
