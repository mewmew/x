package main

import "github.com/llir/llvm/ir"

// translateFuncs translates the given x86 functions to equivalent LLVM IR
// functions.
func (l *lifter) translateFuncs(funcs []*Function) ([]*ir.Function, error) {
	var llFuncs []*ir.Function
	return llFuncs, nil
}
