package main

import (
	"fmt"
	"os"

	"github.com/mceck/clickup-tui/internal/app"
)

func main() {
	p := app.NewProgram()
	if _, err := p.Run(); err != nil {
		fmt.Printf("Errore durante l'avvio dell'applicazione: %v\n", err)
		os.Exit(1)
	}
}
