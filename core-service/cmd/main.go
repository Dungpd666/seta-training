package main

import (
	"os"

	"github.com/dungpd/seta/core-service/internal/initialize"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	_ = godotenv.Load()
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	if err := initialize.Run(); err != nil {
		log.Fatal().Err(err).Send()
	}
}
