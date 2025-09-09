package config

import (
	"PoolManagerVM/backend/models"
	"log"

	"github.com/BurntSushi/toml"
)

func LoadConfig(path string) models.Config {
	var conf models.Config
	if _, err := toml.DecodeFile(path, &conf); err != nil {
		log.Fatalf("Error loading config.toml: %v", err)
	}

	return conf
}
