package ygopro_deck_identifier

type Result struct {
	Deck Deck
	Tags []Tag
}

func (result Result) getName() (name string) {
	name = result.Deck.Name
	newTags := make([]Tag, len(result.Tags));
	for _, tag := range result.Tags {
		if tag.Is("prefix") {
			name = tag.Name + name
		} else if tag.Is("appendix") {
			name = name + tag.Name
		} else {
			newTags = append(newTags, tag)
		}
	}
	result.Tags = newTags
	return name
}