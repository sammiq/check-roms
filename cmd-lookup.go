package main

import (
	"fmt"
	"strconv"

	"github.com/antchfx/xmlquery"
)

type lookupCommand struct {
	LookupKey  string `short:"k" long:"key" description:"key to use for lookup (ignored for game mode)" choice:"name" choice:"crc" choice:"md5" choice:"sha1" default:"name"`
	LookupMode string `short:"m" long:"mode" description:"element to lookup" choice:"rom" choice:"game" default:"rom"`
	ExactMatch bool   `short:"x" long:"exact" description:"use exact match (otherwise use substring match)"`
	Positional struct {
		Keys []string `description:"list of keys to lookup" required:"true"`
	} `positional-args:"true" required:"true"`
}

var lookupCmd lookupCommand

func (x *lookupCommand) Execute(args []string) error {
	for i, key := range lookupCmd.Positional.Keys {
		if i > 0 {
			fmt.Println("----")
		}
		var list []*xmlquery.Node
		if lookupCmd.LookupMode == "game" {
			if lookupCmd.ExactMatch {
				list = matchGameEntriesByName(datfile, key)
			} else {
				list = findGameEntriesWithName(datfile, key)
			}
			printGameEntries(list)
		} else {
			if lookupCmd.LookupKey == "name" {
				if lookupCmd.ExactMatch {
					list = matchRomEntriesByName(datfile, key)
				} else {
					list = findRomEntriesWithName(datfile, key)
				}
				printRomEntries(list)
			} else {
				printRomEntries(matchRomEntriesByHexString(datfile, lookupCmd.LookupKey, key))
			}
		}
	}
	return nil
}

func printRomEntries(list []*xmlquery.Node) {
	for _, rom := range list {
		printRomEntry(rom, 0)
	}
}

func printRomEntry(rom *xmlquery.Node, indent int) {
	printEntryAttributes(rom, indent)
	outputIndent(indent, "Contained in game:")
	printGameEntry(rom.Parent, indent+1)
}

func printGameEntries(list []*xmlquery.Node) {
	for _, game := range list {
		printGameEntry(game, 0)
	}
}

func printGameEntry(game *xmlquery.Node, indent int) {
	printEntryAttributes(game, indent)
	for el := game.FirstChild; el != nil; el = el.NextSibling {
		if el.Type != xmlquery.ElementNode {
			continue
		}
		outputIndent(indent, "%s: %s", el.Data, el.InnerText())
		printEntryAttributes(el, indent+1)
	}
}

func printEntryAttributes(node *xmlquery.Node, indent int) {
	attr := mapAttr(node)
	if _, ok := attr["name"]; ok {
		outputIndent(indent, "name: %s", attr["name"])
		delete(attr, "name")
	}
	for k, v := range attr {
		if k == "size" {
			val, err := strconv.ParseUint(v, 10, 64)
			if err == nil {
				v = iecPrefix(val)
			}
		}
		outputIndent(indent, "%s: %s", k, v)
	}
}

func init() {
	parser.AddCommand("lookup",
		"Lookup a datfile rom entry",
		"This command will search for a datfile rom entry that matches the given key (default is sha, use -m to change key type)",
		&lookupCmd)
}
