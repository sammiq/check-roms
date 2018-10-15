package main

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/antchfx/xmlquery"
	flags "github.com/jessevdk/go-flags"
)

type match int

const (
	matchNone match = iota
	matchName
	matchHash
	matchAll
)

func matchEntriesBySha(doc *xmlquery.Node, sha string) []*xmlquery.Node {
	list := xmlquery.Find(doc, fmt.Sprintf("/datafile/game/rom[@sha1='%s']", strings.ToLower(sha)))
	if len(list) == 0 {
		//some DAT files have SHA1 in UPPER CASE
		list = xmlquery.Find(doc, fmt.Sprintf("/datafile/game/rom[@sha1='%s']", strings.ToUpper(sha)))
	}
	return list
}

func matchEntriesByName(doc *xmlquery.Node, name string) []*xmlquery.Node {
	var b strings.Builder
	err := xml.EscapeText(&b, []byte(name))
	errorExit(err)
	return xmlquery.Find(doc, fmt.Sprintf("/datafile/game/rom[@name='%s']", b.String()))
}

func findEntries(doc *xmlquery.Node, name string, sha string) ([]*xmlquery.Node, match) {
	list := matchEntriesBySha(doc, sha)
	vLog("MSG: Found %d entries matching hash %s, checking name %s...\n", len(list), sha, name)
	if len(list) == 0 {
		list = matchEntriesByName(doc, name)
		if len(list) == 0 {
			vLog("MSG: Found no entries matching %s %s...\n", len(list), sha, name)
			return list, matchNone
		}
		vLog("MSG: Found %d entries matching name %s...\n", len(list), name)
		return list, matchName
	}
	for _, node := range list {
		if findAttr(node, "name") == name {
			vLog("MSG: Found exact match for hash %s and name %s\n", sha, name)
			return []*xmlquery.Node{node}, matchAll
		}
	}
	vLog("MSG: Found %d entries matching hash %s, but found no match for name %s...\n", len(list), sha, name)
	return list, matchHash
}

func updateGameMap(romNode *xmlquery.Node, gameMap map[*xmlquery.Node]NodeSet) {
	gameNode := romNode.Parent
	roms, ok := gameMap[gameNode]
	if !ok {
		roms = childNodeSet(gameNode, "rom")
		gameMap[gameNode] = roms
		vLog("MSG: Adding game %s with %d missing sets...\n", findAttr(gameNode, "name"), len(roms))
	}
	vLog("MSG: Removing rom %s %s from %s...\n",
		findAttr(romNode, "sha1"), findAttr(romNode, "name"), findAttr(gameNode, "name"))
	delete(roms, romNode)
	vLog("MSG: Game %s now has %d missing roms\n", findAttr(gameNode, "name"), len(roms))
}

func printMatch(prefix string, fileInfo os.FileInfo, fileSha string, romNode *xmlquery.Node, matchType match) {
	fileName := fileInfo.Name()
	romAttr := mapAttr(romNode)
	switch matchType {
	case matchAll:
		fmt.Printf("[ OK ] %s %s %s\n", prefix, fileSha, fileName)
	case matchHash:
		fmt.Printf("[WARN] %s %s %s - misnamed, should be %s\n", prefix,
			fileSha, fileName, romAttr["name"])
	case matchName:
		message := ""
		if opts.Size {
			fileSize := fileInfo.Size()
			romSize, err1 := strconv.ParseInt(romAttr["size"], 10, 64)
			if err1 == nil && fileSize != romSize {
				if fileSize > romSize {
					message = fmt.Sprintf("(Possible overdump; size %s, expected %s)",
						iecPrefix(uint64(fileInfo.Size())), iecPrefix(uint64(romSize)))
				} else {
					message = fmt.Sprintf("(Possible underdump; size %s, expected %s)",
						iecPrefix(uint64(fileInfo.Size())), iecPrefix(uint64(romSize)))
				}
			}
		}
		fmt.Printf("[BAD ] %s %s %s - incorrect, expected %s %s\n", prefix,
			fileSha, fileName, strings.ToLower(romAttr["sha1"]), message)

	}
}

func checkRom(doc *xmlquery.Node, filePath string, fileInfo os.FileInfo, fileSha string,
	gameMap map[*xmlquery.Node]NodeSet, prefix string) string {
	fileName := fileInfo.Name()
	print := opts.Print != "sets"
	vLog("MSG: Checking %s %s...\n", fileSha, fileName)
	romList, matchType := findEntries(doc, fileName, fileSha)
	matches := len(romList)
	if matches > 0 {
		for _, romNode := range romList {
			romName := findAttr(romNode, "name")
			if print {
				printMatch(prefix, fileInfo, fileSha, romNode, matchType)
			}
			if matchType != matchName {
				//consider this as part of the game set
				updateGameMap(romNode, gameMap)
			}
			if matches == 1 {
				//unambiguous name, so return it
				return romName
			}
		}
	} else if print {
		fmt.Printf("[MISS] %s %s %s\n", prefix, fileSha, fileName)
	}
	//not found or ambiguous
	return ""
}

func checkLooseRom(doc *xmlquery.Node, filePath string, gameMap map[*xmlquery.Node]NodeSet) {
	f, err := os.Open(filePath)
	errorExit(err)
	defer f.Close()

	fileInfo, err := f.Stat()
	errorExit(err)

	print := opts.Print != "sets"
	romName := checkRom(doc, filePath, fileInfo, shaHashFile(f), gameMap, "")
	fileName := fileInfo.Name()
	if opts.Rename && romName != "" && romName != fileName {
		if renameFile(filePath, romName) {
			if print {
				fmt.Printf("[ OK ] %s - renamed from %s\n", romName, fileName)
			}
		}
	}
}

func matchFileToGame(filePath string, fileName string,
	gameNode *xmlquery.Node, roms NodeSet, print bool, rename bool) bool {
	gameName := findAttr(gameNode, "name")
	if len(roms) == 0 {
		expectedName := gameName + ".zip"
		if fileName == expectedName {
			if print {
				fmt.Printf("[ OK ]  %s - %s\n", fileName, gameName)
			}
			return true
		}

		if rename {
			if renameFile(filePath, expectedName) {
				if print {
					fmt.Printf("[ OK ]  %s - renamed from %s\n", expectedName, fileName)
				}
				return true
			}
		}

		if print {
			fmt.Printf("[WARN]  %s - misnamed, should be %s\n", fileName, expectedName)
		}
	} else {
		for romNode := range roms {
			romAttr := mapAttr(romNode)
			romSha := strings.ToLower(romAttr["sha1"])
			romName := romAttr["name"]
			fmt.Printf("[WARN]  %s - %s is missing %s %s\n", fileName, gameName, romSha, romName)
		}
	}
	return false
}

func checkRomSet(doc *xmlquery.Node, filePath string) {
	gameMap := make(map[*xmlquery.Node]NodeSet)
	fileName := filepath.Base(filePath)

	reader, err := zip.OpenReader(filePath)
	if err != nil {
		log.Printf("ERROR: %s could not be opened : %s\n", fileName, err)
		return
	}
	defer reader.Close()

	for _, f := range reader.File {
		//skip anything that is not a regular file
		if !f.Mode().IsRegular() {
			vLog("MSG: %s is not a regular file, skipping", f.Name)
			continue
		}

		r, err := f.Open()
		if err != nil {
			log.Printf("ERROR: %s could not be opened: %s\n", fileName, err)
			return
		}
		defer r.Close()
		checkRom(doc, f.Name, f.FileInfo(), shaHashFile(r), gameMap, " "+fileName+" -")
	}

	matches := len(gameMap)
	if matches == 0 {
		fmt.Printf("[ERR ]  %s - contains no recognised roms\n", fileName)
	} else {
		for gameNode, roms := range gameMap {
			matchFileToGame(filePath, fileName, gameNode, roms, opts.Print != "files", opts.Rename && matches == 1)
		}
	}
}

var opts struct {
	Datfile    string              `short:"d" long:"datfile" description:"dat file to use as reference database" required:"true"`
	Exclude    map[string]struct{} `short:"e" long:"exclude" description:"extension to exclude from file list (can be specified multiple times)"`
	Print      string              `short:"p" long:"print" description:"which information to print" choice:"files" choice:"sets" choice:"all" default:"all"`
	Rename     bool                `short:"r" long:"rename" description:"rename unabiguous misnamed files (only loose files and zipped sets supported)"`
	Size       bool                `short:"s" long:"size" description:"check size on name only match (helps detect possible under/over-dumps)"`
	Verbose    bool                `short:"v" long:"verbose" description:"show lots more information than is probably necessary"`
	Positional struct {
		Files []string `description:"list of files to check against dat file" required:"true"`
	} `positional-args:"true" required:"true"`
}

func main() {
	_, err := flags.Parse(&opts)
	if err != nil {
		os.Exit(1)
	}

	f, err := os.Open(opts.Datfile)
	errorExit(err)
	defer f.Close()

	doc, err := xmlquery.Parse(f)
	errorExit(err)

	gameMap := make(map[*xmlquery.Node]NodeSet)

	if opts.Print == "all" {
		fmt.Println("--FILES--")
	}
	for _, filePath := range opts.Positional.Files {
		f, err := os.Stat(filePath)
		if err != nil {
			vLog("ERROR: Cannot check %s, skipping", filePath)
			continue
		}

		fileExt := strings.TrimLeft(filepath.Ext(filePath), ".")
		if _, ok := opts.Exclude[fileExt]; ok {
			vLog("MSG: %s has excluded extension, skipping", filePath)
			continue
		}

		//skip anything that is not a regular file
		if !f.Mode().IsRegular() {
			vLog("MSG: %s is not a regular file, skipping", filePath)
			continue
		}

		if fileExt == "zip" {
			checkRomSet(doc, filePath)
		} else {
			checkLooseRom(doc, filePath, gameMap)
		}
	}

	if opts.Print != "files" {
		if opts.Print == "all" {
			fmt.Println("--SETS--")
		}
		for gameNode, roms := range gameMap {
			gameName := findAttr(gameNode, "name")
			if len(roms) == 0 {
				fmt.Printf("[ OK ]  %s\n", gameName)
			} else {
				for romNode := range roms {
					romAttr := mapAttr(romNode)
					romSha := strings.ToLower(romAttr["sha1"])
					romName := romAttr["name"]
					fmt.Printf("[WARN]  %s is missing %s %s\n", gameName, romSha, romName)
				}
			}
		}
	}
}
