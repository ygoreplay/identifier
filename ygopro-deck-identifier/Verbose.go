package ygopro_deck_identifier

import (
	"bytes"
	"github.com/iamipanda/ygopro-data"
	"sort"
)

type VerboseRestrain interface {
	verboseJudge(deck *ygopro_data.Deck) VerboseRestrainAnswer
}

type VerboseRestrainAnswer struct {
	restrain Restrain
	value    int
	is       bool
	children []VerboseRestrainAnswer
}

type VerboseDeckAnswer struct {
	deck     Deck
	is       bool
	children []VerboseRestrainAnswer
}

type VerboseTagAnswer struct {
	tag      Tag
	is       bool
	children []VerboseRestrainAnswer
}

type VerboseResult struct {
	*Result
	verboseDecks      []VerboseDeckAnswer
	verboseCheckTags  []VerboseTagAnswer
	verboseGlobalTags []VerboseTagAnswer
	forcedTags        []Tag
	removedTags       []Tag
	polymerizedTags   []string
}

func (restrain CardRestrain) verboseJudge(deck *ygopro_data.Deck) VerboseRestrainAnswer {
	target := GetDeckTargetClassifiedRange(deck, restrain.Range)
	value := (*target)[restrain.Id]
	return VerboseRestrainAnswer{restrain, value, restrain.Condition.Judge(value), nil}
}

func (restrain SetRestrain) verboseJudge(deck *ygopro_data.Deck) VerboseRestrainAnswer {
	target := GetDeckTargetClassifiedRange(deck, restrain.Range)
	count := 0
	for _, id := range restrain.Set.Ids {
		if value, ok := (*target)[id]; ok {
			count += value
		}
	}
	if restrain.Set.Name == "影依" {
		Logger.Debugf("[%s] = %d", restrain.Range, len(*target))
		Logger.Debugf("Count = %d", count)
	}
	return VerboseRestrainAnswer{restrain, count, restrain.Condition.Judge(count), nil}
}

func (restrain RestrainGroup) verboseJudge(deck *ygopro_data.Deck) VerboseRestrainAnswer {
	count := 0
	children := make([]VerboseRestrainAnswer, 0)
	for _, restrain := range restrain.Restrains {
		if verboseRestrain, ok := restrain.(VerboseRestrain); ok {
			child := verboseRestrain.verboseJudge(deck)
			children = append(children, child)
			if child.is {
				count += 1
			}
		} else {
			Logger.Warningf("No verbose judge defined for a restrain.")
			pass := restrain.Judge(deck)
			children = append(children, VerboseRestrainAnswer{restrain, -1, pass, nil})
			if pass {
				count += 1
			}
		}
	}
	return VerboseRestrainAnswer{restrain, count, restrain.Condition.Judge(count), children}
}

func (classification *Classification) verboseJudge(deck *ygopro_data.Deck) (bool, []VerboseRestrainAnswer) {
	children := make([]VerboseRestrainAnswer, 0)
	for _, restrain := range classification.Restrains {
		if verboseRestrain, ok := restrain.(VerboseRestrain); ok {
			children = append(children, verboseRestrain.verboseJudge(deck))
		} else {
			Logger.Warningf("No verbose judge defined for a restrain.")
			children = append(children, VerboseRestrainAnswer{restrain, -1, restrain.Judge(deck), nil})
		}
	}
	is := true
	for _, child := range children {
		if !child.is {
			is = false
			break
		}
	}
	return is, children
}

func (deckType Deck) verboseJudge(deck *ygopro_data.Deck) VerboseDeckAnswer {
	is, children := deckType.Classification.verboseJudge(deck)
	return VerboseDeckAnswer{deckType, is, children}
}

func (tag Tag) verboseJudge(deck *ygopro_data.Deck) VerboseTagAnswer {
	is, children := tag.Classification.verboseJudge(deck)
	return VerboseTagAnswer{tag, is, children}
}

func (identifier *Identifier) verboseRecognize(deck ygopro_data.Deck) *VerboseResult {
	result := identifier.verboseRecognizeDeck(&deck)
	tags, answers := identifier.verboseRecognizeTags(&deck)
	result.verboseGlobalTags = answers
	if result.Result == nil {
		return identifier.verbosePolymerizeTags(result, tags)
	}
	for _, tag := range tags {
		result.Tags = append(result.Tags, tag)
	}
	result.removedTags = result.Deck.RemoveRefusedTags(result.Result)
	return result
}

func (identifier *Identifier) verboseRecognizeDeck(deck *ygopro_data.Deck) *VerboseResult {
	answers := make([]VerboseDeckAnswer, 0)
	var correctDeckType *Deck = nil
	for _, deckType := range identifier.Decks {
		answer := deckType.verboseJudge(deck)
		answers = append(answers, answer)
		if answer.is && correctDeckType == nil {
			// Fuck Golang.
			tempDeck := deckType
			correctDeckType = &tempDeck
		}
	}
	verboseCheckTags := make([]VerboseTagAnswer, 0)
	var result *Result = nil
	var forceTags []Tag = nil
	if correctDeckType != nil {
		result = &Result{*correctDeckType, make([]Tag, 0), make([]Tag, 0)}
		for _, tag := range correctDeckType.CheckTags {
			answer := tag.verboseJudge(deck)
			verboseCheckTags = append(verboseCheckTags, answer)
			if answer.is {
				result.Tags = append(result.Tags, tag)
			}
		}
		for _, tag := range correctDeckType.ForceTags {
			result.Tags = append(result.Tags, tag)
		}
		forceTags = correctDeckType.ForceTags
	}
	return &VerboseResult{result, answers, verboseCheckTags, nil, forceTags, nil, nil}
}

func (identifier *Identifier) verboseRecognizeTags(deck *ygopro_data.Deck) ([]Tag, []VerboseTagAnswer) {
	tags := make([]Tag, 0)
	answers := make([]VerboseTagAnswer, 0)
	for _, tag := range identifier.GlobalTags {
		answer := tag.verboseJudge(deck)
		if answer.is {
			tags = append(tags, tag)
		}
		answers = append(answers, answer)
	}
	return tags, answers
}

func (identifier *Identifier) verbosePolymerizeTags(answer *VerboseResult, tags []Tag) *VerboseResult {
	sort.Sort(sort.Reverse(TagSort(tags)))
	upgradeTags := make([]Tag, 0)
	normalTags := make([]Tag, 0)
	for _, tag := range tags {
		if tag.Is("upgrade") {
			upgradeTags = append(upgradeTags, tag)
		} else {
			normalTags = append(normalTags, tag)
		}
	}
	if len(upgradeTags) == 0 {
		return answer
	}
	answer.polymerizedTags = make([]string, 0)
	var buffer bytes.Buffer
	for _, tag := range upgradeTags {
		buffer.WriteString(tag.Name)
		answer.polymerizedTags = append(answer.polymerizedTags, tag.Name)
	}
	result := new(Result)
	result.Deck = Deck{}
	result.Deck.Name = buffer.String()
	result.Tags = normalTags
	answer.Result = result
	return answer
}
