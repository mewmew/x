// The x tool lifts binary executables to LLVM IR assembly.
package main

import (
	"debug/pe"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/mewkiz/pkg/term"
	"github.com/pkg/errors"
)

var (
	// dbg is a logger which logs debug messages with "x:" prefix to standard
	// error.
	dbg = log.New(os.Stderr, term.MagentaBold("x:")+" ", 0)
)

func main() {
	flag.Parse()
	for _, binPath := range flag.Args() {
		if err := lift(binPath); err != nil {
			log.Fatalf("%+v", err)
		}
	}
}

// lift lifts the given binary executable to LLVM IR assembly.
func lift(binPath string) error {
	dbg.Println("lift:", binPath)
	file, err := pe.Open(binPath)
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
		fmt.Printf("=== [ section %q ] ===\n", sect.Name)
		switch {
		case isExec(sect):
			rel := Addr(sect.VirtualAddress)
			addr := base + rel
			if err := liftCode(addr, data); err != nil {
				return errors.WithStack(err)
			}
		}
	}
	return nil
}

// liftCode lifts the code of the given section to LLVM IR.
func liftCode(addr Addr, data []byte) error {
	fmt.Println("addr:", addr)
	fmt.Println(hex.Dump(data))
	return nil
}

// ### [ Helper functions ] ####################################################

// isExec reports whether the given section is executable.
func isExec(sect *pe.Section) bool {
	const codeMask = 0x00000020
	return sect.Characteristics&codeMask != 0
}

// Addr is an address.
type Addr uint32

// String returns the string representation of the address.
func (addr Addr) String() string {
	return fmt.Sprintf("0x%08X", uint32(addr))
}
