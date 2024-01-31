package main

import (
	"fmt"
	"github.com/fatih/color"
)

func printColorMessage(symbol, message, colorCode string) {
	coloredMessage := color.New(color.FgHiGreen).Sprintf("[%s] %s", symbol, message)
	fmt.Println(coloredMessage)
}
