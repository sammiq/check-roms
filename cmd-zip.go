package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/antchfx/xmlquery"
)

type zipCommand struct {
	Exclude    map[string]struct{} `short:"e" long:"exclude" description:"extension to exclude from file list (can be specified multiple times)"`
	InfoZip    bool                `short:"i" long:"infozip" description:"use info-zip command line tool instead of internal zip function"`
	Positional struct {
		Files []string `description:"list of files to check and zip" required:"true"`
	} `positional-args:"true" required:"true"`
}

var zipCmd zipCommand

func (x *zipCommand) Execute(args []string) error {
	gameFiles := make(map[*xmlquery.Node][]string)

	for _, filePath := range zipCmd.Positional.Files {
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			vLog("ERROR: Cannot check %s, skipping\n", filePath)
			continue
		}

		fileExt := strings.TrimLeft(filepath.Ext(filePath), ".")
		if _, ok := zipCmd.Exclude[fileExt]; ok {
			vLog("MSG: %s has excluded extension, skipping\n", filePath)
			continue
		}

		//skip anything that is not a regular file
		if !fileInfo.Mode().IsRegular() {
			vLog("MSG: %s is not a regular file, skipping\n", filePath)
			continue
		}

		fin, err := os.Open(filePath)
		errorExit(err)
		defer fin.Close()

		matches := matchRomEntriesBySha(datfile, shaHashFile(fin))
		vLog("MSG: found %d matches for %s\n", len(matches), filePath)
		for _, match := range matches {
			if match.SelectAttr("name") == filepath.Base(filePath) {
				list, ok := gameFiles[match.Parent]
				if !ok {
					list = make([]string, 0)
				}
				gameFiles[match.Parent] = append(list, filePath)
			}
		}
	}

	for game, fileList := range gameFiles {
		gameName := game.SelectAttr("name")
		roms := game.SelectElements("rom")
		vLog("MSG: game %s needs %d file(s), found %d\n", gameName, len(roms), len(fileList))
		if len(roms) == len(fileList) {
			fmt.Printf("Creating %s with %d file(s)...\n", gameName+".zip", len(fileList))
			if zipCmd.InfoZip {
				externalZip(gameName+".zip", fileList)
			} else {
				internalZip(gameName+".zip", fileList)
			}
			fmt.Printf("Finished writing %s\n", gameName+".zip")
		}
	}
	return nil
}

func externalZip(zipFileName string, fileList []string) {
	argList := []string{zipFileName}
	argList = append(argList, fileList...)
	cmd := exec.Command("zip", argList...)
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	errorExit(err)
}

func internalZip(zipFileName string, fileList []string) {
	zipFile, err := os.Create(zipFileName)
	errorExit(err)
	defer zipFile.Close()
	zipper := zip.NewWriter(zipFile)

	for _, filePath := range fileList {
		fileName := filepath.Base(filePath)
		fmt.Printf("Writing %s to %s...", fileName, zipFileName)
		romFile, err := os.Open(filePath)
		errorExit(err)
		defer romFile.Close()

		fileInfo, err := os.Stat(filePath)
		errorExit(err)

		header := &zip.FileHeader{
			Name:     fileName,
			Method:   zip.Deflate,
			Modified: fileInfo.ModTime(),
		}
		header.SetMode(fileInfo.Mode())
		fileWriter, err := zipper.CreateHeader(header)
		errorExit(err)
		io.Copy(fileWriter, romFile)
		fmt.Println("Done!")
	}

	err = zipper.Close()
	errorExit(err)
}

func init() {
	parser.AddCommand("zip",
		"Zip complete roms into sets",
		"This command will search for all files relating to a game and zip them together into a zip file",
		&zipCmd)
}
