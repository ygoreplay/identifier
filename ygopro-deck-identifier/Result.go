package ygopro_deck_identifier

type Result struct {
	Deck Deck
	Tags []Tag
}

func (result *Result) processAffixAndGetName(save bool) (name string) {
	name = result.Deck.Name
	newTags := make([]Tag, len(result.Tags))
	for _, tag := range result.Tags {
		if tag.Is("prefix") {
			name = tag.Name + name
		} else if tag.Is("appendix") {
			name = name + tag.Name
		} else {
			newTags = append(newTags, tag)
		}
	}
	if save {
		result.Deck.Name = name
	}
	result.Tags = newTags
	return name
}
