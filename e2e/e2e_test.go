// Package e2e drives built mods through a real `claude -p` run. It needs a
// Claude Code with mods (2.1.290+) and skips without one. Set CLAUDE_BIN to
// choose the binary.
package e2e

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func claudeBin(t *testing.T) string {
	t.Helper()
	bin := os.Getenv("CLAUDE_BIN")
	if bin == "" {
		bin = "claude"
	}
	p, err := exec.LookPath(bin)
	if err != nil {
		t.Skip("claude not found; set CLAUDE_BIN")
	}
	if out, err := exec.Command(p, "plugin", "validate", "--help").CombinedOutput(); err != nil {
		t.Skipf("claude has no mods support: %v\n%s", err, out)
	}
	return p
}

func buildPlugin(t *testing.T, pkg, transport string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "plugin")
	modgo := filepath.Join(t.TempDir(), "modgo")
	run(t, "", "go", "build", "-o", modgo, "../cmd/modgo")
	run(t, "", modgo, "build", "-o", dir, "-transport", transport, pkg)
	return dir
}

func run(t *testing.T, dir, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
	return string(out)
}

func claude(t *testing.T, bin, plugin, prompt string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "-p", prompt, "--plugin-dir", plugin)
	cmd.Dir = t.TempDir()
	cmd.Stdin = strings.NewReader("")
	// Claude Code 2.1.283+ loads hooks modules for installed plugins only when this is set.
	cmd.Env = append(os.Environ(), "CLAUDE_CODE_ENABLE_FUNCTION_HOOKS=1")
	out, _ := cmd.CombinedOutput()
	return string(out)
}

func TestMods(t *testing.T) {
	bin := claudeBin(t)
	for _, transport := range []string{"daemon", "once", "auto"} {
		t.Run(transport, func(t *testing.T) {
			plugin := buildPlugin(t, "./probe", transport)

			if out := run(t, "", bin, "plugin", "validate", plugin); !strings.Contains(out, "Validation passed") {
				t.Fatalf("validate: %s", out)
			}

			target := filepath.Join(t.TempDir(), "f.txt")
			out := claude(t, bin, plugin, "/probe "+target)
			if !strings.Contains(out, "read=from go proc=ran") {
				t.Fatalf("/probe output = %q", out)
			}
			if b, err := os.ReadFile(target); err != nil || string(b) != "from go" {
				t.Fatalf("file = %q, %v", b, err)
			}

			out = claude(t, bin, plugin, "/boom")
			if !strings.Contains(out, "fs.read") {
				t.Fatalf("/boom should surface the failed fs.read, got %q", out)
			}
		})
	}
}

func TestTallyExample(t *testing.T) {
	bin := claudeBin(t)
	plugin := buildPlugin(t, "../examples/tally", "auto")
	out := claude(t, bin, plugin, "/tally")
	if !strings.Contains(out, "tool calls") {
		t.Fatalf("/tally output = %q", out)
	}
}
