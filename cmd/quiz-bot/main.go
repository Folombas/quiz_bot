package main

import (
	"flag"
	"log"
	"os"

	"quiz_bot/internal/bot"
	"quiz_bot/internal/config"
)

var (
	configPath = flag.String("config", "configs/config.dev.yaml", "Path to configuration file")
	version    = flag.Bool("version", false, "Print version and exit")
)

func main() {
	flag.Parse()

	if *version {
		println(bot.Version)
		os.Exit(0)
	}

	// Загружаем конфигурацию
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Printf("Failed to load config: %v, using defaults", err)
		cfg = config.DefaultConfig()
	}

	// Создаём и запускаем бота
	b, err := bot.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	if err := b.Run(); err != nil {
		log.Fatalf("Bot error: %v", err)
	}
}
