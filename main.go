package main

import (
	"flag"
	"fmt"
	"os"
)

const usage = `usage:

	refmt [-t type] INPUT_FILE|"-" OUTPUT_FILE|"-"

Converts from one encoding to another. Supported formats (and their file extensions):

	- HCL (.hcl or .tf)
	- JSON (.json)
	- YAML (.yaml or .yml)

If INPUT_FILE's extension is not recognized or INPUT_FILE is "-" (stdin),
refmt will try to guess input format.

If OUTPUT_FILE is "-" (stdout), destination format type is required to be
passed with -t flag.

	refmt merge [-t type] ORIGINAL_FILE|"-" MIXIN_FILE|"-" OUTPUT_FILE|"-"

Merges the object defined in ORIGINAL_FILE with the object from MIXIN_FILE, writing
the resulting object to the OUTPUT_FILE.

The ORIGINAL_FILE, MIXIN_FILE and OUTPUT_FILE can have different encodings.

If ORIGINAL_FILE's extension is not recognized or ORIGINAL_FILE is "-" (stdin),
refmt will try to guess original format.

If ORIGINAL_FILE does not exist or is empty, refmt is going to use empty
object instead.

If MIXIN_FILE's extension is not recognized or MIXIN_FILE is "-" (stdin),
refmt will try to guess mixin format.

If OUTPUT_FILE is "-" (stdout), destination format type is required to be
passed with -t flag.`

func die(v ...interface{}) {
	fmt.Fprintln(os.Stderr, v...)
	os.Exit(1)
}

func main() {
	flag.Parse()

	if flag.NArg() != 2 && flag.NArg() != 4 {
		die(usage)
	}

	var err error
	switch flag.Arg(0) {
	case "merge":
		err = Merge(flag.Arg(1), flag.Arg(2), flag.Arg(3))
	default:
		err = Refmt(flag.Arg(0), flag.Arg(1))
	}

	if err != nil {
		die(err)
	}
}
