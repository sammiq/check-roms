package main

import (
	"fmt"
	"log"
)

func errorExit(err error) {
	if err != nil {
		log.Fatal("ERROR:", err)
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
