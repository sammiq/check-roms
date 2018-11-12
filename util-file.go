package main

import (
	"bufio"
	"crypto/md5"
	"crypto/sha1"
	"fmt"
	"hash/crc32"
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

func md5HashFile(reader io.Reader) string {
	hash := md5.New()
	_, err := io.Copy(hash, reader)
	errorExit(err)
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func crcHashFile(reader io.Reader) string {
	hash := crc32.NewIEEE()
	_, err := io.Copy(hash, reader)
	errorExit(err)
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func hashFile(reader io.Reader) string {
	switch checkCmd.Method {
	case "sha1":
		return shaHashFile(reader)
	case "md5":
		return md5HashFile(reader)
	case "crc":
		return crcHashFile(reader)
	}
	return ""
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

func filesInDirectory(dirName string) []string {
	dirFile, err := os.Open(dirName)
	errorExit(err)

	infos, err := dirFile.Readdir(0)
	errorExit(err)

	var fileNames []string
	for _, info := range infos {
		//ignore non-regular files
		if !info.Mode().IsRegular() {
			continue
		}

		//ignore dotfiles
		if strings.HasPrefix(info.Name(), ".") {
			continue
		}

		fileNames = append(fileNames, filepath.Join(dirName, info.Name()))
	}

	return fileNames
}
