// Packet vlog provides package level verbose logging.
// The actual logging is done with the standard log package.
//
// To define a package level logging variable,
//   var v = vlog.New()
//
// To log a message at INFO level,
//   v.I("a")
//
// The logging level can be set with either the flag -vlog or
// the environment variable GO_VLOG.
//
// The -vlog or GO_VLOG format is,
//  k=v(,k=v)*
//  k can be exact match like "foo/bar" or prefix match like "foo/*".
//  v can be w|i|v1|v2
// Default level can be set with prefix match "*".
package vlog

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
)

//go:generate stringer -type=Level
type Level int32

const (
	v2 Level = -2 + iota
	v1
	info
	err
)

// E logs error message.
// If args[0] is a format string, args is formatted with Printf,
// otherwise args is formatted with Println.
func (v *Level) E(args ...interface{}) {
	if *v <= err {
		lg.Log("E " + Format(args...))
	}
}

func E(args ...interface{}) {
	levelVars[0].Level.E(args...)
}

// I logs info message.
func (v *Level) I(args ...interface{}) {
	if *v <= info {
		lg.Log(Format(args...))
	}
}

func I(args ...interface{}) {
	levelVars[0].Level.I(args...)
}

// V1 logs verbose level 1 message.
func (v *Level) V1(args ...interface{}) {
	if *v <= v1 {
		lg.Log(Format(args...))
	}
}

func V1(args ...interface{}) {
	levelVars[0].Level.V1(args...)
}

// V2 logs verbose level 2 message.
func (v *Level) V2(args ...interface{}) {
	if *v <= v2 {
		lg.Log(Format(args...))
	}
}

func V2(args ...interface{}) {
	levelVars[0].Level.V2(args...)
}

// Vstack logs the message and the stacktrace of this goroutine.
// It is noop when verbose logging is not enabled.
func (v *Level) Vstack(args ...interface{}) {
	if *v >= info {
		return
	}
	s := Format(args...)
	lg.Log(stackTrace(s))
}

func Vstack(args ...interface{}) {
	levelVars[0].Level.Vstack(args...)
}

// On returns true if the specific verbose level 1-3 is enabled.
func (v *Level) On(l int) bool {
	lv := Level(-l)
	return *v <= lv
}

func On(l int) bool {
	return levelVars[0].Level.On(l)
}

// Vset sets the verbose logging level.
func (v *Level) Vset(l int) Level {
	lv := Level(-l)
	if lv < v2 || lv >= info {
		lg.Log(Format("invalid verbose level=%d", l))
		return *v
	}
	old := *v
	atomic.StoreInt32((*int32)(v), int32(lv))
	return old
}

func Vset(l int) Level {
	return levelVars[0].Level.Vset(l)
}

// Error returns an error. The message of the error is formatted
// from args. If verbose level 1 is enabled, the error messag includes
// the caller, and if verbose level 2 is enabled, the error message
// includes the call stack.
func (v *Level) Error(args ...interface{}) error {
	return v.newError(Format(args...))
}

func Error(args ...interface{}) error {
	return levelVars[0].Level.Error(args...)
}

// newError is necessary to get the correct call stack
func (v *Level) newError(s string) error {
	switch *v {
	default:
		return errors.New(s)

	case v1:
		_, fn, ln, ok := runtime.Caller(2)
		if !ok {
			return errors.New("???: " + s)
		}
		return errors.New(fn + ":" + strconv.Itoa(ln) + " " + s)

	case v2:
		return errors.New(stackTrace(s))
	}
}

func parseLevel(lvs string) Level {
	switch strings.ToLower(lvs) {
	case "2", "v2":
		return v2
	case "1", "v1":
		return v1
	case "i", "info":
		return info
	case "e", "err":
		return err
	default:
		lg.Log(Format("ignore invalid logging level=%s", lvs))
		return info
	}
}

// New returns a vlog Level variable.
// The name of the Level variable is inferred with these rules,
//  - file name of the caller must be under "/src/", to follow go path convention
//  - if the file is ".../src/<foo_pkg>/{main,cmd}/bar.go", name is "foo_pkg/bar".
//  - if the file is ".../src/<foo_pkg>/bar.go", name is "foo_pkg".
//
// New must be called before Parse() is callled.
func New() *Level {
	// Note: 1 to skip New
	_, fn, _, ok := runtime.Caller(1)
	if !ok {
		lg.Log("fail to get file from runtime.caller")
		return &levelVars[0].Level // [0] is default
	}
	name := inferName(fn)
	return newVar(name, fn)
}

func newVar(name, fn string) *Level {
	if name == "" {
		lg.Log(Format("fail to infer name from file=%s", fn))
		return &levelVars[0].Level // [0] is default
	}
	for _, lv := range levelVars {
		if lv.Name == name {
			lg.Log(Format("dup level name=%s inferred from file=%s", name, fn))
			return &lv.Level
		}
	}
	lv := &levelVar{
		Name: name,
		File: fn,
	}
	levelVars = append(levelVars, lv)
	return &lv.Level
}

func inferName(fn string) string {
	fn = strings.ToLower(fn)
	i := strings.LastIndex(fn, "/src/")
	if i < 0 {
		return ""
	}
	dn, fn := path.Split(fn[i+len("/src/"):])
	if !strings.HasSuffix(fn, ".go") {
		return ""
	}
	for _, p := range []string{"/main/", "/cmd/"} {
		if strings.HasSuffix(dn, p) {
			dn = dn[:len(dn)-len(p)]
			fn = fn[:len(fn)-len(".go")]
			return path.Join(dn, fn)
		}
	}
	return strings.TrimRight(dn, "/")
}

type levelVar struct {
	Name  string
	File  string
	Level Level
}

func (lv *levelVar) String() string {
	return fmt.Sprintf("%s@%s", lv.Name, lv.File)
}

var levelVars = []*levelVar{&levelVar{}} // default level

func Parse() {
	flag.Parse()
	setLevels(*vlogFlag)
	if *vlogHelp {
		lg.Log("vlog setting:" + printLevelVars())
		flag.Usage()
		os.Exit(2)
	}
	if *vlogFile != "" {
		lg = newRotateLogger(*vlogFile)
	}
}

func ParseEnv() {
	if val := os.Getenv("GO_VLOG"); val != "" { // for testing
		setLevels(val)
		lg.Log(printLevelVars())
	}
}

// TODO: set level from a string at runtime
func setLevels(value string) {
	exact, prefix := parseFlag(value)
	if v, ok := prefix["/"]; ok {
		levelVars[0].Level = v // default level
	}
	def := levelVars[0].Level
	for _, lv := range levelVars[1:] {
		lv.Level = def
	}
	if len(prefix) > 0 {
		prefixes := make([]string, 0, len(prefix))
		for k := range prefix {
			prefixes = append(prefixes, k)
		}
		sort.Strings(prefixes)

		for _, lv := range levelVars[1:] {
			for i := len(prefixes) - 1; i >= 0; i-- {
				k := prefixes[i]
				// Match "foo" with "foo/" and "foo/bar" with "foo/"
				if lv.Name == k[:len(k)-1] || strings.HasPrefix(lv.Name, k) {
					lv.Level = prefix[k]
					break
				}
			}
		}
	}
	for _, lv := range levelVars[1:] {
		if i, ok := exact[lv.Name]; ok {
			lv.Level = i
		}
	}
}

func parseFlag(value string) (exact, prefix map[string]Level) {
	exact = make(map[string]Level)
	prefix = make(map[string]Level)
	s := value
	for s != "" {
		k := s
		if i := strings.Index(s, ","); i >= 0 {
			k, s = s[:i], s[i+1:]
		} else {
			s = ""
		}
		j := strings.Index(k, "=")
		if j < 0 {
			panic(Format("malformed: no level", value))
		}
		k, v := k[:j], k[j+1:]
		lv := parseLevel(v)
		k = strings.ToLower(k)
		pre := false
		if k == "*" || strings.HasSuffix(k, "/*") {
			k = k[:len(k)-1]
			if strings.Contains(k, "*") {
				panic(Format("malformed: multiple star", s))
			}
			pre = true
		} else if strings.Contains(k, "*") {
			panic(Format("malformed: star in middle", s))
		}
		k = strings.TrimRight(k, "/")
		if pre {
			prefix[k+"/"] = lv
		} else {
			exact[k] = lv
		}
	}
	return exact, prefix
}

func printLevelVars() string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "*=%v", levelVars[0].Level)
	for _, lv := range levelVars[1:] {
		fmt.Fprintf(&b, ",%s=%v", lv.Name, lv.Level)
	}
	return b.String()
}

var (
	vlogFlag = flag.String("vlog", "", "vlog settings, k=v(,k=v)*")
	vlogFile = flag.String("vlogfile", "", "vlog file prefix")
	vlogHelp = flag.Bool("vloghelp", false, "show vlog setting and flag help")
)

type Logger interface {
	Log(s string)
	Flush()
}

type stderrLogger struct {
	lg *log.Logger
}

func (l *stderrLogger) Log(s string) {
	l.lg.Output(3, s)
}

func (l *stderrLogger) Flush() {}

const logPrefix = log.Ldate | log.Lmicroseconds | log.Lshortfile

// lg should always be available
var lg Logger = &stderrLogger{lg: log.New(os.Stderr, "", logPrefix)}

var stackTraceBegin = []byte("/vlog.go:")

func stackTrace(s string) string {
	var buf [4 << 10]byte
	m := copy(buf[:], s)
	n := runtime.Stack(buf[m:], false)
	n += m

	// Trim the frames in vlog.go, any line that contains "/vlog.go:"
	b := buf[m:n]
	j := bytes.LastIndex(b, stackTraceBegin)
	if j < 0 {
		return string(buf[:n])
	}
	b = b[j:]
	j = bytes.IndexAny(b, "\n")
	if j < 0 {
		return string(buf[:n])
	}
	b = b[j+1:]
	buf[m] = '\n' // put a newline between s and stack frames
	n = copy(buf[m+1:], b)
	n += m + 1
	if buf[n] != '\n' {
		buf[n] = '\n' // always end with newline
	}
	return string(buf[:n])
}
