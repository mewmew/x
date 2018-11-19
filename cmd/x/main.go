// The x tool lifts binary executables to LLVM IR assembly.
//
// Separation of concern is handled through reliance on oracles, which provide
// addresses of basic block, function calling conventions, type information,
// etc.
package main

import (
	"debug/pe"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/mewkiz/pkg/term"
	"github.com/pkg/errors"
	"golang.org/x/arch/x86/x86asm"
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
	// Basic blocks.
	blocks []*BasicBlock
}

// newLifter returns a new lifter based on the given binary executable path.
func newLifter(binPath string) (*lifter, error) {
	l := &lifter{
		binPath: binPath,
	}
	// Parse function addresses.
	if err := decodeJSON("funcs.json", &l.funcAddrs); err != nil {
		return nil, errors.WithStack(err)
	}
	// Parse basic block addresses.
	if err := decodeJSON("blocks.json", &l.blockAddrs); err != nil {
		return nil, errors.WithStack(err)
	}
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

// BasicBlock is a basic block; a sequence of non-branching instructions
// terminated by an explicit or implicit (fake) control flow instruction.
type BasicBlock struct {
	// Instructions.
	insts []*Inst
}

// Inst is an x86 instruction.
type Inst struct {
	// Address of instruction.
	addr Addr
	// Instruction.
	inst x86asm.Inst
}

// liftCode lifts the code of the given section to LLVM IR.
func (l *lifter) liftCode(start Addr, data []byte) error {
	if err := l.decodeBlocks(start, data); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// decodeBlocks decodes the x86 basic blocks of the given section.
func (l *lifter) decodeBlocks(start Addr, data []byte) error {
	dbg.Printf("decodeBlocks(start = %v, data)\n", start)
	for j, blockAddr := range l.blockAddrs {
		dbg.Printf("   block_%08X:\n", uint32(blockAddr))
		block := &BasicBlock{}
		instAddr := blockAddr
		for {
			offset := int(instAddr - start)
			inst, err := x86asm.Decode(data[offset:], cpuMode)
			if err != nil {
				end := offset + 16
				if end > len(data) {
					end = len(data)
				}
				fmt.Fprintln(os.Stderr, hex.Dump(data[offset:end]))
				return errors.Errorf("unable to parse instruction at address %v; %v", instAddr, err)
			}
			i := &Inst{
				addr: instAddr,
				inst: inst,
			}
			instAddr += Addr(inst.Len)
			dbg.Println("      addr:", i.addr)
			dbg.Println("      inst:", i.inst)
			block.insts = append(block.insts, i)
			if isTerm(i.inst) || (j+1 < len(l.blockAddrs) && instAddr >= l.blockAddrs[j+1]) {
				break
			}
		}
		l.blocks = append(l.blocks, block)
	}
	return nil
}

// ### [ Helper functions ] ####################################################

// isTerm reports whether the given instruction is a terminator instruction.
func isTerm(inst x86asm.Inst) bool {
	switch inst.Op {
	// Loop terminators.
	case x86asm.LOOP, x86asm.LOOPE, x86asm.LOOPNE:
		return true
	// Conditional jump terminators.
	case x86asm.JA, x86asm.JAE, x86asm.JB, x86asm.JBE, x86asm.JCXZ, x86asm.JE, x86asm.JECXZ, x86asm.JG, x86asm.JGE, x86asm.JL, x86asm.JLE, x86asm.JNE, x86asm.JNO, x86asm.JNP, x86asm.JNS, x86asm.JO, x86asm.JP, x86asm.JRCXZ, x86asm.JS:
		return true
	// Unconditional jump terminators.
	case x86asm.JMP:
		return true
	// Return terminators.
	case x86asm.RET:
		return true
	}
	return false
}

// isExec reports whether the given section is executable.
func isExec(sect *pe.Section) bool {
	const codeMask = 0x00000020
	return sect.Characteristics&codeMask != 0
}

// Addr is an address.
type Addr uint32

const (
	// Address size in number of bits.
	addrSize = 32
	// Processor mode (16, 32 or 64-bit execution mode).
	cpuMode = addrSize
)

// String returns the string representation of the address.
func (addr Addr) String() string {
	return fmt.Sprintf("0x%08X", uint32(addr))
}

// UnmarshalJSON unmarshals the given string representation of the address.
func (addr *Addr) UnmarshalJSON(b []byte) error {
	s, err := strconv.Unquote(string(b))
	if err != nil {
		return errors.WithStack(err)
	}
	if !strings.HasPrefix(s, "0x") {
		return errors.Errorf("invalid hex representation %q; missing 0x prefix", s)
	}
	s = s[len("0x"):]
	x, err := strconv.ParseUint(s, 16, addrSize)
	if err != nil {
		return errors.WithStack(err)
	}
	*addr = Addr(x)
	return nil
}

// decodeJSON decodes the given JSON file into v.
func decodeJSON(jsonPath string, v interface{}) error {
	dbg.Printf("decodeJSON(jsonPath = %q)\n", jsonPath)
	f, err := os.Open(jsonPath)
	if err != nil {
		return errors.WithStack(err)
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	if err := dec.Decode(v); err != nil {
		return errors.WithStack(err)
	}
	return nil
}
