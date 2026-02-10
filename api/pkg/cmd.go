package pkg

import (
	"bytes"
	"os"
	"os/exec"
)

func RunCommandLine(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// RunCommandLineWithOutput runs a command and captures stdout and stderr into buffers.
func RunCommandLineWithOutput(dir string, name string, args ...string) (stdout string, stderr string, err error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}
