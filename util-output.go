package main

import (
	"fmt"
	"os"
	"strings"
)

type msgLevel int

const (
	levelDebug msgLevel = iota
	levelInfo
	levelWarn
	levelError
)

var outputLevel = levelError

func (level msgLevel) String() string {
	switch level {
	case levelDebug:
		return "DBG"
	case levelInfo:
		return "INFO"
	case levelWarn:
		return "WARN"
	case levelError:
		return "ERR"
	}
	return "???"
}

func output(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

func outputIndent(indent int, format string, args ...interface{}) {
	if indent == 0 {
		output(format, args...)
	} else {
		fmt.Print(strings.Repeat("\t", indent))
		fmt.Printf(format+"\n", args...)
	}
}

func message(level msgLevel, format string, args ...interface{}) {
	if level >= outputLevel {
		innerFormat := fmt.Sprintf("%-4.4s: %s\n", level, format)
		fmt.Fprintf(os.Stderr, innerFormat, args...)
	}
}
