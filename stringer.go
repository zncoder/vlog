package vlog

import (
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode"
)

// Format formats args.
// If args[0] is a format string, Printf is used;
// otherwise Println is used.
func Format(args ...interface{}) string {
	if len(args) == 0 {
		return ""
	}
	sfmt, ok := args[0].(string)
	if ok && len(args) == 1 {
		return sfmt
	}
	if !ok || strings.Index(sfmt, "%") == -1 {
		s := fmt.Sprintln(args...)
		return s[:len(s)-1] // trim ending newline
	}
	return fmt.Sprintf(sfmt, args[1:]...)
}

// Print formats args and prints to Stdout.
func Print(args ...interface{}) {
	s := Format(args...)
	fmt.Println(s)
}

// Fprint formats args and prints to w.
func Fprint(w io.Writer, args ...interface{}) {
	s := Format(args...)
	fmt.Fprintln(w, s)
}

// Panic formats args and panic.
func Panic(args ...interface{}) {
	lg.Log(Format(args...))
	panic("panic")
}

// Fatal formats args and panic.
func Fatal(args ...interface{}) {
	lg.Log(Format(args...))
	os.Exit(1)
}

// CheckOK checks c is true. If c is false, CheckOK formats args and panic.
func CheckOK(c bool, args ...interface{}) {
	if c {
		return
	}
	lg.Log(Format(args...))
	panic("CHECK failure")
}

// Check checks err is nil. If err is not nil, Check formats args and panic.
func Check(err error, args ...interface{}) {
	if err == nil {
		return
	}
	lg.Log(Format(args...))
	panic("CHECK error:" + err.Error())
}

// CheckFlag checks c is true.
// If c is false, CheckFlag formats args, prints flag usage and exit.
func CheckFlag(c bool, args ...interface{}) {
	if c {
		return
	}
	lg.Log(Format(args...))
	flag.Usage()
	os.Exit(2)
}

// Must checks err is nil and returns result.
// If err is not nil, Must formats args and panic.
//
// An one-liner to open a file can be,
// file := Must(os.Open(filename)).(*os.File)
func Must(result interface{}, err error, args ...interface{}) interface{} {
	if err == nil {
		return result
	}
	lg.Log(Format(args...))
	panic("CHECK error:" + err.Error())
}

// Stringer delays converting arg to string.
// Stringer pretty print certain type of arg,
// - if arg is []byte and every byte is printable,
//   it is converted to string. Otherwise it is formatted as hex string.
// - if arg is Time, it is formatted with "20060102-15:04:05"
// - if arg is "func() string", it is called.
// - otherwise arg is formatted with Format
func Stringer(arg interface{}) stringer {
	return stringer{arg: arg}
}

// stringer is necessary because an interface type cannot be receiver
type stringer struct {
	arg interface{}
}

func (sr stringer) String() string {
	switch x := sr.arg.(type) {
	case []byte:
		for _, c := range x {
			if !unicode.IsPrint(rune(c)) {
				return hex.EncodeToString(x)
			}
		}
		return string(x)

	case time.Time:
		return x.Format("20060102-15:04:05")

	case func() string:
		return x()

	default:
		return Format(x)
	}
}
