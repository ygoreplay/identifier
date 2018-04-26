package ygopro_deck_identifier

import (
	"github.com/iamipanda/ygopro-data"
	"sort"
	"strconv"
	"path"
)

type astIdentifier struct {
	decks []*astNode
	tags  []*astNode
	sets  []*astNode

	tagNameHash map[string]Tag
}

func (identifier *astIdentifier) registerNode(node *astNode) {
	switch node.Type {
	case "root":
		for _, childNode := range node.Children {
			identifier.registerNode(childNode)
		}
	case "deck":
		identifier.decks = append(identifier.decks, node)
	case "tag":
		identifier.tags = append(identifier.tags, node)
	case "set":
		identifier.sets = append(identifier.sets, node)
	default:
		Logger.Warning(originMessageLoggerHead(node) + " Unknown child node under Root node when register: " + node.Type)
	}
}

func (identifier *astIdentifier) clear() {
	identifier.decks = identifier.decks[:0]
	identifier.tags = identifier.tags[:0]
	identifier.sets = identifier.sets[:0]
	identifier.tagNameHash = make(map[string]Tag)
}

func (identifier *astIdentifier) prepare(target *Identifier, backup *Identifier) *Identifier {
	// 递归解 Sets 序列
	identifier.prepareSets(target, backup)
	// 转换 Tags 序列
	identifier.prepareTags(target, backup)
	// 转换 Decks 序列（包含检索空 Tag 序列）
	identifier.prepareDecks(target, backup)
	// 排序
	sort.Reverse(DeckSort(target.Decks))
	return target
}

func (identifier *astIdentifier) prepareSets(target *Identifier, backup *Identifier) {
	setNames := make(map[string]*astNode)
	sets := make(map[string]ygopro_data.Set)
	for _, set := range target.BindingEnvironment.Sets {
		sets[set.Name] = set
	}
	if backup != nil {
		for _, set := range backup.CustomSets {
			sets[set.Name] = set
		}
	}
	for _, setNode := range identifier.sets {
		setNames[setNode.Value] = setNode
	}
	for _, setNode := range identifier.sets {
		target.CustomSets = append(target.CustomSets, transformSet(setNode, &setNames, &sets, target))
	}
	target.generateSetHash()
}

func (identifier *astIdentifier) prepareTags(target *Identifier, backup *Identifier) {
	for _, tag := range identifier.tags {
		target.Tags = append(target.Tags, identifier.transformTag(tag, target, backup, false))
	}
}

func (identifier *astIdentifier) prepareDecks(target *Identifier, backup *Identifier) {
	for _, deck := range identifier.decks {
		target.Decks = append(target.Decks, identifier.transformDeck(deck, target, backup))
	}
}

func transformSet(node *astNode, nonConvertedSets *map[string]*astNode, convertedSets *map[string]ygopro_data.Set, target *Identifier) ygopro_data.Set {
	set := ygopro_data.Set{}
	set.Locale = target.BindingEnvironment.Locale
	set.Code = 0
	set.OriginName = ""
	set.Name = node.Value
	for _, childNode := range node.Children {
		switch childNode.Type {
		case "inner set":
			if innerSet, ok := (*convertedSets)[childNode.Value]; ok {
				for _, id := range innerSet.Ids {
					set.Ids = append(set.Ids, id)
				}
			} else if innerNode, ok := (*nonConvertedSets)[childNode.Value]; ok {
				innerSet := transformSet(innerNode, nonConvertedSets, convertedSets, target)
				for _, id := range innerSet.Ids {
					set.Ids = append(set.Ids, id)
				}
			} else if innerSet := target.BindingEnvironment.GetAllNamedCard(childNode.Value); len(innerSet.Ids) > 0 {
				(*convertedSets)[childNode.Value] = innerSet
				for _, id := range innerSet.Ids {
					set.Ids = append(set.Ids, id)
				}
			} else {
				Logger.Warning(originMessageLoggerHead(childNode) + " Unknown child node under Set node: " + childNode.Value)
			}
		case "set card":
			if card, ok := transformCard(childNode.Value, target.BindingEnvironment); ok {
				set.Ids = append(set.Ids, card.Id)
			} else {
				Logger.Warning(originMessageLoggerHead(childNode) + " Can't find card named: " + childNode.Value)
			}
		default:
			Logger.Warning(originMessageLoggerHead(childNode) + " Unknown child node under Set node: " + childNode.Type)
		}
	}
	if len(set.Ids) == 0 {
		Logger.Warningf(originMessageLoggerHead(node) + " Created an empty user defined Set named %v", set.Name)
	} else {
		Logger.Infof(originMessageLoggerHead(node) + " Created user defined Set named %v with %d Cards.", set.Name, len(set.Ids))
	}
	if _, ok := (*convertedSets)[node.Value]; ok {
		Logger.Warningf("Rewriting existing set %v.", node.Value)
	}
	(*convertedSets)[node.Value] = set
	set.Sort()
	return set
}

func transformRestrain(node *astNode, target *Identifier, backup *Identifier) Restrain {
	switch node.Value {
	case "card":
		restrain := CardRestrain{}
		for _, childNode := range node.Children {
			switch childNode.Type {
			case "condition":
				restrain.Condition, _ = CreateConditionFromString(childNode.Value)
			case "range":
				restrain.Range = childNode.Value
			case "target":
				if card, ok := transformCard(childNode.Value, target.BindingEnvironment); ok {
					restrain.Id = card.Id
				} else {
					Logger.Warning(originMessageLoggerHead(node) + " Can't find card named: " + childNode.Value)
				}
			default:
				Logger.Warning(originMessageLoggerHead(node) + " Unknown child node under card Restrain: " + childNode.Type)
			}
		}
		return restrain
	case "set":
		restrain := SetRestrain{}
		for _, childNode := range node.Children {
			switch childNode.Type {
			case "condition":
				restrain.Condition, _ = CreateConditionFromString(childNode.Value)
			case "range":
				restrain.Range = childNode.Value
			case "target":
				if set, ok := target.searchNamedSet(childNode.Value); ok {
					restrain.Set = set
				} else if backup != nil {
					if set, ok := backup.searchNamedSet(childNode.Value); ok {
						restrain.Set = set
					} else {
						Logger.Warning(originMessageLoggerHead(node) + " Can't find set named " + childNode.Value)
					}
				} else {
					Logger.Warning(originMessageLoggerHead(node) + " Can't find set named " + childNode.Value)
				}
			default:
				Logger.Warning(originMessageLoggerHead(node) + " Unknown child node under set Restrain: " + childNode.Type)
			}
		}
		return restrain
	case "and", "or":
		restrain := RestrainGroup{}
		for _, childNode := range node.Children {
			if childNode.Type == "restrain" {
				restrain.Restrains = append(restrain.Restrains, transformRestrain(childNode, target, backup))
			} else {
				Logger.Warning(originMessageLoggerHead(childNode) + " non-restrain child node under group Restrain: " + childNode.Type)
			}
		}
		if node.Value == "and" {
			restrain.Condition = NewCondition("and", len(restrain.Restrains))
		} else if node.Value == "or" {
			restrain.Condition = NewCondition("or", 1)
		}
		return restrain
	default:
		if match := conditionStringReg.FindString(node.Value); len(match) > 0 {
			restrain := RestrainGroup{}
			restrain.Condition, _ = CreateConditionFromString(node.Value)
			for _, childNode := range node.Children {
				if childNode.Type == "restrain" {
					restrain.Restrains = append(restrain.Restrains, transformRestrain(childNode, target, backup))
				} else {
					Logger.Warning(originMessageLoggerHead(node) + " non-restrain child node under group Restrain: " + childNode.Type)
				}
			}
			return restrain
		} else {
			Logger.Warning(originMessageLoggerHead(node) + " Unknown restrain type: " + node.Value)
		}
	}
	return CardRestrain{}
}

func (identifier *astIdentifier) transformTag(node *astNode, target *Identifier, backup *Identifier, checkEmpty bool) Tag {
	tag := Tag{}
	tag.Name = node.Value
	for _, childNode := range node.Children {
		switch childNode.Type {
		case "restrain":
			tag.Restrains = append(tag.Restrains, transformRestrain(childNode, target, backup))
		case "config":
			tag.Configs = append(tag.Configs, childNode.Value)
		case "priority":
			tag.Priority, _ = strconv.Atoi(childNode.Value)
		default:
			Logger.Warning(originMessageLoggerHead(childNode) + " Unknown child node under Tag node: " + childNode.Type)
		}
	}
	if checkEmpty && len(node.Children) == 0 {
		if namedTag, ok := identifier.tagNameHash[tag.Name]; ok {
			return namedTag
		} else if backup != nil {
			if namedTag, ok := backup.prototype.tagNameHash[tag.Name]; ok {
				return namedTag
			}
		}
		Logger.Warning(originMessageLoggerHead(node) + " Empty Tag: " + tag.Name)
	}
	if tag.Is("global") {
		target.GlobalTags = append(target.GlobalTags, tag)
	}
	return tag
}

func (identifier *astIdentifier) transformDeck(node *astNode, target *Identifier, backup *Identifier) Deck {
	deck := Deck{}
	deck.Name = node.Value
	for _, childNode := range node.Children {
		switch childNode.Type {
		case "restrain":
			deck.Restrains = append(deck.Restrains, transformRestrain(childNode, target, backup))
		case "check tag":
			deck.CheckTags = append(deck.CheckTags, identifier.transformTag(childNode, target, backup, true))
		case "force tag":
			deck.ForceTags = append(deck.ForceTags, identifier.transformTag(childNode, target, backup, true))
		case "refuse tag":
			deck.RefuseTags = append(deck.RefuseTags, identifier.transformTag(childNode, target, backup, true))
		case "priority":
			deck.Priority, _ = strconv.Atoi(childNode.Value)
		default:
			Logger.Warning(originMessageLoggerHead(node) + "Unknown child node under Deck node: " + childNode.Type)
		}
	}
	if len(deck.Restrains) == 0 {
		Logger.Warningf("%v No restrains registered to deck %v, there won't be deck named that.", originMessageLoggerHead(node), deck.Name)
	}
	return deck
}

func transformCard(value string, environment *ygopro_data.Environment) (ygopro_data.Card, bool) {
	id, err := strconv.Atoi(value)
	if err == nil {
		return environment.GetCard(id)
	} else {
		return environment.GetNamedCardCached(value)
	}
	return ygopro_data.Card{}, false
}

func originMessageLoggerHead(node *astNode) string {
	return "[" + path.Base(node.Origin.File) + "] L" + strconv.Itoa(node.Origin.Line)
}