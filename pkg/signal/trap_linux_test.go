// +build linux

package signal

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTrap(t *testing.T) {
	sigmap := map[string]os.Signal{
		"TERM": syscall.SIGTERM,
		"QUIT": syscall.SIGQUIT,
		"INT":  os.Interrupt,
	}

	if os.Getenv("TEST_TRAP") == "1" {
		defer time.Sleep(5 * time.Second)

		Trap(func() {
			time.Sleep(1 * time.Second)
			os.Exit(99)
		})

		go func() {
			p, err := os.FindProcess(os.Getpid())
			if err != nil {
				t.Fatalf("unable to get current process id: %v", err)
			}

			switch s := os.Getenv("SIGNAL_TYPE"); s {
			case "TERM":
				for {
					p.Signal(sigmap[s])
				}
			case "QUIT":
				p.Signal(sigmap[s])
			case "INT":
				p.Signal(sigmap[s])
			}
		}()
		select {}
	}

	for k, v := range sigmap {
		cmd := exec.Command(os.Args[0], "-test.run=TestTrap")
		cmd.Env = append(os.Environ(), "TEST_TRAP=1", fmt.Sprintf("SIGNAL_TYPE=%s", k))

		err := cmd.Start()
		if err != nil {
			t.Fatalf("unable to start command: %v", err)
		}

		err = cmd.Wait()
		if e, ok := err.(*exec.ExitError); ok {
			code := e.Sys().(syscall.WaitStatus).ExitStatus()

			switch k {
			case "TERM", "QUIT":
				if code != (128 + int(v.(syscall.Signal))) {
					t.Errorf("exit code (%d) is not as expected (%d)", code, (128 + int(v.(syscall.Signal))))
				}
			case "INT":
				if code != 99 {
					t.Error("cleanup function didn't get executed")
				}
			}
			continue
		}

		t.Fatal("process didn't end with any error")
	}
}

func TestDumpStacks(t *testing.T) {
	directory, errorDir := ioutil.TempDir("", "test")
	assert.NoError(t, errorDir)
	defer os.RemoveAll(directory)

	_, error := DumpStacks(directory)
	path := filepath.Join(directory, fmt.Sprintf(stacksLogNameTemplate, strings.Replace(time.Now().Format(time.RFC3339), ":", "", -1)))
	readFile, _ := ioutil.ReadFile(path)
	fileData := string(readFile)
	assert.NotEqual(t, fileData, "")
	assert.NoError(t, error)
	path, errorPath := DumpStacks("")
	assert.NoError(t, errorPath)
	file := os.Stderr
	assert.EqualValues(t, file.Name(), path)
}
