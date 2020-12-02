// Package bin provides a uniform representation of binary executables.
package bin

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// Addr is a virtual address that may be specified in hexadecimal notation. It
// implements the flag.Value and encoding.TextUnmarshaler interfaces.
type Addr uint32

// Address size in number of bits.
const addrSize = 32

// String returns the hexadecimal string representation of v.
func (v Addr) String() string {
	return fmt.Sprintf("0x%08X", uint32(v))
}

// Set sets v to the numberic value represented by s.
func (v *Addr) Set(s string) error {
	x, err := parseUint32(s)
	if err != nil {
		return errors.WithStack(err)
	}
	*v = Addr(x)
	return nil
}

// UnmarshalText unmarshals the text into v.
func (v *Addr) UnmarshalText(text []byte) error {
	return v.Set(string(text))
}

// MarshalText returns the textual representation of v.
func (v Addr) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}

// Addrs implements the sort.Sort interface, sorting addresses in ascending
// order.
type Addrs []Addr

func (as Addrs) Len() int           { return len(as) }
func (as Addrs) Swap(i, j int)      { as[i], as[j] = as[j], as[i] }
func (as Addrs) Less(i, j int) bool { return as[i] < as[j] }

// ### [ Helper functions ] ####################################################

// parseUint32 interprets the given string in base 10 or base 16 (if prefixed
// with `0x` or `0X`) and returns the corresponding value.
func parseUint32(s string) (uint32, error) {
	base := 10
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		s = s[len("0x"):]
		base = 16
	}
	x, err := strconv.ParseUint(s, base, 32)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	return uint32(x), nil
}
