package main

import (
	"flag"
	"fmt"
	"os"
)

const usage = `usage: refmt [-f format] INPUT_FILE|"-" OUTPUT_FILE|"-"

Converts from one encoding to another. Supported formats (and their file extensions):

	- HCL (.hcl)
	- JSON (.json)
	- YAML (.yaml or .yml)

If INPUT_FILE extension is not recognized or INPUT_FILE is "-" (stdin),
refmt will try to guess input format.

If OUTPUT_FILE is "-" (stdout), destination format is required to be
passed with -f flag.`

func die(v ...interface{}) {
	fmt.Fprintln(os.Stderr, v...)
	os.Exit(1)
}

func main() {
	flag.Parse()

	if flag.NArg() != 2 {
		die(usage)
	}

	if err := Refmt(flag.Arg(0), flag.Arg(1)); err != nil {
		die(err)
	}
}
