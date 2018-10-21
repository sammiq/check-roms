package main

import (
	"archive/zip"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/antchfx/xmlquery"
)

type checkCommand struct {
	Exclude    map[string]struct{} `short:"e" long:"exclude" description:"extension to exclude from file list (can be specified multiple times)"`
	Print      string              `short:"p" long:"print" description:"which information to print" choice:"files" choice:"sets" choice:"all" default:"all"`
	Rename     bool                `short:"r" long:"rename" description:"rename unabiguous misnamed files (only loose files and zipped sets supported)"`
	Positional struct {
		Files []string `description:"list of files to check against dat file" required:"true"`
	} `positional-args:"true" required:"true"`
}

type gameRomMap = map[*xmlquery.Node]NodeSet

var check checkCommand

func (x *checkCommand) Execute(args []string) error {
	globalMap := make(gameRomMap)

	if check.Print == "all" {
		fmt.Println("--FILES--")
	}
	for _, filePath := range check.Positional.Files {
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			vLog("ERROR: Cannot check %s, skipping", filePath)
			continue
		}

		fileExt := strings.TrimLeft(filepath.Ext(filePath), ".")
		if _, ok := check.Exclude[fileExt]; ok {
			vLog("MSG: %s has excluded extension, skipping", filePath)
			continue
		}

		//skip anything that is not a regular file
		if !fileInfo.Mode().IsRegular() {
			vLog("MSG: %s is not a regular file, skipping", filePath)
			continue
		}

		if fileExt == "zip" {
			checkRomSet(filePath)
		} else {
			checkLooseRom(filePath, fileInfo, globalMap)
		}
	}

	if check.Print != "files" {
		if check.Print == "all" {
			fmt.Println("--SETS--")
		}
		for gameNode, roms := range globalMap {
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
	return nil
}

type match int

const (
	matchNone match = iota
	matchName
	matchHash
	matchAll
)

func findEntries(doc *xmlquery.Node, name string, sha string) ([]*xmlquery.Node, match) {
	list := matchRomEntriesBySha(doc, sha)
	vLog("MSG: Found %d entries matching hash %s, checking name %s...\n", len(list), sha, name)
	if len(list) == 0 {
		list = matchRomEntriesByName(doc, name)
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

func updateGameMap(romNode *xmlquery.Node, gameMap gameRomMap) {
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

func printSizeMismatch(fileInfo os.FileInfo, romAttr map[string]string) string {
	message := ""
	fileSize := fileInfo.Size()
	romSize, err1 := strconv.ParseInt(romAttr["size"], 10, 64)
	if err1 == nil && fileSize != romSize {
		fileSizeText := iecPrefix(uint64(fileInfo.Size()))
		romSizeText := iecPrefix(uint64(romSize))
		if fileSize > romSize {
			message = fmt.Sprintf("(Possible overdump; size %s, expected %s)", fileSizeText, romSizeText)
		} else {
			message = fmt.Sprintf("(Possible underdump; size %s, expected %s)", fileSizeText, romSizeText)
		}
	}
	return message
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
		fmt.Printf("[BAD ] %s %s %s - incorrect, expected %s %s\n", prefix,
			fileSha, fileName, strings.ToLower(romAttr["sha1"]),
			printSizeMismatch(fileInfo, romAttr))

	}
}

func checkRom(filePath string, fileInfo os.FileInfo, fileSha string,
	gameMap gameRomMap, prefix string) string {
	fileName := fileInfo.Name()
	print := check.Print != "sets"
	vLog("MSG: Checking %s %s...\n", fileSha, fileName)
	romList, matchType := findEntries(datfile, fileName, fileSha)
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

func checkLooseRom(filePath string, fileInfo os.FileInfo, gameMap gameRomMap) {
	f, err := os.Open(filePath)
	errorExit(err)
	defer f.Close()

	print := check.Print != "sets"
	romName := checkRom(filePath, fileInfo, shaHashFile(f), gameMap, "")
	fileName := fileInfo.Name()
	if check.Rename && romName != "" && romName != fileName {
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

		if print {
			fmt.Printf("[WARN]  %s - misnamed, should be %s\n", fileName, expectedName)
		}
		if rename {
			if renameFile(filePath, expectedName) {
				if print {
					fmt.Printf("[ OK ]  %s - renamed from %s\n", expectedName, fileName)
				}
				return true
			}
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

func checkRomSet(filePath string) {
	gameMap := make(gameRomMap)
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
		checkRom(f.Name, f.FileInfo(), shaHashFile(r), gameMap, " "+fileName+" -")
	}

	matches := len(gameMap)
	if matches == 0 {
		fmt.Printf("[ERR ]  %s - contains no recognised roms\n", fileName)
	} else {
		for gameNode, roms := range gameMap {
			matchFileToGame(filePath, fileName, gameNode, roms, check.Print != "files", check.Rename && matches == 1)
		}
	}
}

func init() {
	parser.AddCommand("check",
		"Check files agains datfile",
		"This command will check a files against a datfile and determine if all files for a game are present",
		&check)
}
