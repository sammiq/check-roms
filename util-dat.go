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
