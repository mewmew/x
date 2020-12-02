package main

import (
	"fmt"
	"sort"

	"github.com/kr/pretty"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
	"github.com/pkg/errors"
	"golang.org/x/arch/x86/x86asm"
)

// translate translates the given x86 binary executable into an equivalent LLVM
// IR module.
func (l *lifter) translate() (*ir.Module, error) {
	// Index functions.
	l.indexFuncs()

	// Create LLVM IR module.
	// TODO: move ir.NewModule to newLifter and move m to a lifter field?
	m := ir.NewModule()
	return m, nil
}

// indexFunc indexes the LLVM IR function definitions based on function address.
func (l *lifter) indexFuncs() {
	// TODO: handle function signatures.
	for _, asmFunc := range l.asmFuncs {
		funcName := fmt.Sprintf("func_%08X", uint32(asmFunc.entry))
		f := ir.NewFunction(funcName, types.Void)
		l.funcs[asmFunc.entry] = f
	}
}

// liftFuncs lifts the given x86 functions to equivalent LLVM IR functions.
func (l *lifter) liftFuncs() error {
	for _, asmFunc := range l.asmFuncs {
		f, ok := l.funcs[asmFunc.entry]
		if !ok {
			return errors.Errorf("unable to locate function at %v", asmFunc.entry)
		}
		fl := newFuncLifter(l, f)
		if err := fl.liftFunc(asmFunc); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

// liftFunc lifts the given x86 function to an equivalent LLVM IR function.
func (fl *funcLifter) liftFunc(asmFunc *Function) error {
	var keys Addrs
	for key := range asmFunc.blocks {
		keys = append(keys, key)
	}
	sort.Sort(keys)
	for _, key := range keys {
		asmBlock := asmFunc.blocks[key]
		if err := fl.liftBlock(asmBlock); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

// liftBlock lifts the given x86 basic block to an equivalent LLVM IR basic
// block.
func (l *lifter) liftBlock(f *ir.Function, asmBlock *BasicBlock) error {
	blockName := fmt.Sprintf("block_%08X", uint32(asmBlock.Entry()))
	llBlock := ir.NewBlock(blockName)
	for _, asmInst := range asmBlock.insts {
		llInst, err := l.translateInst(asmInst)
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
	pretty.Println("inst:", inst)
	switch inst.Op {
	case x86asm.JMP:
		return l.translateInstJMP(inst)
	default:
		panic(fmt.Errorf("support for instruction %v not yet implemented; unable to translate instruction at %v", inst.Op, inst.addr))
	}
}

// translateInstJMP translates the given x86 JMP instruction to an equivalent
// LLVM IR instruction.
func (l *lifter) translateInstJMP(inst *Instruction) (ir.Instruction, error) {
	arg, err := l.translateArg(inst, inst.Args[0])
	if err != nil {
		return nil, errors.WithStack(err)
	}
	pretty.Println("arg:", arg)
	return nil, nil
}

// translateArg translates the given x86 instruction argument to an equivalent
// LLVM IR value.
func (l *lifter) translateArg(inst *Instruction, arg x86asm.Arg) (value.Value, error) {
	switch arg := arg.(type) {
	case x86asm.Rel:
		relAddr := int64(inst.addr) + int64(inst.Len) + int64(arg)
		return constant.NewInt(types.I32, relAddr), nil
	default:
		panic(fmt.Errorf("support for instruction argument %T not yet implemented; unable to translate argument used in instruction at %v", arg, inst.addr))
	}
}
