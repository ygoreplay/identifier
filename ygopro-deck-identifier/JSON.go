package ygopro_deck_identifier

import (
	"github.com/iamipanda/ygopro-data"
)

func (deckType *Deck) ToJson() (json map[string]interface{}) {
	json = make(map[string]interface{})
	json["name"] = deckType.Name
	json["priority"] = deckType.Priority

	checkTags := make([]interface{}, 0)
	forceTags := make([]interface{}, 0)
	refuseTags := make([]interface{}, 0)
	restrains := make([]interface{}, 0)

	for _, tag := range deckType.CheckTags {
		checkTags = append(checkTags, tag.ToJson())
	}
	for _, tag := range deckType.ForceTags {
		forceTags = append(forceTags, tag.ToJson())
	}
	for _, tag := range deckType.RefuseTags {
		refuseTags = append(refuseTags, tag.ToJson())
	}
	for _, restrain := range deckType.Restrains {
		restrains = append(restrains, restrain.ToJson())
	}

	json["checkTags"] = checkTags
	json["forceTags"] = forceTags
	json["refuseTags"] = refuseTags
	json["restrains"] = restrains
	return json
}

func (tag *Tag) ToJson() (json map[string]interface{}) {
	json = make(map[string]interface{})
	json["name"] = tag.Name
	json["priority"] = tag.Priority

	restrains := make([]interface{}, 0)
	configs := make([]interface{}, 0)

	for _, restrain := range tag.Restrains {
		restrains = append(restrains, restrain.ToJson())
	}
	for _, config := range tag.Configs {
		configs = append(configs, config)
	}

	json["restrains"] = restrains
	json["configs"] = configs
	return json
}

func (restrain CardRestrain) ToJson() (json map[string]interface{}) {
	json = make(map[string]interface{})
	json["type"] = restrain.Type()
	json["id"] = restrain.Id
	if card, ok := ygopro_data.GetEnvironment("zh-CN").GetCard(restrain.Id); ok {
		json["name"] = card.Name
	}
	json["range"] = restrain.Range
	json["condition"] = restrain.Condition.ToJson()
	return json
}

func (restrain SetRestrain) ToJson() (json map[string]interface{}) {
	json = make(map[string]interface{})
	json["type"] = restrain.Type()
	json["set"] = SetToJson(restrain.Set)
	json["condition"] = restrain.Condition.ToJson()
	return json
}

func (restrain RestrainGroup) ToJson() (json map[string]interface{}) {
	json = make(map[string]interface{})
	restrains := make([]interface{}, 0)
	for _, restrain := range restrain.Restrains {
		restrains = append(restrains, restrain.ToJson())
	}
	json["type"] = restrain.Type()
	json["restrains"] = restrains
	json["condition"] = restrain.Condition.ToJson()
	return json
}

func SetToJson(set ygopro_data.Set) (json map[string]interface{}) {
	json = make(map[string]interface{})
	json["name"] = set.Name
	json["originName"] = set.OriginName
	json["ids"] = set.Ids
	return json
}

func (condition *Condition) ToJson() (json map[string]interface{}) {
	json = make(map[string]interface{})
	json["operator"] = condition.operator
	json["number"] = condition.number
	return json
}

func (identifier *IdentifierWrapper) ToJson() (json map[string]interface{}) {
	json = make(map[string]interface{})
	decks := make([]map[string]interface{}, 0)
	tags := make([]map[string]interface{}, 0)
	sets := make([]map[string]interface{}, 0)
	for _, deck := range identifier.Decks {
		decks = append(decks, deck.ToJson())
	}
	for _, tag := range identifier.Tags {
		tags = append(tags, tag.ToJson())
	}
	for _, set := range identifier.CustomSets {
		sets = append(sets, SetToJson(set))
	}
	json["decks"] = decks
	json["tags"] = tags
	json["sets"] = sets
	return json
}

// Result#ToJson will remove the deck/tag details, only return the name.
func (result *Result) ToJson() map[string]interface{} {
	json := make(map[string]interface{})
	if result == nil {
		json["deck"] = Config.UnknownDeck
	} else if len(result.Deck.Name) == 0 {
		json["deck"] = Config.UnknownDeck
	} else {
		json["deck"] = result.Deck.Name

		tags := make([]string, 0)
		for _, tag := range result.Tags {
			tags = append(tags, tag.Name)
		}
		json["tag"] = tags

		deckTags := make([]string, 0)
		for _, tag := range result.DeckTags {
			deckTags = append(deckTags, tag.Name)
		}
		json["deckTag"] = deckTags
	}
	return json
}

// =========================
// Verbose Area
// =========================

// VerboseDeckAnswer#ToJson will remove the deck details, only return the name.
func (answer *VerboseDeckAnswer) ToJson() map[string]interface{} {
	json := make(map[string]interface{})
	json["deck"] = answer.deck.Name
	json["is"] = answer.is
	children := make([]interface{}, 0)
	for _, child := range answer.children {
		children = append(children, child.ToJson())
	}
	json["children"] = children
	return json
}

// VerboseTagAnswer#ToJson will remove the tag details, only return the name.
func (answer *VerboseTagAnswer) ToJson() map[string]interface{} {
	json := make(map[string]interface{})
	json["tag"] = answer.tag.Name
	json["is"] = answer.is
	children := make([]interface{}, 0)
	for _, child := range answer.children {
		children = append(children, child.ToJson())
	}
	json["children"] = children
	return json
}

func (answer *VerboseRestrainAnswer) ToJson() map[string]interface{} {
	json := answer.restrain.ToJson()
	json["value"] = answer.value
	json["is"] = answer.is
	children := make([]interface{}, 0)
	for _, child := range answer.children {
		children = append(children, child.ToJson())
	}
	json["children"] = children
	return json
}

func (result *VerboseResult) ToJson() map[string]interface{} {
	json := result.Result.ToJson()

	verboseDecks := make([]interface{}, 0)
	verboseCheckTags := make([]interface{}, 0)
	verboseGlobalTags := make([]interface{}, 0)
	for _, answer := range result.verboseDecks {
		verboseDecks = append(verboseDecks, answer.ToJson())
	}
	for _, answer := range result.verboseCheckTags {
		verboseCheckTags = append(verboseCheckTags, answer.ToJson())
	}
	for _, answer := range result.verboseGlobalTags {
		verboseGlobalTags = append(verboseGlobalTags, answer.ToJson())
	}
	json["verboseDecks"] = verboseDecks
	json["verboseCheckTags"] = verboseCheckTags
	json["verboseGlobalTags"] = verboseGlobalTags
	json["polymerizedTags"] = result.polymerizedTags

	forcedTags := make([]string, 0)
	removedTags := make([]string, 0)
	for _, tag := range result.forcedTags {
		forcedTags = append(forcedTags, tag.Name)
	}
	for _, tag := range result.removedTags {
		removedTags = append(removedTags, tag.Name)
	}
	json["forcedTags"] = forcedTags
	json["removedTags"] = removedTags

	return json
}
