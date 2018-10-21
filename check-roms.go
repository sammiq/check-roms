package main

import (
	"fmt"
	"os"

	"github.com/antchfx/xmlquery"
	flags "github.com/jessevdk/go-flags"
)

type options struct {
	Datfile string `short:"d" long:"datfile" description:"dat file to use as reference database"`
	Verbose bool   `short:"v" long:"verbose" description:"show lots more information than is probably necessary"`
}

var opts options
var datfile *xmlquery.Node

var parser = flags.NewParser(&opts, flags.Default)

func main() {
	parser.CommandHandler = func(cmd flags.Commander, args []string) error {
		if cmd != nil {
			datfile = checkDatFileAndOpen()
			return cmd.Execute(args)
		}
		return nil
	}
	_, err := parser.Parse()
	if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}

func checkDatFileAndOpen() *xmlquery.Node {
	if opts.Datfile == "" {
		opts.Datfile = readFirstLine("./.dat")
	}

	if opts.Datfile == "" {
		fmt.Println("the required flag `-d, --datfile` was not specified")
		os.Exit(1)
	}

	return parseDatFile(opts.Datfile)
}
