package main

import (
	"bufio"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestExec(t *testing.T) {
	runCmd := exec.Command(dockerBinary, "run", "-d", "--name", "testing", "busybox", "sh", "-c", "echo test > /tmp/file && sleep 100")
	out, _, _, err := runCommandWithStdoutStderr(runCmd)
	errorOut(err, t, out)

	execCmd := exec.Command(dockerBinary, "exec", "testing", "cat", "/tmp/file")

	out, _, err = runCommandWithOutput(execCmd)
	errorOut(err, t, out)

	out = strings.Trim(out, "\r\n")

	if expected := "test"; out != expected {
		t.Errorf("container exec should've printed %q but printed %q", expected, out)
	}

	deleteAllContainers()

	logDone("exec - basic test")
}

func TestExecInteractive(t *testing.T) {
	runCmd := exec.Command(dockerBinary, "run", "-d", "--name", "testing", "busybox", "sh", "-c", "echo test > /tmp/file && sleep 100")
	out, _, _, err := runCommandWithStdoutStderr(runCmd)
	errorOut(err, t, out)

	execCmd := exec.Command(dockerBinary, "exec", "-i", "testing", "sh")
	stdin, err := execCmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := execCmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}

	if err := execCmd.Start(); err != nil {
		t.Fatal(err)
	}
	if _, err := stdin.Write([]byte("cat /tmp/file\n")); err != nil {
		t.Fatal(err)
	}

	r := bufio.NewReader(stdout)
	line, err := r.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	line = strings.TrimSpace(line)
	if line != "test" {
		t.Fatalf("Output should be 'test', got '%q'", line)
	}
	if err := stdin.Close(); err != nil {
		t.Fatal(err)
	}
	finish := make(chan struct{})
	go func() {
		if err := execCmd.Wait(); err != nil {
			t.Fatal(err)
		}
		close(finish)
	}()
	select {
	case <-finish:
	case <-time.After(1 * time.Second):
		t.Fatal("docker exec failed to exit on stdin close")
	}

	deleteAllContainers()

	logDone("exec - Interactive test")
}

func TestExecAfterContainerRestart(t *testing.T) {
	runCmd := exec.Command(dockerBinary, "run", "-d", "busybox", "top")
	out, _, err := runCommandWithOutput(runCmd)
	errorOut(err, t, out)

	cleanedContainerID := stripTrailingCharacters(out)

	runCmd = exec.Command(dockerBinary, "restart", cleanedContainerID)
	out, _, err = runCommandWithOutput(runCmd)
	errorOut(err, t, out)

	runCmd = exec.Command(dockerBinary, "exec", cleanedContainerID, "echo", "hello")
	out, _, err = runCommandWithOutput(runCmd)
	errorOut(err, t, out)

	outStr := strings.TrimSpace(out)
	if outStr != "hello" {
		t.Errorf("container should've printed hello, instead printed %q", outStr)
	}

	deleteAllContainers()

	logDone("exec - exec running container after container restart")
}

func TestExecAfterDaemonRestart(t *testing.T) {
	d := NewDaemon(t)
	if err := d.StartWithBusybox(); err != nil {
		t.Fatalf("Could not start daemon with busybox: %v", err)
	}
	defer d.Stop()

	if out, err := d.Cmd("run", "-d", "--name", "top", "-p", "80", "busybox:latest", "top"); err != nil {
		t.Fatalf("Could not run top: err=%v\n%s", err, out)
	}

	if err := d.Restart(); err != nil {
		t.Fatalf("Could not restart daemon: %v", err)
	}

	if out, err := d.Cmd("start", "top"); err != nil {
		t.Fatalf("Could not start top after daemon restart: err=%v\n%s", err, out)
	}

	out, err := d.Cmd("exec", "top", "echo", "hello")
	if err != nil {
		t.Fatalf("Could not exec on container top: err=%v\n%s", err, out)
	}

	outStr := strings.TrimSpace(string(out))
	if outStr != "hello" {
		t.Errorf("container should've printed hello, instead printed %q", outStr)
	}

	logDone("exec - exec running container after daemon restart")
}
