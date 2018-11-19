package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// Addr is an address.
type Addr uint32

// Address size in number of bits.
const addrSize = 32

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
