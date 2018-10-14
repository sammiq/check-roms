package main

import (
	"archive/zip"
	"crypto/sha1"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/antchfx/xmlquery"
	flags "github.com/jessevdk/go-flags"
)

func errorExit(err error) {
	if err != nil {
		log.Fatal("ERROR:", err)
	}
}

func vLog(fmt string, v ...interface{}) {
	if opts.Verbose {
		log.Printf(fmt, v...)
	}
}

func mapAttr(node *xmlquery.Node) map[string]string {
	attrMap := make(map[string]string)
	for _, attr := range node.Attr {
		attrMap[attr.Name.Local] = attr.Value
	}
	return attrMap
}

func findAttr(node *xmlquery.Node, name string) string {
	for _, attr := range node.Attr {
		if attr.Name.Local == name {
			return attr.Value
		}
	}
	return ""
}

func shaHashFile(reader io.Reader) string {
	hash := sha1.New()
	_, err := io.Copy(hash, reader)
	errorExit(err)
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func shaHashFileAtPath(filePath string) string {
	f, err := os.Open(filePath)
	errorExit(err)
	defer f.Close()

	return shaHashFile(f)
}

func matchEntriesBySha(doc *xmlquery.Node, sha string) []*xmlquery.Node {
	list := xmlquery.Find(doc, fmt.Sprintf("/datafile/game/rom[@sha1='%s']", strings.ToLower(sha)))
	if len(list) == 0 {
		//some DAT files have SHA1 in UPPER CASE
		list = xmlquery.Find(doc, fmt.Sprintf("/datafile/game/rom[@sha1='%s']", strings.ToUpper(sha)))
	}
	return list
}

func findEntries(doc *xmlquery.Node, name string, sha string) []*xmlquery.Node {
	list := matchEntriesBySha(doc, sha)
	listLength := len(list)
	if listLength == 0 {
		vLog("MSG: No match for hash %s, checking name %s...\n", sha, name)
		var b strings.Builder
		err := xml.EscapeText(&b, []byte(name))
		errorExit(err)
		//no SHA1 match, return any matching filename
		list = xmlquery.Find(doc, fmt.Sprintf("/datafile/game/rom[@name='%s']", b.String()))
		vLog("MSG: Found %d entries matching name %s...\n", len(list), name)
		return list
	}
	vLog("MSG: Found %d entries matching hash %s, checking name %s...\n", listLength, sha, name)
	for _, node := range list {
		romName := findAttr(node, "name")
		if romName == name {
			vLog("MSG: Found exact match for hash %s and name %s\n", sha, name)
			return []*xmlquery.Node{node}
		}
	}
	vLog("MSG: Found %d entries matching hash %s, but found no match for name %s...\n", listLength, sha, name)
	return list
}

//NodeSet is an alias a map of nodes without values as a node set
type NodeSet = map[*xmlquery.Node]struct{}

func makeRomSet(gameNode *xmlquery.Node) NodeSet {
	roms := make(NodeSet)
	for other := gameNode.FirstChild; other != nil; other = other.NextSibling {
		if other.Type == xmlquery.ElementNode && other.Data == "rom" {
			roms[other] = struct{}{}
		}
	}
	return roms
}

func updateGameMap(romNode *xmlquery.Node, gameMap map[*xmlquery.Node]NodeSet) {
	gameNode := romNode.Parent
	roms, ok := gameMap[gameNode]
	if !ok {
		roms = makeRomSet(gameNode)
		gameMap[gameNode] = roms
		vLog("MSG: Adding game %s with %d missing sets...\n", findAttr(gameNode, "name"), len(roms))
	}
	vLog("MSG: Removing rom %s %s from %s...\n",
		strings.ToLower(findAttr(romNode, "sha1")), findAttr(romNode, "name"), findAttr(gameNode, "name"))
	delete(roms, romNode)
	vLog("MSG: Game %s now has %d missing roms\n", findAttr(gameNode, "name"), len(roms))
}

func renameFile(filePath string, newName string) bool {
	newPath := filepath.Join(filepath.Dir(filePath), newName)
	err := os.Rename(filePath, newPath)
	if err != nil {
		log.Printf("ERROR: Unable to rename file : %s", err)
		return false
	}
	return true
}

func matchFileToRom(filePath string, fileName string, fileSha string,
	romNode *xmlquery.Node, print bool, rename bool) bool {
	romAttr := mapAttr(romNode)
	romSha := strings.ToLower(romAttr["sha1"])
	romName := romAttr["name"]
	if romSha != fileSha {
		if print {
			fmt.Printf("[BAD ] %s %s - incorrect, should be %s\n", fileSha, fileName, romSha)
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
	checkRom(doc, filePath, shaHashFileAtPath(filePath), gameMap, opts.Show != "sets", opts.Rename)
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
			matchFileToGame(filePath, fileName, gameNode, roms, opts.Show != "roms", opts.Rename && matches == 1)
		}
	}
}

var opts struct {
	Datfile    string              `short:"d" long:"datfile" description:"dat file to use as reference database" required:"true"`
	Exclude    map[string]struct{} `short:"e" long:"exclude" description:"extension to exclude from file list (can be specified multiple times)"`
	Show       string              `short:"s" long:"show" description:"which information to print" choice:"roms" choice:"sets" choice:"all" default:"all"`
	Rename     bool                `short:"r" long:"rename" description:"rename unabiguous misnamed roms (only loose roms and zipped sets supported)"`
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

	if opts.Show == "all" {
		fmt.Println("--ROM FILES--")
	}
	for _, filePath := range opts.Positional.Files {
		fileExt := strings.TrimLeft(filepath.Ext(filePath), ".")
		if _, ok := opts.Exclude[fileExt]; ok {
			continue
		}
		if fileExt == "zip" {
			checkRomSet(doc, filePath)
		} else {
			checkLooseRom(doc, filePath, gameMap)
		}
	}

	if opts.Show != "roms" {
		if opts.Show == "all" {
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
