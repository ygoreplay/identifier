package ygopro_deck_identifier

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type Condition struct {
	operator string
	number   int
}

var conditionStringReg, _ = regexp.Compile(`([><=]*=*)(\s*)(\d+)`)

func NewCondition(operator string, number int) Condition {
	return Condition{strings.TrimSpace(operator), number}
}

func CreateConditionFromString(string string) (Condition, bool) {
	if matches := conditionStringReg.FindStringSubmatch(string); matches == nil {
		Logger.Warningf("Can't realize the condition string %v", string)
		return Condition{}, false
	} else {
		number, _ := strconv.ParseInt(matches[3], 10, 64)
		condition := Condition{matches[1], int(number)}
		return condition, true
	}
}

func (condition Condition) Judge(value int) bool {
	switch condition.operator {
	case ">":
		return value > condition.number
	case "<":
		return value < condition.number
	case ">=":
		return value >= condition.number
	case "<=":
		return value <= condition.number
	case "=", "==":
		return value == condition.number
	case "&", "&&", "and":
		return value == condition.number
	case "|", "||", "or":
		return value >= 1
	default:
		return false
	}
}

func (condition Condition) String() string {
	return fmt.Sprintf("Condition [%v %v]", condition.operator, condition.number)
}
