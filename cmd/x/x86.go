package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"sort"

	"github.com/pkg/errors"
	"golang.org/x/arch/x86/x86asm"
)

// Processor mode (16, 32 or 64-bit execution mode).
const cpuMode = addrSize

// Function is a function consisting of one or more basic blocks.
type Function struct {
	// Address of entry basic block.
	entry Addr
	// Map from basic block address to basic block, containing one or more basic
	// blocks.
	blocks map[Addr]*BasicBlock
}

// newFunc returns a new function.
func newFunc(entry Addr) *Function {
	return &Function{
		entry:  entry,
		blocks: make(map[Addr]*BasicBlock),
	}
}

// String returns the string representation of the function.
func (f *Function) String() string {
	buf := &bytes.Buffer{}
	fmt.Fprintf(buf, "func_%08X() {\n", uint32(f.entry))
	var keys Addrs
	for key := range f.blocks {
		keys = append(keys, key)
	}
	sort.Sort(keys)
	for i, key := range keys {
		block := f.blocks[key]
		if i != 0 {
			buf.WriteString("\n")
		}
		fmt.Fprintf(buf, "%v\n", block)
	}
	buf.WriteString("}")
	return buf.String()
}

// BasicBlock is a basic block; a sequence of non-branching instructions
// terminated by an explicit or implicit (fake) control flow instruction.
type BasicBlock struct {
	// One or more instructions.
	insts []*Instruction
}

// String returns the string representation of the basic block.
func (block *BasicBlock) String() string {
	buf := &bytes.Buffer{}
	fmt.Fprintf(buf, "block_%08X:\n", uint32(block.Entry()))
	for i, inst := range block.insts {
		if i != 0 {
			buf.WriteString("\n")
		}
		fmt.Fprintf(buf, "\t%v", inst)
	}
	return buf.String()
}

// Entry returns the entry address of the basic block.
func (block *BasicBlock) Entry() Addr {
	return block.insts[0].addr
}

// Instruction is an x86 instruction.
type Instruction struct {
	// Address of instruction.
	addr Addr
	// Instruction.
	x86asm.Inst
}

// liftCode lifts the code of the given section to LLVM IR.
func (l *lifter) liftCode(start Addr, data []byte) error {
	blocks, err := l.decodeBlocks(start, data)
	if err != nil {
		return errors.WithStack(err)
	}
	funcs, err := l.decodeFuncs(blocks)
	if err != nil {
		return errors.WithStack(err)
	}
	llFuncs, err := l.translateFuncs(funcs)
	if err != nil {
		return errors.WithStack(err)
	}
	_ = llFuncs
	return nil
}

// decodeFuncs decodes the x86 functions based on the given basic blocks.
func (l *lifter) decodeFuncs(blocks []*BasicBlock) ([]*Function, error) {
	dbg.Println("decodeFuncs(blocks)")
	j := 0
	var funcs []*Function
	funcFromAddr := make(map[Addr]*Function)
	// Add continuous basic blocks.
	for i, funcAddr := range l.funcAddrs {
		start := funcAddr
		end := Addr(math.MaxUint32)
		if i+1 < len(l.funcAddrs) {
			end = l.funcAddrs[i+1]
		}
		f := newFunc(funcAddr)
		for _, block := range blocks[j:] {
			blockAddr := block.Entry()
			if blockAddr >= end {
				break
			}
			if blockAddr < start {
				return nil, errors.Errorf("unable to locate function containing basic block; expected address >= %v, got %v", start, blockAddr)
			}
			f.blocks[blockAddr] = block
			j++
		}
		funcs = append(funcs, f)
		funcFromAddr[f.entry] = f
	}
	// Add non-continuous basic blocks.
	if len(l.chunks) > 0 {
		blockFromAddr := make(map[Addr]*BasicBlock)
		for _, block := range blocks {
			blockFromAddr[block.Entry()] = block
		}
		for blockAddr, chunk := range l.chunks {
			block, ok := blockFromAddr[blockAddr]
			if !ok {
				return nil, errors.Errorf("unable to locate basic block at %v", blockAddr)
			}
			for funcAddr := range chunk {
				dbg.Printf("   add basic block %v to non-continuous function %v", blockAddr, funcAddr)
				f, ok := funcFromAddr[funcAddr]
				if !ok {
					return nil, errors.Errorf("unable to locate function at %v", funcAddr)
				}
				f.blocks[blockAddr] = block
			}
		}
	}
	//for _, f := range funcs {
	//	dbg.Println(f)
	//}
	return funcs, nil
}

// decodeBlocks decodes the x86 basic blocks of the given section.
func (l *lifter) decodeBlocks(start Addr, data []byte) ([]*BasicBlock, error) {
	var blocks []*BasicBlock
	//dbg.Printf("decodeBlocks(start = %v, data)", start)
	for j, blockAddr := range l.blockAddrs {
		//dbg.Printf("   block_%08X:", uint32(blockAddr))
		block := &BasicBlock{}
		instAddr := blockAddr
		for {
			offset := int(instAddr - start)
			inst, err := l.decodeInst(instAddr, data[offset:])
			if err != nil {
				return nil, errors.WithStack(err)
			}
			instAddr += Addr(inst.Len)
			//dbg.Println("      addr:", inst.addr)
			//dbg.Println("      inst:", inst)
			block.insts = append(block.insts, inst)
			if isTerm(inst) || (j+1 < len(l.blockAddrs) && instAddr >= l.blockAddrs[j+1]) {
				break
			}
		}
		blocks = append(blocks, block)
	}
	return blocks, nil
}

// decodeInst decodes the leading bytes in src as a single x86 instruction, and
// annotates the instruction with the given address.
func (l *lifter) decodeInst(instAddr Addr, src []byte) (*Instruction, error) {
	inst, err := x86asm.Decode(src, cpuMode)
	if err != nil {
		end := 16
		if end > len(src) {
			end = len(src)
		}
		fmt.Fprintln(os.Stderr, hex.Dump(src[:end]))
		return nil, errors.Errorf("unable to parse instruction at address %v; %v", instAddr, err)
	}
	return &Instruction{
		addr: instAddr,
		Inst: inst,
	}, nil
}

// ### [ Helper functions ] ####################################################

// isTerm reports whether the given instruction is a terminator instruction.
func isTerm(inst *Instruction) bool {
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
