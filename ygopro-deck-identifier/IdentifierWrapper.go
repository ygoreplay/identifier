package ygopro_deck_identifier

import (
	"github.com/iamipanda/ygopro-data"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"bytes"
	"github.com/op/go-logging"
	"os/exec"
)

type IdentifierWrapper struct {
	Identifier
}

var GlobalIdentifierMap map[string]*IdentifierWrapper = make(map[string]*IdentifierWrapper)

func GetWrappedIdentifier(name string) *IdentifierWrapper {
	if wrapper, ok := GlobalIdentifierMap[name]; ok {
		return wrapper
	} else {
		identifier := new(IdentifierWrapper)
		identifier.Name = name
		identifier.prototype = new(astIdentifier)
		identifier.BindingEnvironment = ygopro_data.GetEnvironment("zh-CN")
		GlobalIdentifierMap[name] = identifier
		return identifier
	}
}

func RegisterIdentifiersAccordingToConfig() {
	for _, name := range Config.IdentifierNames {
		identifier := GetWrappedIdentifier(name)
		identifier.Reload()
	}
}

func (identifier *Identifier) RecognizeAsJson(deck ygopro_data.Deck) (json map[string]interface{}) {
	result := identifier.Recognize(deck)
	json = make(map[string]interface{})
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
	}
	return json
}

func (identifier *IdentifierWrapper) GetPath() string {
	return path.Join(Config.DeckDefPath, identifier.Name)
}

func (identifier *IdentifierWrapper) CheckPathExist() bool {
	path := identifier.GetPath()
	fileInfo, err := os.Stat(path)
	if err == nil {
		if fileInfo.Mode().IsDir() {
			return true
		} else {
			Logger.Errorf("Deck definition path %v is not a directory!", path)
			return false
		}
	}
	if err = os.Mkdir(path, 0777); err != nil {
		Logger.Errorf("Failed to create deck definition directory: %v", path)
		return false
	} else {
		return true
	}
}

func (identifier *IdentifierWrapper) GetFileList() []string {
	if !identifier.CheckPathExist() {
		return nil
	} else {
		fileList := make([]string, 0)
		filepath.Walk(identifier.GetPath(), func(path string, info os.FileInfo, err error) error {
			if strings.HasSuffix(path, ".deckdef") {
				fileList = append(fileList, filepath.Base(path))
			}
			return nil
		})
		return fileList
	}
}

func (identifier *IdentifierWrapper) GetFile(filename string) (string, bool) {
	if !identifier.CheckPathExist() {
		return "Identifier path doesn't exist.", false
	} else {
		file := path.Join(identifier.GetPath(), filename)
		if content, err := ioutil.ReadFile(file); err == nil {
			return string(content), true
		} else {
			Logger.Errorf("Failed to read file %v: %v", file, err)
			return "Failed to read file " + filename + " " + err.Error(), false
		}
	}
}

var ReloadReport bytes.Buffer
func (identifier *IdentifierWrapper) Reload() (bool, string) {
	// Logger hook
	ReloadReport.Reset()
	backend := logging.AddModuleLevel(logging.NewLogBackend(&ReloadReport, "", 0))
	backend.SetLevel(logging.NOTICE, "")
	logging.SetBackend(NormalLoggingBackend, backend)

	if !identifier.CheckPathExist() {
		return false, ""
	}
	identifier.clear()
	identifier.RegisterFolder(identifier.GetPath())
	identifier.Ready(nil)

	// FIXME: Make it graceful.
	logging.SetBackend(NormalLoggingBackend)
	return true, ReloadReport.String()
}

func (identifier *IdentifierWrapper) SetFile(filename, content string) (string, bool) {
	if !identifier.CheckPathExist() {
		return "Identifier path doesn't exist.", false
	} else {
		file := path.Join(identifier.GetPath(), filename)
		if err := ioutil.WriteFile(file, []byte(content), 0777); err == nil {
			return "", true
		} else {
			return err.Error(), false
		}
	}
}

func (identifier *IdentifierWrapper) Pull() (string, bool) {
	command := exec.Command("git", "-C", identifier.GetPath(), "pull", "-f", "origin", "master")
	if answer, err := command.Output(); err != nil {
		return err.Error(), false
	} else {
		return string(answer), true
	}
}

func (identifier *IdentifierWrapper) Push(message string) (string, bool) {
	command := exec.Command("git", "add", ".")
	command.Dir = identifier.GetPath()
	output := ""
	if answer, err := command.Output(); err != nil {
		return err.Error(), false
	} else {
		output += string(answer)
	}
	command = exec.Command("git", "commit", "-a", "-m", message)
	command.Dir = identifier.GetPath()
	if answer, err := command.Output(); err != nil {
		output += "Commit -- " + err.Error() + "\n"
	} else {
		output += string(answer)
	}
	command = exec.Command("git", "push", "origin", "master")
	command.Dir = identifier.GetPath()
	if answer, err := command.Output(); err != nil {
		return output + "Push -- " + err.Error(), false
	} else {
		output += string(answer)
	}
	return output, true
}

func (identifier *IdentifierWrapper) GetRuntimeList() (list map[string]interface{}) {
	list = make(map[string]interface{})
	deckNames := make([]string, 0)
	tagNames := make([]string, 0)
	setNames := make([]string, 0)
	for _, deck := range identifier.Decks {
		deckNames = append(deckNames, deck.Name)
	}
	for _, tag := range identifier.Tags {
		tagNames = append(tagNames, tag.Name)
	}
	for _, set := range identifier.CustomSets {
		setNames = append(setNames, set.Name)
	}
	list["decks"] = deckNames
	list["tags"] = tagNames
	list["sets"] = setNames
	return list
}

func (identifier *IdentifierWrapper) GetRuntimeStructure(class, name string) (map[string]interface{}, bool) {
	class = strings.ToLower(class)
	switch class {
	case "deck":
		for _, deck := range identifier.Decks {
			if deck.Name == name {
				return deck.ToJson(), true
			}
		}
	case "tag":
		for _, tag := range identifier.Tags {
			if tag.Name == name {
				return tag.ToJson(), true
			}
		}
	case "set":
		for _, set := range identifier.CustomSets {
			if set.Name == name {
				return SetToJson(set), true
			}
		}
		for _, set := range identifier.BindingEnvironment.Sets {
			if set.Name == name {
				return SetToJson(set), true
			}
		}
	}
	Logger.Warningf("Can't find %v named [%v] in identifier [%v].", strings.ToUpper(class), name, identifier.Name)
	return make(map[string]interface{}), false
}

func (identifier *IdentifierWrapper) GetCompilePreview(content string, newName string) (*IdentifierWrapper, string) {
	// Logger hook
	ReloadReport.Reset()
	backend := logging.AddModuleLevel(logging.NewLogBackend(&ReloadReport, "", 0))
	backend.SetLevel(logging.NOTICE, "")
	logging.SetBackend(NormalLoggingBackend, backend)

	target := GetWrappedIdentifier(newName)
	target.clear()
	target.RegisterDSL(content)
	// Stupid golang
	//mirror := Identifier{identifier.Name, identifier.Decks, identifier.Tags, identifier.GlobalTags, identifier.CustomSets, identifier.prototype, identifier.BindingEnvironment, identifier.SetNameHash}
	target.Ready(&identifier.Identifier)

	logging.SetBackend(NormalLoggingBackend)
	return target, ReloadReport.String()
}
