package role

import "github.com/fatih/color"

type Roleless struct {
}

func (r Roleless) Name() string {
	return "roleless"
}

func (r Roleless) GetColor() *color.Color {
	return color.New(color.FgWhite)
}
