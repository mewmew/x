// The x tool lifts binary executables to LLVM IR assembly.
//
// Separation of concern is handled through reliance on oracles, which provide
// addresses of basic block, function calling conventions, type information,
// etc.
package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"

	"github.com/mewkiz/pkg/term"
)

var (
	// dbg is a logger which logs debug messages with "x:" prefix to standard
	// error.
	dbg = log.New(os.Stderr, term.MagentaBold("x:")+" ", 0)
	// warn is a logger which logs warning messages with "warning:" prefix to
	// standard error.
	warn = log.New(os.Stderr, term.RedBold("warning:")+" ", 0)
)

func main() {
	// Parse command line arguments.
	var (
		// quiet specifies whether to suppress non-error messages.
		quiet bool
	)
	flag.BoolVar(&quiet, "q", false, "suppress non-error messages")
	flag.Parse()
	// Skip debug output if -q is set.
	if quiet {
		dbg.SetOutput(ioutil.Discard)
	}

	// Lift binary executables.
	for _, binPath := range flag.Args() {
		l, err := newLifter(binPath)
		if err != nil {
			log.Fatalf("%+v", err)
		}
		if err := l.lift(); err != nil {
			log.Fatalf("%+v", err)
		}
	}
}
