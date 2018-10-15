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

func findEntries(doc *xmlquery.Node, name string, sha string) []*xmlquery.Node {
	list := matchEntriesBySha(doc, sha)
	vLog("MSG: Found %d entries matching hash %s, checking name %s...\n", len(list), sha, name)
	if len(list) == 0 {
		list = matchEntriesByName(doc, name)
		vLog("MSG: Found %d entries matching name %s...\n", len(list), name)
		return list
	}
	for _, node := range list {
		if findAttr(node, "name") == name {
			vLog("MSG: Found exact match for hash %s and name %s\n", sha, name)
			return []*xmlquery.Node{node}
		}
	}
	vLog("MSG: Found %d entries matching hash %s, but found no match for name %s...\n", len(list), sha, name)
	return list
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

func matchFileToRom(filePath string, fileName string, fileSha string,
	romNode *xmlquery.Node, print bool, rename bool) bool {
	romAttr := mapAttr(romNode)
	romSha := strings.ToLower(romAttr["sha1"])
	romName := romAttr["name"]
	if romSha != fileSha {
		if print {
			message := ""
			if opts.Size {
				romSize, err1 := strconv.ParseInt(romAttr["size"], 10, 64)
				fileInfo, err2 := os.Stat(filePath)
				if err1 == nil && err2 == nil && fileInfo.Size() != romSize {
					if fileInfo.Size() > romSize {
						message = fmt.Sprintf("(Possible overdump; size %s, expected %s)",
							iecPrefix(uint64(fileInfo.Size())), iecPrefix(uint64(romSize)))
					} else {
						message = fmt.Sprintf("(Possible underdump; size %s, expected %s)",
							iecPrefix(uint64(fileInfo.Size())), iecPrefix(uint64(romSize)))
					}
				}
			}
			fmt.Printf("[BAD ] %s %s - incorrect, expected %s %s\n", fileSha, fileName, romSha, message)
		}
		//if it's not the right rom, don't count it as a game
		return false
	} else if romName != fileName {
		if rename {
			if renameFile(filePath, romName) {
				if print {
					fmt.Printf("[ OK ] %s %s - renamed from %s\n", fileSha, romName, fileName)
				}
			}
		}

		if print {
			fmt.Printf("[WARN] %s %s - misnamed, should be %s\n", fileSha, fileName, romName)
		}
	} else {
		if print {
			fmt.Printf("[ OK ] %s %s\n", fileSha, fileName)
		}
	}
	return true
}

func checkRom(doc *xmlquery.Node,
	filePath string, fileSha string,
	gameMap map[*xmlquery.Node]NodeSet,
	print bool, rename bool) {
	fileName := filepath.Base(filePath)
	vLog("MSG: Checking %s %s...\n", fileSha, fileName)
	romList := findEntries(doc, fileName, fileSha)
	matches := len(romList)
	if matches > 0 {
		for _, romNode := range romList {
			if matchFileToRom(filePath, fileName, fileSha, romNode, print, rename && matches == 1) {
				updateGameMap(romNode, gameMap)
				break
			}
		}
	} else {
		if print {
			fmt.Printf("[MISS] %s %s\n", fileSha, fileName)
		}
	}
}

func checkLooseRom(doc *xmlquery.Node, filePath string, gameMap map[*xmlquery.Node]NodeSet) {
	checkRom(doc, filePath, shaHashFileAtPath(filePath), gameMap, opts.Print != "sets", opts.Rename)
}

func matchFileToGame(filePath string, fileName string,
	gameNode *xmlquery.Node, roms NodeSet, print bool, rename bool) bool {
	gameName := findAttr(gameNode, "name")
	if len(roms) == 0 {
		expectedName := gameName + ".zip"
		if fileName == expectedName {
			if print {
				fmt.Printf("[ OK ] %s - %s\n", fileName, gameName)
			}
			return true
		}

		if rename {
			if renameFile(filePath, expectedName) {
				if print {
					fmt.Printf("[ OK ] %s - renamed from %s\n", expectedName, fileName)
				}
				return true
			}
		}

		if print {
			fmt.Printf("[WARN] %s - misnamed, should be %s\n", fileName, expectedName)
		}
	} else {
		for romNode := range roms {
			romAttr := mapAttr(romNode)
			romSha := strings.ToLower(romAttr["sha1"])
			romName := romAttr["name"]
			fmt.Printf("[WARN] %s - %s is missing %s %s\n", fileName, gameName, romSha, romName)
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
		checkRom(doc, f.Name, shaHashFile(r), gameMap, false, false)
	}

	matches := len(gameMap)
	if matches == 0 {
		fmt.Printf("[ERR ] %s - contains no recognised roms\n", fileName)
	} else {
		for gameNode, roms := range gameMap {
			matchFileToGame(filePath, fileName, gameNode, roms, opts.Print != "roms", opts.Rename && matches == 1)
		}
	}
}

var opts struct {
	Datfile    string              `short:"d" long:"datfile" description:"dat file to use as reference database" required:"true"`
	Exclude    map[string]struct{} `short:"e" long:"exclude" description:"extension to exclude from file list (can be specified multiple times)"`
	Print      string              `short:"p" long:"print" description:"which information to print" choice:"roms" choice:"sets" choice:"all" default:"all"`
	Rename     bool                `short:"r" long:"rename" description:"rename unabiguous misnamed roms (only loose roms and zipped sets supported)"`
	Size       bool                `short:"s" long:"size" description:"check size on name only match (helps detect under/over-dumps)"`
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
		fmt.Println("--ROM FILES--")
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

	if opts.Print != "roms" {
		if opts.Print == "all" {
			fmt.Println("--GAME SETS--")
		}
		for gameNode, roms := range gameMap {
			gameName := findAttr(gameNode, "name")
			if len(roms) == 0 {
				fmt.Printf("[ OK ] %s\n", gameName)
			} else {
				for romNode := range roms {
					romAttr := mapAttr(romNode)
					romSha := strings.ToLower(romAttr["sha1"])
					romName := romAttr["name"]
					fmt.Printf("[WARN] %s is missing %s %s\n", gameName, romSha, romName)
				}
			}
		}
	}
}
