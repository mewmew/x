// The x tool lifts binary executables to LLVM IR assembly.
//
// Separation of concern is handled through reliance on oracles, which provide
// addresses of basic block, function calling conventions, type information,
// etc.
package main

import (
	"debug/pe"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"sort"

	"github.com/mewkiz/pkg/jsonutil"
	"github.com/mewkiz/pkg/term"
	"github.com/pkg/errors"
)

var (
	// dbg is a logger which logs debug messages with "x:" prefix to standard
	// error.
	dbg = log.New(os.Stderr, term.MagentaBold("x:")+" ", 0)
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

// lifter is a binary executable to LLVM IR lifter.
type lifter struct {
	// Binary executable path.
	binPath string
	// Parse function addresses.
	funcAddrs []Addr
	// Parse basic block addresses.
	blockAddrs []Addr
	// Functions.
	funcs []*Function
}

// newLifter returns a new lifter based on the given binary executable path.
func newLifter(binPath string) (*lifter, error) {
	l := &lifter{
		binPath: binPath,
	}
	// Parse function addresses.
	funcsPath := "funcs.json"
	dbg.Printf("jsonutil.ParseFile(jsonPath = %q)\n", funcsPath)
	if err := jsonutil.ParseFile(funcsPath, &l.funcAddrs); err != nil {
		return nil, errors.WithStack(err)
	}
	sort.Slice(l.funcAddrs, func(i, j int) bool {
		return l.funcAddrs[i] < l.funcAddrs[j]
	})
	// Parse basic block addresses.
	blocksPath := "blocks.json"
	dbg.Printf("jsonutil.ParseFile(jsonPath = %q)\n", blocksPath)
	if err := jsonutil.ParseFile(blocksPath, &l.blockAddrs); err != nil {
		return nil, errors.WithStack(err)
	}
	sort.Slice(l.blockAddrs, func(i, j int) bool {
		return l.blockAddrs[i] < l.blockAddrs[j]
	})
	return l, nil
}

// lift lifts the given binary executable to LLVM IR assembly.
func (l *lifter) lift() error {
	dbg.Printf("lift(binPath = %q)\n", l.binPath)
	file, err := pe.Open(l.binPath)
	if err != nil {
		return errors.WithStack(err)
	}
	defer file.Close()
	optHdr, ok := file.OptionalHeader.(*pe.OptionalHeader32)
	if !ok {
		return errors.New("support for 64-bit executables not yet implemented")
	}
	base := Addr(optHdr.ImageBase)
	for _, sect := range file.Sections {
		data, err := sect.Data()
		if err != nil {
			return errors.WithStack(err)
		}
		dbg.Printf("=== [ section %q ] ===\n", sect.Name)
		switch {
		case isExec(sect):
			rel := Addr(sect.VirtualAddress)
			addr := base + rel
			if err := l.liftCode(addr, data); err != nil {
				return errors.WithStack(err)
			}
		}
	}
	return nil
}
