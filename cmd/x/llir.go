package main

import (
	"fmt"
	"sort"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/types"
	"github.com/pkg/errors"
)

// translateFuncs translates the given x86 functions to equivalent LLVM IR
// functions.
func (l *lifter) translateFuncs(funcs []*Function) ([]*ir.Function, error) {
	var llFuncs []*ir.Function
	for _, f := range funcs {
		llFunc, err := l.translateFunc(f)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		llFuncs = append(llFuncs, llFunc)
	}
	return llFuncs, nil
}

// translateFunc translates the given x86 function to an equivalent LLVM IR
// function.
func (l *lifter) translateFunc(f *Function) (*ir.Function, error) {
	// TODO: handle function signatures.
	llFunc := ir.NewFunction("", types.Void)
	var keys Addrs
	for key := range f.blocks {
		keys = append(keys, key)
	}
	sort.Sort(keys)
	for _, key := range keys {
		block := f.blocks[key]
		llBlock, err := l.translateBlock(block)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		llFunc.Blocks = append(llFunc.Blocks, llBlock)
	}
	return llFunc, nil
}

// translateBlock translates the given x86 basic block to an equivalent LLVM IR
// basic block.
func (l *lifter) translateBlock(block *BasicBlock) (*ir.BasicBlock, error) {
	blockName := fmt.Sprintf("block_%08X", uint32(block.Entry()))
	llBlock := ir.NewBlock(blockName)
	for _, inst := range block.insts {
		llInst, err := l.translateInst(inst)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		// TODO: handle terminators.
		llBlock.Insts = append(llBlock.Insts, llInst)
	}
	return llBlock, nil
}

// translateInst translates the given x86 instruction to an equivalent LLVM IR
// instruction.
func (l *lifter) translateInst(inst *Instruction) (ir.Instruction, error) {
	switch inst.Op {
	default:
		panic(fmt.Errorf("support for instruction %v not yet implemented", inst.Op))
	}
}
