package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/antchfx/xmlquery"
)

func parseDatFile(filePath string) *xmlquery.Node {
	f, err := os.Open(filePath)
	errorExit(err)
	defer f.Close()

	doc, err := xmlquery.Parse(f)
	errorExit(err)

	return doc
}

func matchRomEntriesByHexString(doc *xmlquery.Node, attribute string, hex string) []*xmlquery.Node {
	return xmlquery.Find(doc, fmt.Sprintf("/datafile/game/rom[@%s='%s' or @%s='%s']",
		attribute, strings.ToLower(hex), attribute, strings.ToUpper(hex)))
}

func matchRomEntriesBySha(doc *xmlquery.Node, sha string) []*xmlquery.Node {
	return matchRomEntriesByHexString(doc, "sha1", sha)
}

func matchRomEntriesByName(doc *xmlquery.Node, name string) []*xmlquery.Node {
	//limitation with xpath is that you can delimit with ' and use "
	//or delimit with " and use ' but you cannot escape either
	return xmlquery.Find(doc, fmt.Sprintf("/datafile/game/rom[@name=\"%s\"]", name))
}

func matchGameEntriesByName(doc *xmlquery.Node, name string) []*xmlquery.Node {
	return xmlquery.Find(doc, fmt.Sprintf("/datafile/game[@name=\"%s\"]", name))
}

func findRomEntriesWithName(doc *xmlquery.Node, name string) []*xmlquery.Node {
	return xmlquery.Find(doc, fmt.Sprintf("/datafile/game/rom[contains(@name, \"%s\")]", name))
}

func findGameEntriesWithName(doc *xmlquery.Node, name string) []*xmlquery.Node {
	return xmlquery.Find(doc, fmt.Sprintf("/datafile/game[contains(@name, \"%s\")]", name))
}

func findGameEntries(doc *xmlquery.Node) []*xmlquery.Node {
	return xmlquery.Find(doc, "/datafile/game")
}

type match int

const (
	matchNone match = iota
	matchName
	matchHash
	matchAll
)

func matchEntries(doc *xmlquery.Node, name string, hash string, hashMethod string) ([]*xmlquery.Node, match) {
	list := matchRomEntriesByHexString(doc, hashMethod, hash)
	listLength := len(list)
	message(levelDebug, "Found %d entries matching hash %s, checking name %s...", listLength, hash, name)
	if listLength == 0 {
		list = matchRomEntriesByName(doc, name)
		if listLength == 0 {
			message(levelInfo, "Found no entries matching %s %s...", hash, name)
			return list, matchNone
		}
		message(levelInfo, "Found %d entries matching name %s...", listLength, name)
		return list, matchName
	}

	message(levelDebug, "Looking for name match %s for hash %s...", name, hash)
	matched := make([]*xmlquery.Node, 0, listLength)
	for _, node := range list {
		foundName := findAttr(node, "name")
		message(levelDebug, "Matched name %s for hash %s", foundName, hash)
		if foundName == name {
			matched = append(matched, node)
		}
	}

	if len(matched) == 0 {
		message(levelInfo, "Found %d entries matching hash %s, but found no match for name %s...", listLength, hash, name)
		return list, matchHash
	}

	message(levelInfo, "Found %d entries matching hash %s and name %s", len(matched), hash, name)
	return matched, matchAll
}
