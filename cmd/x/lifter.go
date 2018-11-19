package main

import (
	"debug/pe"
	"sort"

	"github.com/pkg/errors"
)

// lifter is a binary executable to LLVM IR lifter.
type lifter struct {
	// Binary executable path.
	binPath string
	// Parse function addresses.
	funcAddrs Addrs
	// Parse basic block addresses.
	blockAddrs Addrs
	// Maps from basic block address to the set of non-continuous functions that
	// basic block belongs to.
	chunks map[Addr]map[Addr]bool
	// Functions.
	funcs []*Function
}

// newLifter returns a new lifter based on the given binary executable path.
func newLifter(binPath string) (*lifter, error) {
	l := &lifter{
		binPath: binPath,
	}
	// Parse function addresses.
	if err := parseJSON("funcs.json", &l.funcAddrs); err != nil {
		return nil, errors.WithStack(err)
	}
	sort.Sort(l.funcAddrs)
	// Parse basic block addresses.
	if err := parseJSON("blocks.json", &l.blockAddrs); err != nil {
		return nil, errors.WithStack(err)
	}
	sort.Sort(l.blockAddrs)
	// Parse non-continuous basic block addresses.
	if err := parseJSON("chunks.json", &l.chunks); err != nil {
		return nil, errors.WithStack(err)
	}
	return l, nil
}

// lift lifts the given binary executable to LLVM IR assembly.
func (l *lifter) lift() error {
	dbg.Printf("lift(binPath = %q)", l.binPath)
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
		dbg.Printf("=== [ section %q ] ===", sect.Name)
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
