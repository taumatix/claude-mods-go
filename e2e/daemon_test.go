package e2e

import (
	"bufio"
	"bytes"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestDaemonProcessExitsWhenIdle runs the built mod binary as the shim would,
// talks to it over its real socket, and checks it leaves on its own. It needs
// no claude: the leak it guards against is a daemon outliving its session.
func TestDaemonProcessExitsWhenIdle(t *testing.T) {
	bin := filepath.Join(t.TempDir(), exe("probe"))
	run(t, "", "go", "build", "-o", bin, "./probe")

	cmd := exec.Command(bin, "serve")
	cmd.Env = append(cmd.Environ(), "MODGO_TOKEN=secret")
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	t.Cleanup(func() { _ = cmd.Process.Kill() })

	line, err := bufio.NewReader(out).ReadString('\n')
	if err != nil {
		t.Fatalf("no LISTEN line: %v\n%s", err, stderr.String())
	}
	addr := strings.TrimSpace(strings.TrimPrefix(line, "LISTEN "))

	res, err := http.Post("http://"+addr+"/start", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated /start = %d, want 401", res.StatusCode)
	}

	select {
	case err := <-exited:
		if err != nil {
			t.Fatalf("daemon exited with %v\n%s", err, stderr.String())
		}
	case <-time.After(15 * time.Second):
		t.Fatal("daemon still running 15s after its last request (probe idle timeout is 1s)")
	}
	if _, err := http.Post("http://"+addr+"/start", "application/json", nil); err == nil {
		t.Fatal("port still open after the daemon exited")
	}
}

// TestDaemonReloadsAfterItVanishes has the daemon that handled session.start
// die when the command reaches it, and expects the shim to start a new daemon
// and still answer.
func TestDaemonReloadsAfterItVanishes(t *testing.T) {
	bin := claudeBin(t)
	plugin := buildPlugin(t, "./reload", "daemon")
	out := claude(t, bin, plugin, "/pid")
	if !strings.Contains(out, "kept-memory=false") {
		t.Fatalf("/pid output = %q, want an answer from a fresh daemon", out)
	}
}
