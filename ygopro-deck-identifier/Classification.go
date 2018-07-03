package ygopro_deck_identifier

import "github.com/iamipanda/ygopro-data"

type Classification struct {
	Name      string
	Priority  int
	Restrains []Restrain
}

func (classification Classification) Judge(deck ygopro_data.Deck) bool {
	if len(classification.Restrains) == 0 {
		return false
	}
	for _, restrain := range classification.Restrains {
		if !restrain.Judge(&deck) {
			return false
		}
	}
	return true
}

type Deck struct {
	Classification
	CheckTags, ForceTags, RefuseTags []Tag
	RefuseHash map[string]bool
}

func (deckType Deck) Execute(deck ygopro_data.Deck) (*Result) {
	if deckType.Judge(deck) {
		result := new(Result)
		result.Deck = deckType
		for _, tag := range deckType.ForceTags {
			result.Tags = append(result.Tags, tag)
		}
		for _, tag := range deckType.CheckTags {
			if tag.Judge(deck) {
				result.Tags = append(result.Tags, tag)
			}
		}
		return result
	} else {
		return nil
	}
}

func (deckType Deck) RemoveRefusedTags(result *Result) []Tag {
	if deckType.RefuseHash == nil {
		deckType.RefuseHash = make(map[string]bool)
		for _, tag := range deckType.RefuseTags {
			deckType.RefuseHash[tag.Name] = true
		}
	}
	newTags := make([]Tag, 0)
	refusedTags := make([]Tag, 0)
	for _, tag := range result.Tags {
		if _, ok := deckType.RefuseHash[tag.Name]; !ok {
			newTags = append(newTags, tag)
		} else {
			refusedTags = append(refusedTags, tag)
		}
	}
	result.Tags = newTags
	return refusedTags
}

type DeckSort []Deck
func (sort DeckSort) Len() int { return len(sort) }
func (sort DeckSort) Less(i, j int) bool { return sort[i].Priority < sort[j].Priority }
func (sort DeckSort) Swap(i, j int) { sort[i], sort[j] = sort[j], sort[i] }

type Tag struct {
	Classification
	Configs []string
	ConfigCache map[string]bool
}

func (tag Tag) Is(config string) (bool) {
	if tag.ConfigCache == nil {
		tag.ConfigCache = make(map[string]bool)
	}
	value, ok := tag.ConfigCache[config]
	if ok {
		return value
	}
	value = false
	for _, tagConfig := range tag.Configs {
		if tagConfig == config {
			value = true
			break
		}
	}
	tag.ConfigCache[config] = value
	return value
}

type TagSort []Tag
func (sort TagSort) Len() int { return len(sort) }
func (sort TagSort) Less(i, j int) bool { return sort[i].Priority < sort[j].Priority }
func (sort TagSort) Swap(i, j int) { sort[i], sort[j] = sort[j], sort[i] }