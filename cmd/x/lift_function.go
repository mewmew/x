package main

import "github.com/llir/llvm/ir"

// funcLifter is a lifter for a given LLVM IR function.
type funcLifter struct {
	// Binary executable lifter.
	l *lifter

	// LLVM IR function being lifted.
	f *ir.Function
	// Current basic block being lifted.
	cur *ir.BasicBlock
}
