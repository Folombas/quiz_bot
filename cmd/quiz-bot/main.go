package main

import (
	"log"

	"quiz_bot/internal/bot"
)

func main() {
	cfg := bot.DefaultConfig()
	b := bot.New(cfg)

	if err := b.Run(); err != nil {
		log.Fatal(err)
	}
}
