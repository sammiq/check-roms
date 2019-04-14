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
	vLog("MSG: Found %d entries matching hash %s, checking name %s...\n", listLength, hash, name)
	if listLength == 0 {
		list = matchRomEntriesByName(doc, name)
		if listLength == 0 {
			vLog("MSG: Found no entries matching %s %s...\n", hash, name)
			return list, matchNone
		}
		vLog("MSG: Found %d entries matching name %s...\n", listLength, name)
		return list, matchName
	}

	matched := make([]*xmlquery.Node, 0, listLength)
	for _, node := range list {
		if findAttr(node, "name") == name {
			matched = append(matched, node)
		}
	}

	if len(matched) == 0 {
		vLog("MSG: Found %d entries matching hash %s, but found no match for name %s...\n", listLength, hash, name)
		return list, matchHash
	}
	vLog("MSG: Found %d entries matching hash %s and name %s\n", len(matched), hash, name)
	return matched, matchAll
}
