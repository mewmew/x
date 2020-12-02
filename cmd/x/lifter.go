package main

import (
	"debug/pe"
	"sort"

	"github.com/llir/llvm/ir"
	"github.com/pkg/errors"
)

// lifter is a binary executable to LLVM IR lifter.
type lifter struct {
	// x86 disassembler.

	// Binary executable path.
	binPath string
	// Function addresses.
	funcAddrs Addrs
	// Basic block addresses.
	blockAddrs Addrs
	// Maps from basic block address to the set of non-continuous functions that
	// basic block belongs to.
	chunks map[Addr]map[Addr]bool

	// x86 functions.
	asmFuncs []*Function

	// LLVM IR lifter.

	// Maps from function address to LLVM IR function.
	funcs map[Addr]*ir.Function
}

// newLifter returns a new lifter based on the given binary executable path.
func newLifter(binPath string) (*lifter, error) {
	l := &lifter{
		binPath: binPath,
		funcs:   make(map[Addr]*ir.Function),
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
func (l *lifter) lift() (*ir.Module, error) {
	dbg.Printf("lift(binPath = %q)", l.binPath)
	// Parse PE file.
	file, err := pe.Open(l.binPath)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer file.Close()
	optHdr, ok := file.OptionalHeader.(*pe.OptionalHeader32)
	if !ok {
		return nil, errors.New("support for 64-bit executables not yet implemented")
	}
	base := Addr(optHdr.ImageBase)
	// Decode x86 instructions of binary executable.
	for _, sect := range file.Sections {
		data, err := sect.Data()
		if err != nil {
			return nil, errors.WithStack(err)
		}
		dbg.Printf("=== [ section %q ] ===", sect.Name)
		switch {
		case isExec(sect):
			rel := Addr(sect.VirtualAddress)
			addr := base + rel
			if err := l.decodeCodeSection(addr, data); err != nil {
				return nil, errors.WithStack(err)
			}
		}
	}
	// Translate x86 binary executable to LLVM IR module.
	m, err := l.translate()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return m, nil
}
