package vlog

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func cleanUpTmpLogs(t *testing.T, pattern string) {
	olds, _ := filepath.Glob(pattern)
	for _, f := range olds {
		if err := os.Remove(f); err != nil {
			t.Fatalf("failed to remove old log file=%s err=%v", f, err)
		}
	}
}

func listLogFiles(t *testing.T, want int, pattern string) []string {
	fns, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("glob log files pattern=%s err=%v", pattern, err)
	}
	if len(fns) != want {
		t.Fatalf("matches pattern=%s want=%d got=%d", pattern, want, len(fns))
	}
	return fns
}

func TestRotateLogger(t *testing.T) {
	prefix := fmt.Sprintf("/tmp/rotate_log_test.%d", os.Getpid())
	pattern := prefix + "*.log"
	cleanUpTmpLogs(t, pattern)
	defer cleanUpTmpLogs(t, pattern)

	defer func(old int) {
		logLimit = old
	}(logLimit)
	logLimit = 180

	lg = newRotateLogger(prefix)
	data1 := []string{"a", "bc"}
	for _, d := range data1 {
		lg.Log(d)
	}
	listLogFiles(t, 1, pattern)
	data2 := []string{string(make([]byte, 128)), "def"}
	for _, d := range data2 {
		lg.Log(d)
	}
	listLogFiles(t, 3, pattern)
}

func TestRotateLoggerN(t *testing.T) {
	prefix := fmt.Sprintf("/tmp/rotate_log_test.%d", os.Getpid())
	pattern := prefix + "*.log"
	cleanUpTmpLogs(t, pattern)

	rl := newRotateLogger(prefix)
	lg = rl
	ch := make(chan struct{}, 100)
	for i := 0; i < 100; i++ {
		i := i
		go func() {
			for j := 0; j < 50; j++ {
				lg.Log(fmt.Sprintf("goroutine %d log %d", i, j))
			}
			ch <- struct{}{}
		}()
	}
	for i := 0; i < 100; i++ {
		<-ch
	}
	if rl.nextID != 1 {
		t.Errorf("nextid want 1 got %d", rl.nextID)
	}
	listLogFiles(t, 1, pattern)
}
