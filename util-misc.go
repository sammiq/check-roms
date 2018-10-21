package main

import (
	"fmt"
	"log"
	"strings"
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

func binaryPrefix(value uint64, units []string, maxUnit string, divisor float64) string {
	num := float64(value)
	for _, unit := range units {
		if num < divisor {
			return fmt.Sprintf("%3.2f%s", num, unit)
		}
		num = (num / divisor)
	}
	return fmt.Sprintf("%.2f%s", num, maxUnit)
}

func iecPrefix(value uint64) string {
	return binaryPrefix(value,
		[]string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB", "ZiB"}, "Yi",
		1024.0)
}

func indentPrintf(indent int, format string, a ...interface{}) {
	fmt.Print(strings.Repeat("\t", indent))
	fmt.Printf(format, a...)
}
