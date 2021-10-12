package ygopro_deck_identifier

import (
	"encoding/json"
	"os"
)

type Configuration struct {
	DatabasePath    string
	DeckDefPath     string
	UnknownDeck     string
	IdentifierNames []string
	Listening       string
	AccessKey       string
}

var Config Configuration

func InitializeConfig() {
	file, err := os.Open("./ygopro-deck-identifier/Config.json")
	if err != nil {
		Logger.Errorf("Failed to open Config.json. %v", err)
	}
	decoder := json.NewDecoder(file)
	Config = Configuration{}
	err = decoder.Decode(&Config)
	if err != nil {
		Logger.Errorf("Failed to load config: %v", err)
	}
}
