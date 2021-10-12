package main

import (
	ygopro_data "github.com/iamipanda/ygopro-data"
	ygopro_deck_identifier "identifier/ygopro-deck-identifier"
	"os"
	"path/filepath"
)

func main() {
	ygopro_data.LuaPath = filepath.Join(os.Getenv("GOPATH"), "pkg/mod/github.com/iamipanda/ygopro-data@v0.0.0-20190116110429-360968dc5c66/Constant.lua")

	ygopro_deck_identifier.Initialize()
	ygopro_deck_identifier.StartServer()
}
