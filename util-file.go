package main

import (
	"bufio"
	"crypto/sha1"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func renameFile(filePath string, newName string) bool {
	newPath := filepath.Join(filepath.Dir(filePath), newName)
	err := os.Rename(filePath, newPath)
	if err != nil {
		log.Printf("ERROR: Unable to rename file : %s", err)
		return false
	}
	return true
}

func shaHashFile(reader io.Reader) string {
	hash := sha1.New()
	_, err := io.Copy(hash, reader)
	errorExit(err)
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func readFirstLine(filePath string) string {
	f, err := os.Open(filePath)
	if err != nil {
		vLog("MSG: Cannot open %s %s", filePath, err)
		return ""
	}
	defer f.Close()

	r := bufio.NewReader(f)
	var line []byte
	for prefix := true; prefix; {
		read, prefix, err := r.ReadLine()
		if err != nil && err != io.EOF {
			vLog("MSG: Error while reading %s %s", filePath, err)
		}
		line = append(line, read...)
		if !prefix {
			break
		}
	}
	return strings.Trim(string(line), " \t\v\f\r\x85\xa0")
}
