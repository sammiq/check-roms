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
	Method     string              `short:"m" long:"method" description:"method to use to match roms" choice:"sha1" choice:"md5" choice:"crc" default:"sha1"`
	Print      string              `short:"p" long:"print" description:"which information to print" choice:"all" choice:"files" choice:"sets" default:"all"`
	Rename     bool                `short:"r" long:"rename" description:"rename unambiguous misnamed files (only loose files and zipped sets supported)"`
	Positional struct {
		Files []string `description:"list of files to check against dat file (default: *)"`
	} `positional-args:"true"`
}

type gameRomMap = map[*xmlquery.Node]NodeSet

var checkCmd checkCommand

func (x *checkCommand) Execute(args []string) error {
	globalMap := make(gameRomMap)

	if checkCmd.Print == "all" {
		fmt.Println("--FILES--")
	}

	if len(checkCmd.Positional.Files) == 0 {
		dirName, err := os.Getwd()
		errorExit(err)
		checkCmd.Positional.Files = filesInDirectory(dirName)
	}

	for _, filePath := range checkCmd.Positional.Files {
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			vLog("ERROR: Cannot check %s, skipping\n", filePath)
			continue
		}

		fileExt := strings.TrimPrefix(filepath.Ext(filePath), ".")
		if _, ok := checkCmd.Exclude[fileExt]; ok {
			vLog("MSG: %s has excluded extension, skipping\n", filePath)
			continue
		}

		//skip anything that is not a regular file
		if !fileInfo.Mode().IsRegular() {
			vLog("MSG: %s is not a regular file, skipping\n", filePath)
			continue
		}

		if fileExt == "zip" {
			checkRomSet(filePath)
		} else {
			checkLooseRom(filePath, fileInfo, globalMap)
		}
	}

	if checkCmd.Print != "files" {
		if checkCmd.Print == "all" {
			fmt.Println("--SETS--")
		}
		for gameNode, roms := range globalMap {
			gameName := findAttr(gameNode, "name")
			if len(roms) == 0 {
				fmt.Printf("[ OK ]  %s\n", gameName)
			} else {
				for romNode := range roms {
					romAttr := mapAttr(romNode)
					romHash := strings.ToLower(romAttr[checkCmd.Method])
					romName := romAttr["name"]
					fmt.Printf("[WARN]  %s is missing %s %s\n", gameName, romHash, romName)
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

func findEntries(doc *xmlquery.Node, name string, hash string) ([]*xmlquery.Node, match) {
	list := matchRomEntriesByHexString(doc, checkCmd.Method, hash)
	vLog("MSG: Found %d entries matching hash %s, checking name %s...\n", len(list), hash, name)
	if len(list) == 0 {
		list = matchRomEntriesByName(doc, name)
		if len(list) == 0 {
			vLog("MSG: Found no entries matching %s %s...\n", hash, name)
			return list, matchNone
		}
		vLog("MSG: Found %d entries matching name %s...\n", len(list), name)
		return list, matchName
	}
	for _, node := range list {
		if findAttr(node, "name") == name {
			vLog("MSG: Found exact match for hash %s and name %s\n", hash, name)
			return []*xmlquery.Node{node}, matchAll
		}
	}
	vLog("MSG: Found %d entries matching hash %s, but found no match for name %s...\n", len(list), hash, name)
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
		findAttr(romNode, checkCmd.Method), findAttr(romNode, "name"), findAttr(gameNode, "name"))
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

func printMatch(prefix string, fileInfo os.FileInfo, fileHash string, romNode *xmlquery.Node, matchType match) {
	fileName := fileInfo.Name()
	romAttr := mapAttr(romNode)
	switch matchType {
	case matchAll:
		fmt.Printf("[ OK ] %s %s %s\n", prefix, fileHash, fileName)
	case matchHash:
		fmt.Printf("[WARN] %s %s %s - misnamed, should be %s\n", prefix,
			fileHash, fileName, romAttr["name"])
	case matchName:
		fmt.Printf("[BAD ] %s %s %s - incorrect, expected %s %s\n", prefix,
			fileHash, fileName, strings.ToLower(romAttr[checkCmd.Method]),
			printSizeMismatch(fileInfo, romAttr))

	}
}

func checkRom(filePath string, fileInfo os.FileInfo, fileHash string,
	gameMap gameRomMap, prefix string, renameFunc func(string, string) bool) {
	fileName := fileInfo.Name()
	print := checkCmd.Print != "sets"
	vLog("MSG: Checking %s %s...\n", fileHash, fileName)
	romList, matchType := findEntries(datfile, fileName, fileHash)
	matches := len(romList)
	if matches > 0 {
		for _, romNode := range romList {
			romName := findAttr(romNode, "name")
			if checkCmd.Rename && matches == 1 && matchType == matchHash {
				ok := renameFunc(filePath, romName)
				if print {
					if ok {
						fmt.Printf("[ OK ] %s - renamed from %s\n", romName, fileName)
					} else {
						fmt.Printf("[WARN] %s %s %s - misnamed, should be %s (cannot rename)\n", prefix,
							fileHash, fileName, romName)
					}
				}
			} else if print {
				printMatch(prefix, fileInfo, fileHash, romNode, matchType)
			}
			if matchType != matchName {
				//consider this as part of the game set
				updateGameMap(romNode, gameMap)
			}
		}
	} else if print {
		fmt.Printf("[MISS] %s %s %s\n", prefix, fileHash, fileName)
	}
}

func checkLooseRom(filePath string, fileInfo os.FileInfo, gameMap gameRomMap) {
	f, err := os.Open(filePath)
	errorExit(err)
	defer f.Close()

	checkRom(filePath, fileInfo, hashFile(f), gameMap, "", renameFile)
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
			ok := renameFile(filePath, expectedName)
			if print {
				if ok {
					fmt.Printf("[ OK ]  %s - renamed from %s\n", expectedName, fileName)
				} else {
					fmt.Printf("[WARN]  %s - misnamed, should be %s (cannot rename)\n", fileName, expectedName)
				}
			}
			return true
		} else if print {
			fmt.Printf("[WARN]  %s - misnamed, should be %s\n", fileName, expectedName)
		}
	} else {
		for romNode := range roms {
			romAttr := mapAttr(romNode)
			fileHash := strings.ToLower(romAttr[checkCmd.Method])
			romName := romAttr["name"]
			fmt.Printf("[WARN]  %s - %s is missing %s %s\n", fileName, gameName, fileHash, romName)
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
			vLog("MSG: %s is not a regular file, skipping\n", f.Name)
			continue
		}

		r, err := f.Open()
		if err != nil {
			log.Printf("ERROR: %s could not be opened: %s\n", fileName, err)
			return
		}
		defer r.Close()
		checkRom(f.Name, f.FileInfo(), hashFile(r), gameMap, " "+fileName+" -", func(string, string) bool { return false })
	}

	matches := len(gameMap)
	if matches == 0 {
		fmt.Printf("[ERR ]  %s - contains no recognized roms\n", fileName)
	} else {
		for gameNode, roms := range gameMap {
			matchFileToGame(filePath, fileName, gameNode, roms, checkCmd.Print != "files", checkCmd.Rename && matches == 1)
		}
	}
}

func init() {
	parser.AddCommand("check",
		"Check files against datfile",
		"This command will check a files against a datfile and determine if all files for a game are present",
		&checkCmd)
}
