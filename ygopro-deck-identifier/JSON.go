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
	for _,tag := range deckType.ForceTags {
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
