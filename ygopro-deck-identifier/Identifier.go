package ygopro_deck_identifier

import (
	"github.com/iamipanda/ygopro-data"
	"path/filepath"
	"os"
	"strings"
	"github.com/op/go-logging"
	"sort"
	"bytes"
)

type Identifier struct {
	Name string
	Decks []Deck
	Tags []Tag
	GlobalTags []Tag
	CustomSets []ygopro_data.Set

	prototype *astIdentifier
	BindingEnvironment *ygopro_data.Environment
	SetNameHash map[string]ygopro_data.Set
}

func NewIdentifier(name string) *Identifier {
	identifier := new(Identifier)
	identifier.Name = name
	identifier.prototype = new(astIdentifier)
	identifier.BindingEnvironment = ygopro_data.GetEnvironment("zh-CN")
	return identifier
}

func (identifier *Identifier) RegisterFolder(dirName string) {
	filepath.Walk(dirName, func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(path, ".deckdef") {
			identifier.RegisterDSLFile(path)
		}
		return nil
	})
}

func (identifier *Identifier) RegisterDSLFile(filename string) {
	compiler := new(Compiler)
	compiler.CompileFile(filename)
	identifier.prototype.registerNode(compiler.Root)
}

func (identifier *Identifier) RegisterDSL(string string) {
	compiler := new(Compiler)
	compiler.CompileString(string)
	identifier.prototype.registerNode(compiler.Root)
}

func (identifier *Identifier) Ready(backup *Identifier) {
	identifier.prototype.prepare(identifier, backup)
	Logger.Noticef("Identifier %v is Ready, %d Decks, %d Tags (%d is Global), %d Custom Sets loaded.", identifier.Name, len(identifier.Decks), len(identifier.Tags), len(identifier.GlobalTags), len(identifier.CustomSets))
}

func (identifier *Identifier) clear() {
	identifier.Decks = identifier.Decks[:0]
	identifier.Tags = identifier.Tags[:0]
	identifier.GlobalTags = identifier.GlobalTags[:0]
	identifier.CustomSets = identifier.CustomSets[:0]
	identifier.prototype.clear()
	identifier.SetNameHash = make(map[string]ygopro_data.Set)
}

func (identifier *Identifier) Recognize(deck ygopro_data.Deck) (*Result) {
	result := identifier.recognizeDeck(deck)
	tags := identifier.recognizeTags(deck)
	if result == nil {
		return identifier.polymerize(tags)
	}
	if result == nil {
		return nil
	}
	for _, tag := range tags {
		result.Tags = append(result.Tags, tag)
	}
	return result
}

func (identifier *Identifier) recognizeDeck(deck ygopro_data.Deck) (*Result) {
	for _, deckType := range identifier.Decks {
		if result := deckType.Execute(deck); result != nil {
			return result
		}
	}
	return nil
}

func (identifier *Identifier) recognizeTags(deck ygopro_data.Deck) ([]Tag) {
	answer := make([]Tag, 0)
	for _, tag := range identifier.GlobalTags {
		if tag.Judge(deck) {
			answer = append(answer, tag)
		}
	}
	return answer
}

func (identifier *Identifier) polymerize(tags []Tag) *Result {
	sort.Reverse(TagSort(tags))
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
		return nil
	}
	var buffer bytes.Buffer
	for _, tag := range upgradeTags {
		buffer.WriteString(tag.Name)
	}
	result := new(Result)
	result.Deck = Deck {}
	result.Deck.Name = buffer.String()
	result.Tags = normalTags
	return result
}

func (identifier *Identifier) generateSetHash() {
	identifier.SetNameHash = make(map[string]ygopro_data.Set)
	for _, set := range identifier.BindingEnvironment.Sets {
		identifier.SetNameHash[set.Name] = set
		if len(set.OriginName) > 0 {
			identifier.SetNameHash[set.OriginName] = set
		}
	}
	for _, set := range identifier.CustomSets {
		identifier.SetNameHash[set.Name] = set
	}
}

func (identifier *Identifier) searchNamedSet(name string) (ygopro_data.Set, bool) {
	if len(name) == 0 {
		Logger.Warning("Try to search card set named EMPTY.")
	} else if set, ok := identifier.SetNameHash[name]; ok {
		return set, true
	} else if set := identifier.BindingEnvironment.GetAllNamedCard(name); len(set.Ids) > 0 {
		Logger.Infof("Created searched Set named %v under environment %v with %d cards.", name, identifier.BindingEnvironment.Locale, len(set.Ids))
		identifier.SetNameHash[name] = set
		identifier.CustomSets = append(identifier.CustomSets, set)
		return set, true
	}
	return ygopro_data.Set{}, false
}

// ================ Main Functions =================
var Logger = logging.MustGetLogger("standard")
var NormalLoggingBackend logging.Backend

func Initialize() {
	format := logging.MustStringFormatter(
		`%{color} %{id:05x} %{time:15:04:05.000} ▶ %{level:.4s}%{color:reset} %{message} from [%{shortfunc}] `,
	)
	backendPrototype := logging.NewLogBackend(os.Stderr, "", 0)
	fBackend := logging.NewBackendFormatter(backendPrototype, format)
	lBackend := logging.AddModuleLevel(fBackend)
	lBackend.SetLevel(logging.INFO, "")
	NormalLoggingBackend = lBackend
	logging.SetBackend(NormalLoggingBackend)

	InitializeConfig()
	ygopro_data.DatabasePath = Config.DatabasePath
	ygopro_data.InitializeStaticEnvironment()
	RegisterIdentifiersAccordingToConfig()
}