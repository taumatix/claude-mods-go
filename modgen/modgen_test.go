package modgen

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/taumatix/claude-mods-go/mods"
)

func gen(t *testing.T, tr Transport, targets ...Target) (string, string) {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "b")
	if err := os.WriteFile(bin, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	for i := range targets {
		targets[i].Binary = bin
	}
	dir := t.TempDir()
	m := mods.Manifest{Name: "demo", Hooks: []mods.Hook{
		{Event: "command.run", Match: map[string]string{"command": "x"}},
		{Event: "tool.call"},
	}, Uses: []mods.Method{mods.FSRead}}
	if err := Generate(Options{Manifest: m, Dir: dir, Transport: tr, Targets: targets}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "hooks/register.js"))
	if err != nil {
		t.Fatal(err)
	}
	return dir, string(b)
}

func TestShimBridgesOnlyDeclaredMethods(t *testing.T) {
	_, js := gen(t, Auto, Target{OS: "linux", Arch: "amd64"})
	if !strings.Contains(js, "$.fs.read(a[0], a[1])") {
		t.Error("declared fs.read missing")
	}
	if strings.Contains(js, "$.fs.write") || strings.Contains(js, "$.ui.toast") {
		t.Error("undeclared method bridged")
	}
	if !strings.Contains(js, `on("command.run", {"command":"x"}, async`) {
		t.Error("matcher not rendered literally")
	}
}

func TestShimNeverPassesDollar(t *testing.T) {
	_, js := gen(t, Auto, Target{OS: "linux", Arch: "amd64"}, Target{OS: "darwin", Arch: "arm64"})
	if bad := regexp.MustCompile(`[(,=]\s*\$\s*[,)]`).FindString(strings.ReplaceAll(js, "async ($, e, next)", "")); bad != "" {
		t.Errorf("$ used as a value: %q", bad)
	}
}

func TestTransportsOnlyEmitWhatTheyNeed(t *testing.T) {
	_, d := gen(t, Daemon, Target{OS: "linux", Arch: "amd64"})
	if strings.Contains(d, "$.process.run") {
		t.Error("daemon shim uses process.run")
	}
	_, o := gen(t, Once, Target{OS: "linux", Arch: "amd64"})
	if strings.Contains(o, "$.process.spawn") || strings.Contains(o, "$.http.fetch") {
		t.Error("once shim spawns or fetches")
	}
}

func TestGenerateLayoutAndValidation(t *testing.T) {
	dir, _ := gen(t, Once, Target{OS: "linux", Arch: "amd64"})
	for _, f := range []string{".claude-plugin/plugin.json", "hooks/hooks.json", "bin/linux-amd64/demo"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Error(err)
		}
	}
	err := Generate(Options{Manifest: mods.Manifest{Name: "bad name"}, Dir: t.TempDir(), Targets: []Target{{}}})
	if err == nil {
		t.Error("invalid name accepted")
	}
}
