package vlog

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

type rotateLogger struct {
	mu     sync.Mutex
	lg     *log.Logger
	wr     *bufio.Writer
	f      io.WriteCloser
	nbytes int
	prefix string
	nextID int
}

func newRotateLogger(prefix string) *rotateLogger {
	rl := &rotateLogger{
		prefix: prefix,
	}
	rl.rotate()
	go rl.flushloop()
	return rl
}

var (
	logPrefixSize    = 50
	logLimit         = 1 << 30
	logFlushInterval = 29 * time.Second
)

func (rl *rotateLogger) Log(s string) {
	rl.mu.Lock()
	for {
		checklimit := rl.nbytes > 0
		rl.nbytes += logPrefixSize + len(s)
		if checklimit && rl.nbytes > logLimit {
			rl.rotate()
			continue // retry
		}
		rl.lg.Output(3, s)
		break
	}
	rl.mu.Unlock()
}

func (rl *rotateLogger) Flush() {
	rl.mu.Lock()
	rl.wr.Flush() // ignore error
	rl.mu.Unlock()
}

func (rl *rotateLogger) rotate() {
	if rl.f != nil {
		rl.wr.Flush()
		rl.f.Close()
	}
	t := time.Now()
	fn := fmt.Sprintf("%s.%04d%02d%02d-%02d%02d%02d.%02d.log",
		rl.prefix, t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(),
		rl.nextID)
	f, err := os.Create(fn)
	if err != nil {
		panic(fmt.Sprintf("create log file=%s err=%v", fn, err))
	}
	rl.f = f
	rl.wr = bufio.NewWriter(f)
	rl.lg = log.New(rl.wr, "", logPrefix)
	rl.nbytes = 0
	rl.nextID++
}

func (rl *rotateLogger) flushloop() {
	for _ = range time.Tick(logFlushInterval) {
		rl.Flush()
	}
}
