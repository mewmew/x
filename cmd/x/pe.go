package main

import "debug/pe"

// ### [ Helper functions ] ####################################################

// isExec reports whether the given section is executable.
func isExec(sect *pe.Section) bool {
	const codeMask = 0x00000020
	return sect.Characteristics&codeMask != 0
}
