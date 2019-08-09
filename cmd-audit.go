package main

import "time"

type auditCommand struct {
	Exclude     map[string]struct{} `short:"e" long:"exclude" description:"extension to exclude from file list (can be specified multiple times)"`
	Method      string              `short:"m" long:"method" description:"method to use to match roms" choice:"sha1" choice:"md5" choice:"crc" default:"sha1"`
	Rename      bool                `short:"r" long:"rename" description:"rename unambiguous misnamed files (only loose files and zipped sets supported)"`
	WorkerCount int                 `short:"w" long:"workers" description:"number of concurrent workers to use" default:"10"`
	Positional  struct {
		OutputFile string `description:"audit file for output (default: audit_<timestamp>.txt)"`
	} `positional-args:"true"`
}

var auditCmd auditCommand

func (x *auditCommand) Execute(args []string) error {
	checkCmd.AllSets = true
	auditCmd.Exclude["txt"] = struct{}{}
	checkCmd.Exclude = auditCmd.Exclude
	checkCmd.Method = auditCmd.Method
	checkCmd.Quiet = true
	checkCmd.Rename = auditCmd.Rename
	checkCmd.SortFiles = true
	checkCmd.SortSets = true
	checkCmd.WorkerCount = auditCmd.WorkerCount
	checkCmd.ViewSets = "all"
	checkCmd.Positional.Files = []string{}
	if auditCmd.Positional.OutputFile == "" {
		auditCmd.Positional.OutputFile = "audit_" + time.Now().Format("2006-01-02_15:04:05") + ".txt"
	}
	checkCmd.OutputFile = auditCmd.Positional.OutputFile

	return checkCmd.Execute(args)
}

func init() {
	parser.AddCommand("audit",
		"Audit files against datfile",
		"This command will audit the files and create a log about found and missing from a datfile",
		&auditCmd)
}
