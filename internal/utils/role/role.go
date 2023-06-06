package role

import (
	"log"

	"github.com/fatih/color"
)

type Role interface {
	Name() string
	GetColor() *color.Color
}

func CreateRole(name string) Role {
	switch name {
	case "roleless":
		return Roleless{}
	default:
		log.Printf("Can't create role with name %v!\n", name)
		return nil
	}
}
