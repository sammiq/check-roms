package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/antchfx/xmlquery"
)

type checkCommand struct {
	AllSets     bool                `short:"a" long:"allsets" description:"report all sets that are missing"`
	Exclude     map[string]struct{} `short:"e" long:"exclude" description:"extension to exclude from file list (can be specified multiple times)"`
	Method      string              `short:"m" long:"method" description:"method to use to match roms" choice:"sha1" choice:"md5" choice:"crc" default:"sha1"`
	OutputFile  string              `short:"o" long:"output" description:"file for output"`
	Quiet       bool                `short:"q" long:"quiet" description:"do not print rom information for matches"`
	Rename      bool                `short:"r" long:"rename" description:"rename unambiguous misnamed files (only loose files and zipped sets supported)"`
	SortFiles   bool                `short:"f" long:"sort-files" description:"sort files alphabetically rather than by raw order"`
	SortSets    bool                `short:"s" long:"sort-sets" description:"sort sets alphabetically rather than by datfile order"`
	WorkerCount int                 `short:"w" long:"workers" description:"number of concurrent workers to use" default:"10"`
	ViewSets    string              `short:"v" long:"view" description:"which items to view" choice:"all" choice:"complete" choice:"missing" choice:"partial" default:"all"`
	Positional  struct {
		Files []string `description:"list of files to check against dat file (default: *)"`
	} `positional-args:"true"`
}

var checkCmd checkCommand

type gameInfo struct {
	GameName    string
	AllRoms     NodeSet
	MissingRoms NodeSet
}

type gameRomMap = map[*xmlquery.Node]*gameInfo
type nodeList = []*xmlquery.Node

func processFile(filePath string) nodeList {
	message(levelDebug, "Processing %s", filePath)
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		message(levelError, "Cannot check %s, skipping. Reason: %s", filePath, err)
		return nil
	}

	fileExt := strings.TrimPrefix(filepath.Ext(filePath), ".")
	if _, ok := checkCmd.Exclude[fileExt]; ok {
		message(levelInfo, "%s has excluded extension, skipping.", filePath)
		return nil
	}

	//skip anything that is not a regular file
	if !fileInfo.Mode().IsRegular() {
		message(levelWarn, "%s is not a regular file, skipping.", filePath)
		return nil
	}

	if fileExt == "zip" {
		return checkZip(filePath)
	}
	return checkFile(fileInfo, filePath)
}

func checkZip(zipFilePath string) nodeList {
	zipFileName := filepath.Base(zipFilePath)
	reader, err := zip.OpenReader(zipFilePath)
	if err != nil {
		message(levelError, "Cannot open %s, skipping. Reason: %s", zipFilePath, err)
		return nil
	}
	defer reader.Close()

	allMatches := make(nodeList, 0)
	for _, f := range reader.File {
		fileName := f.Name
		fileExt := strings.TrimPrefix(filepath.Ext(fileName), ".")
		if _, ok := checkCmd.Exclude[fileExt]; ok {
			message(levelInfo, "%s has excluded extension, skipping.", fileName)
			continue
		}

		fileInfo := f.FileInfo()
		if !fileInfo.Mode().IsRegular() {
			message(levelWarn, "%s is not a regular file, skipping.", fileName)
			continue
		}

		r, err := f.Open()
		if err != nil {
			message(levelError, "%s could not be opened, skipping. Reason: %s", fileName, err)
			continue
		}
		defer r.Close()
		allMatches = append(allMatches, findRomMatches(fileInfo, r, zipFileName, false, fileName)...)
	}

	if checkCmd.Rename {
		foundName := ""
		for _, match := range allMatches {
			gameNode := match.Parent
			gameName := findAttr(gameNode, "name")
			if foundName == "" {
				foundName = gameName
			} else if foundName != gameName {
				//there are multiple matches, so do not try to rename
				return allMatches
			}
		}

		newFileName := foundName + ".zip"
		if zipFileName != newFileName {
			ok := renameFile(zipFilePath, newFileName)
			if ok {
				message(levelInfo, "SET %s - renamed to %s from %s", foundName, newFileName, zipFileName)
			}
		}
	}
	return allMatches
}

func checkFile(fileInfo os.FileInfo, filePath string) nodeList {
	f, err := os.Open(filePath)
	if err != nil {
		message(levelError, "%s could not be opened : %s", filePath, err)
		return nil
	}
	defer f.Close()
	return findRomMatches(fileInfo, f, "", checkCmd.Rename, filePath)
}

func findRomMatches(fileInfo os.FileInfo, reader io.Reader, container string, rename bool, filePath string) nodeList {
	fileName := fileInfo.Name()
	fileHash := hashFile(reader, checkCmd.Method)

	romList, matchType := matchEntries(datfile, fileName, fileHash, checkCmd.Method)
	if matchType == matchNone {
		output("[MISS] %s %s %s - unknown, no match", fileHash, fileName, container)
	} else {
		for _, romNode := range romList {
			//if there is a single match just by hash, then rename if allowed
			romAttr := mapAttr(romNode)
			if rename && matchType == matchHash && len(romList) == 1 {
				romName := romAttr["name"]
				ok := renameFile(filePath, romName)
				if ok && !checkCmd.Quiet {
					message(levelInfo, "ROM %s - renamed from %s", romName, fileName)
					matchType = matchAll //it now matches all, so print as such
				}
			}
			printMatch(container, fileInfo, fileHash, romAttr, matchType)
		}
	}
	if matchType == matchAll || matchType == matchHash {
		return romList
	}
	return nil
}

func printMatch(container string, fileInfo os.FileInfo, fileHash string, romAttr map[string]string, matchType match) {
	fileName := fileInfo.Name()
	switch matchType {
	case matchAll:
		if !checkCmd.Quiet {
			output("[ OK ] %s %s %s",
				fileHash, fileName, container)
		}
	case matchHash:
		if !checkCmd.Quiet {
			output("[WARN] %s %s %s - misnamed, should be %s",
				fileHash, fileName, container,
				romAttr["name"])
		}
	case matchName:
		output("[BAD ] %s %s %s - incorrect, expected %s %s",
			fileHash, fileName, container,
			strings.ToLower(romAttr[checkCmd.Method]),
			printSizeMismatch(fileInfo, romAttr["size"]))

	}
}

func printSizeMismatch(fileInfo os.FileInfo, sizeText string) string {
	message := ""
	fileSize := fileInfo.Size()
	romSize, err1 := strconv.ParseInt(sizeText, 10, 64)
	if err1 == nil && fileSize != romSize {
		fileSizeText := iecPrefix(uint64(fileSize))
		romSizeText := iecPrefix(uint64(romSize))
		if fileSize > romSize {
			message = fmt.Sprintf("(Possible overdump; size %s, expected %s)", fileSizeText, romSizeText)
		} else {
			message = fmt.Sprintf("(Possible underdump; size %s, expected %s)", fileSizeText, romSizeText)
		}
	}
	return message
}

func updateGameMapFromGameNode(gameNode *xmlquery.Node, gameMap gameRomMap, gameList *[]*gameInfo) *gameInfo {
	info, ok := gameMap[gameNode]
	if !ok {
		gameName := findAttr(gameNode, "name")
		allRoms := childNodeSet(gameNode, "rom")
		//delete is in-place so do not use same reference, copy instead
		missingRoms := make(NodeSet)
		for key, value := range allRoms {
			missingRoms[key] = value
		}
		info = &gameInfo{gameName, allRoms, missingRoms}
		gameMap[gameNode] = info
		*gameList = append(*gameList, info)
		message(levelInfo, "Adding game %s with %d roms...", findAttr(gameNode, "name"), len(allRoms))
	}
	return info
}

func updateGameMapFromRomNode(romNode *xmlquery.Node, gameMap gameRomMap, gameList *[]*gameInfo) {
	gameNode := romNode.Parent
	info := updateGameMapFromGameNode(gameNode, gameMap, gameList)
	if _, ok := info.MissingRoms[romNode]; ok {
		message(levelDebug, "Removing rom %s %s from %s...",
			findAttr(romNode, checkCmd.Method), findAttr(romNode, "name"), findAttr(gameNode, "name"))
		delete(info.MissingRoms, romNode)
		message(levelDebug, "Game %s now has %d missing roms", findAttr(gameNode, "name"), len(info.MissingRoms))
	} else {
		message(levelInfo, "Missing rom %s %s in %s, possible duplicate rom detected",
			findAttr(romNode, checkCmd.Method), findAttr(romNode, "name"), findAttr(gameNode, "name"))
	}
}

func worker(id int, ic <-chan string, oc chan<- nodeList) {
	message(levelDebug, "Worker %d Starting", id)
	for input := range ic {
		message(levelDebug, "Worker %d Processing: %s", id, input)
		oc <- processFile(input)
	}
	message(levelDebug, "Worker %d Exiting", id)
}

func (x *checkCommand) Execute(args []string) error {
	if checkCmd.OutputFile != "" {
		f, err := os.Create(checkCmd.OutputFile)
		if err != nil {
			message(levelError, "%s could not be created : %s", checkCmd.OutputFile, err)
			return err
		}
		outputFile = f
	}
	gameMap := make(gameRomMap)
	gameList := make([]*gameInfo, 0)
	if checkCmd.AllSets {
		//add everything from the datfile to the gameRomMap
		for _, game := range findGameEntries(datfile) {
			updateGameMapFromGameNode(game, gameMap, &gameList)
		}
	}

	if len(checkCmd.Positional.Files) == 0 {
		dirName, err := os.Getwd()
		errorExit(err)
		checkCmd.Positional.Files = filesInDirectory(dirName)
	}
	if checkCmd.SortFiles {
		sort.Strings(checkCmd.Positional.Files)
	}
	length := len(checkCmd.Positional.Files)

	numWorkers := checkCmd.WorkerCount

	//init worker channels
	inputs := make(chan string, length)        //need enough to feed each file into a worker
	outputs := make(chan nodeList, numWorkers) //need enough to feed a result out of each worker

	message(levelDebug, "Initializing %d workers", numWorkers)
	for w := 1; w <= numWorkers; w++ {
		go worker(w, inputs, outputs)
	}

	if checkCmd.Quiet {
		output("--INCORRECT FILES--")
	} else {
		output("--FILES--")
	}
	for _, filePath := range checkCmd.Positional.Files {
		inputs <- filePath
	}

	//gather results and update maps
	for a := 0; a < length; a++ {
		thisResult := <-outputs
		for _, romNode := range thisResult {
			updateGameMapFromRomNode(romNode, gameMap, &gameList)
		}
	}

	//close inputs and close workers
	close(inputs)

	output("--SETS--")
	if checkCmd.SortSets {
		sort.Slice(gameList, func(i, j int) bool { return gameList[i].GameName < gameList[j].GameName })
	}

	completeSets := 0
	missingSets := 0
	partialSets := 0
	for _, info := range gameList {
		numMissing := len(info.MissingRoms)
		if numMissing == 0 {
			completeSets++
			if checkCmd.ViewSets == "all" || checkCmd.ViewSets == "complete" {
				output("[ OK ]  %s", info.GameName)
			}
		} else if len(info.AllRoms) == numMissing {
			missingSets++
			if checkCmd.ViewSets == "all" || checkCmd.ViewSets == "missing" {
				output("[MISS]  %s", info.GameName)
			}
		} else {
			partialSets++
			if checkCmd.ViewSets == "all" || checkCmd.ViewSets == "partial" {
				output("[WARN]  %s is missing:", info.GameName)
				for romNode := range info.MissingRoms {
					romAttr := mapAttr(romNode)
					romHash := strings.ToLower(romAttr[checkCmd.Method])
					romName := romAttr["name"]
					output("        %s %s", romHash, romName)
				}
			}
		}
	}
	output("--SET STATISTICS--")
	output("\tComplete: %d", completeSets)
	output("\tPartial: %d", partialSets)
	output("\tMissing: %d", missingSets)

	return nil
}

func init() {
	parser.AddCommand("check",
		"Check files against datfile",
		"This command will check files against a datfile and determine if all files for a game are present",
		&checkCmd)
}
