package main

import (
	"crypto/sha1"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
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

func shaHashFileAtPath(filePath string) string {
	f, err := os.Open(filePath)
	errorExit(err)
	defer f.Close()

	return shaHashFile(f)
}
