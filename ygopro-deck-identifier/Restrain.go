package ygopro_deck_identifier

import (
	"github.com/iamipanda/ygopro-data"
)

type Restrain interface {
	Judge(deck *ygopro_data.Deck) bool
	Type() string
	ToJson() map[string]interface{}
}

// ======================
// Tool Function
// ======================
/*
func GetDeckTargetRange(deck *ygopro_data.Deck, targetRange string) *[]int {
	switch targetRange {
	case "main":
		return &deck.Main
	case "side":
		return &deck.Side
	case "ex", "extra":
		return &deck.Ex
	case "ori", "origin":
		return &deck.Origin
	case "", "cards":
		return &deck.Cards
	default:
		return &deck.Cards
	}
}*/

func GetDeckTargetClassifiedRange(deck *ygopro_data.Deck, targetRange string) *map[int]int {
	switch targetRange {
	case "main":
		return &deck.ClassifiedMain
	case "side":
		return &deck.ClassifiedSide
	case "ex", "extra":
		return &deck.ClassifiedEx
	case "ori", "origin":
		return &deck.ClassifiedOrigin
	case "", "cards":
		return &deck.ClassifiedCards
	default:
		return &deck.ClassifiedCards
	}
}

// ======================
// Restrain On Cards
// ======================
type CardRestrain struct {
	Id int
	Range string
	Condition Condition
}

func (CardRestrain) Type() string {
	return "Card"
}

func (restrain CardRestrain) Judge(deck *ygopro_data.Deck) bool {
	target := GetDeckTargetClassifiedRange(deck, restrain.Range)
	value := (*target)[restrain.Id]
	return restrain.Condition.Judge(value)
}

// ======================
// Restrain On Sets
// ======================
type SetRestrain struct {
	Set ygopro_data.Set
	Range string
	Condition Condition
}

func (SetRestrain) Type() string {
	return "Set"
}

func (restrain SetRestrain) Judge(deck *ygopro_data.Deck) bool {
	target := GetDeckTargetClassifiedRange(deck, restrain.Range)
	count := 0
	for _, id := range restrain.Set.Ids {
		if value, ok := (*target)[id]; ok {
			count += value
		}
	}
	return restrain.Condition.Judge(count)
}

// ======================
// Combined Restrains
// ======================
type RestrainGroup struct {
	Restrains []Restrain
	Condition Condition
}

func (RestrainGroup) Type() string {
	return "Group"
}

func (restrain RestrainGroup) Judge(deck *ygopro_data.Deck) bool {
	count := 0
	for _, restrain := range restrain.Restrains {
		if restrain.Judge(deck) {
			count += 1
		}
	}
	return restrain.Condition.Judge(count)
}
