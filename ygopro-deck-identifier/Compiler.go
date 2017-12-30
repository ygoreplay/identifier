package ygopro_deck_identifier

import (
	"bufio"
	"os"
	"regexp"
	"strings"
	"path"
	"strconv"
)

const COMPILER_COMMENT_CHARACTER = "#"
const COMPILER_TAB_SPACE_LENGTH = 2
const COMPILER_TYPE_SPLIT_CHARACTER = ":"
const COMPILER_RESTRAIN_IDENTIFIER = "!"

var commentReg, _ = regexp.Compile("(?!\\\\" + COMPILER_COMMENT_CHARACTER + ")" + COMPILER_COMMENT_CHARACTER)
var spaceReg, _ = regexp.Compile(`^(\s+)`)
var setIdentifierReg, _ = regexp.Compile(`\[(.+?)]`)
var tagIdentifierReg, _ = regexp.Compile(`\((.+?)\)`)
var priorityIdentifierReg, _ = regexp.Compile(`\[(\d+?)]`)
var restrainReg, _ = regexp.Compile(`(.+?)(\s+?)(main|side|ex|ori|all)?(\s*?)(>|<|=)(=*)(\s*?)(\d+)`)
var tabSpaceString = strings.Repeat(" ", COMPILER_TAB_SPACE_LENGTH)

type Compiler struct {
	Root    *astNode
	Layers  []*astNode
	current *astNode
}

type originMessage struct {
	Line int
	Text string
	File string
}

type astNode struct {
	Type     string
	Value    string
	Origin *originMessage
	Children []*astNode
}

func newAstNode(Type string, Value string) (node *astNode) {
	node = new(astNode)
	node.Type = Type
	node.Value = Value
	return node
}

func newOriginMessage(Line int, Text string, File string) (message *originMessage) {
	message = new(originMessage)
	message.Line = Line
	message.Text = Text
	message.File = File
	return message
}

func (node astNode) String() string {
	return "[" + node.Type + ": " + node.Value + "]"
}

func (compiler *Compiler) clear() {
	compiler.Root = newAstNode("root", "")
	compiler.Layers = append(make([]*astNode, 0), compiler.Root)
	compiler.current = compiler.Root
}

func (compiler *Compiler) CompileFile(filename string) {
	compiler.clear()
	file, _ := os.Open(filename)
	scanner := bufio.NewScanner(file)
	lineNumber := 1
	for scanner.Scan() {
		line := scanner.Text()
		compiler.compileLine(line, newOriginMessage(lineNumber, line, filename))
		lineNumber += 1
	}
}

func (compiler *Compiler) CompileString(string string) {
	compiler.clear()
	for lineNumber, line := range strings.Split(string, "\n") {
		compiler.compileLine(line, newOriginMessage(lineNumber, line, "anonymous"))
	}
}

func (compiler *Compiler) compileLine(line string, message *originMessage) *astNode {
	if len(line) == 0 || strings.HasPrefix(line, COMPILER_COMMENT_CHARACTER) {
		return nil
	}
	compiler.removeLineComment(&line)
	tab := compiler.measureLineStrip(line)
	compiler.adjustLaysAndFocus(tab)
	line = strings.TrimSpace(line)
	if len(line) == 0 {
		return nil
	}
	Logger.Debug("Processing Compiler line " + line)
	node := compiler.compileLineContent(line, message)
	if node != nil {
		for tab >= len(compiler.Layers) {
			compiler.Layers = append(compiler.Layers, nil)
		}
		compiler.Layers[tab] = node
		compiler.current.Children = append(compiler.current.Children, node)
		node.Origin = message
		return node
	}
	return nil
}

func (compiler *Compiler) removeLineComment(line *string) {
	// FIXME: remove MAGIC character transform.
	middlewareLine := strings.Replace(*line, "\\"+COMPILER_COMMENT_CHARACTER, "^$^", -1)
	part := strings.Split(middlewareLine, COMPILER_COMMENT_CHARACTER)[0]
	*line = strings.Replace(part, "^$^", COMPILER_COMMENT_CHARACTER, -1)
}

func (compiler *Compiler) measureLineStrip(line string) int {
	line = strings.Replace(line, "\t", tabSpaceString, -1)
	line = strings.Replace(line, "\n", "", -1)
	tab := 0
	matches := spaceReg.FindStringSubmatch(line)
	if matches != nil {
		tab = len(matches[1])
	}
	return tab + 1
}

func (compiler *Compiler) adjustLaysAndFocus(tab int) {
	// Adds
	originTab := tab
	for tab -= 1; tab >= 0; tab -= 1 {
		if len(compiler.Layers) > tab && compiler.Layers[tab] != nil {
			compiler.current = compiler.Layers[tab]
			break
		}
	}
	// Cuts
	if originTab < len(compiler.Layers) {
		compiler.Layers = compiler.Layers[0:originTab]
	}
}

func (compiler *Compiler) compileLineContent(line string, message *originMessage) *astNode {
	lineType, exist := compiler.checkLineType(&line)
	if !exist {
		lineType = compiler.guessLineType(&line)
	}
	var node *astNode
	switch lineType {
	case "deck":
		node = compiler.generateClassificationNode(line, "deck")
	case "tag", "check tag":
		tagName := "check tag"
		if compiler.current == compiler.Root {
			tagName = "tag"
		}
		node = compiler.generateClassificationNode(line, tagName)
	case "refuse", "refuse tag":
		node = compiler.generateClassificationNode(line, "refuse tag")
	case "force", "force tag":
		node = compiler.generateClassificationNode(line, "force tag")
	case "classification":
		node = compiler.generateClassificationNode(line, "")
	case "and", "&", "&&":
		node = compiler.generateRestrainsNode(line, "and")
	case "or", "|", "||":
		node = compiler.generateRestrainsNode(line, "or")
	case "restrain group", "restrains":
		node = compiler.generateRestrainsNode(line, "")
	case "card":
		node = compiler.generateRestrainNode(line, "card")
	case "set", "series":
		if compiler.current == compiler.Root {
			node = newAstNode("set", strings.TrimSpace(line))
		} else {
			node = compiler.generateRestrainNode(line, "set")
		}
	case "restrain", COMPILER_RESTRAIN_IDENTIFIER:
		node = compiler.generateRestrainNode(line, "")
	case "set card":
		node = newAstNode("set card", strings.TrimSpace(line))
	case "inner set":
		node = newAstNode("inner set", strings.TrimSpace(line))
	case "priority":
		node = newAstNode("priority", strings.TrimSpace(line))
	case "config":
		node = newAstNode("config", strings.TrimSpace(line))
	default:
		node = nil
	}
	if node == nil {
		Logger.Warning("[" + path.Base(message.File) + "] L" + strconv.Itoa(message.Line) + " Can't parse " + lineType + " line: " + line)
	}
	return node
}

func (compiler *Compiler) checkLineType(line *string) (string, bool) {
	index := strings.Index(*line, COMPILER_TYPE_SPLIT_CHARACTER)
	if index < 0 {
		return "", false
	}
	lineType := (*line)[:index]
	*line = (*line)[(index + 1):]
	return strings.ToLower(lineType), true
}

func (compiler *Compiler) guessLineType(linePointer *string) string {
	// 神奇的 Clone 方法
	originLine := *linePointer
	line := originLine
	if strings.HasPrefix(line, COMPILER_RESTRAIN_IDENTIFIER) {
		*linePointer = line[len(COMPILER_RESTRAIN_IDENTIFIER):]
		return "restrain"
	} else if match := restrainReg.FindString(line); len(match) > 0 {
		*linePointer = match
		return "retrain"
	} else if match := tagIdentifierReg.FindString(line); len(match) > 0 {
		*linePointer = match
		return "tag"
	}
	line = strings.ToLower(line)
	// TODO: start_flags
	switch {
	case compiler.current.Type == "set":
		if matches := setIdentifierReg.FindStringIndex(originLine); matches != nil {
			*linePointer = originLine[matches[0]:matches[1]]
			return "inner set"
		} else {
			return "set card"
		}
	case compiler.Root == compiler.current:
		if setString := setIdentifierReg.FindString(line); len(setString) > 0 {
			return "set"
		} else {
			return "deck"
		}
	}
	return "unknown"
}

func (compiler *Compiler) generateClassificationNode(line string, class string) *astNode {
	priority := compiler.separateClassificationPriority(&line)
	if len(class) == 0 {
		class = compiler.guessClassificationType(&line)
	}
	classificationName := strings.TrimSpace(line)
	node := newAstNode(class, classificationName)
	node.Children = append(node.Children, newAstNode("priority", priority))
	return node
}

func (compiler *Compiler) guessClassificationType(classificationName *string) string {
	matches := tagIdentifierReg.FindStringSubmatch(*classificationName)
	if len(matches) == 0 {
		return "deck"
	} else {
		*classificationName = matches[1]
		if compiler.current.Type == "deck" {
			return "check tag"
		} else {
			return "tag"
		}
	}
}

func (compiler *Compiler) separateClassificationPriority(classificationName *string) string {
	priority := "0"
	matchIndices := priorityIdentifierReg.FindStringIndex(*classificationName)
	if matchIndices != nil {
		priority = (*classificationName)[matchIndices[0]+1 : matchIndices[1]-1]
		*classificationName = (*classificationName)[0:matchIndices[0]]
	}
	return priority
}

func (compiler *Compiler) generateRestrainNode(line, class string) *astNode {
	matches := restrainReg.FindStringSubmatch(line)
	if len(matches) == 0 {
		return nil
	}
	name := matches[1]
	if len(class) == 0 {
		class = compiler.guessRestrainType(&name)
	}
	field := matches[3]
	if len(field) == 0 {
		field = "all"
	}
	node := newAstNode("restrain", class)
	node.Children = append(node.Children, newAstNode("target", strings.TrimSpace(name)))
	node.Children = append(node.Children, newAstNode("range", field))
	node.Children = append(node.Children, newAstNode("condition", strings.Join(matches[5:9], "")))
	return node
}

func (compiler *Compiler) generateRestrainsNode(line, class string) *astNode {
	if len(class) == 0 {
		class = strings.ToLower(line)
		if class == "&&" || class == "&" {
			class = "and"
		}
		if class == "||" || class == "|" {
			class = "or"
		}
	}
	node := newAstNode("restrain", class)
	return node
}

func (compiler *Compiler) guessRestrainType(targetName *string) string {
	matches := setIdentifierReg.FindStringSubmatch(*targetName)
	if len(matches) == 0 {
		return "card"
	} else {
		*targetName = matches[1]
		return "set"
	}
}
