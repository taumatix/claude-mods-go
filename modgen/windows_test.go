package modgen

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const harness = `
const { pathToFileURL } = await import('node:url')
const { register } = await import(pathToFileURL(process.argv[2]).href)
const out = []
const handlers = []
register((...a) => handlers.push(a[a.length - 1]), {})
const $ = {
  process: {
    run: async (argv) => {
      out.push(argv)
      if (argv[0] === 'uname') return { exitCode: 0, stdout: process.env.FAKE_UNAME + '\n', stderr: '' }
      if (argv[0] === 'cmd') return { exitCode: 0, stdout: process.env.FAKE_ARCH + '\r\n', stderr: '' }
      return { exitCode: 0, stdout: JSON.stringify({ op: 'done', value: { ok: true } }), stderr: '' }
    },
  },
}
try { await handlers[0]($, {}, async (e) => e) } catch (err) { out.push('error: ' + err.message) }
console.log(JSON.stringify(out))
`

// runShim renders the shim, pretends it was loaded from url, and returns the
// argv of every process.run it made.
func runShim(t *testing.T, targets []Target, url string, env ...string) []any {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not installed")
	}
	_, js := gen(t, Once, targets...)
	js = strings.ReplaceAll(js, "import.meta.url", `"`+url+`"`)
	dir := t.TempDir()
	mod, h := filepath.Join(dir, "register.mjs"), filepath.Join(dir, "h.mjs")
	for p, s := range map[string]string{mod: js, h: harness} {
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command(node, h, mod)
	cmd.Env = append(os.Environ(), env...)
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("node: %v\n%s", err, b)
	}
	var out []any
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(b))), &out); err != nil {
		t.Fatalf("output %q: %v", b, err)
	}
	return out
}

func firstArg(t *testing.T, call any) string {
	t.Helper()
	argv, ok := call.([]any)
	if !ok {
		t.Fatalf("call is %v, not an argv", call)
	}
	return argv[0].(string)
}

func TestSingleWindowsTargetRunsTheExe(t *testing.T) {
	out := runShim(t, []Target{{OS: "windows", Arch: "amd64"}}, "file:///C:/Users/me/.claude/plugins/demo/hooks/register.js")
	if got := firstArg(t, out[0]); got != "C:/Users/me/.claude/plugins/demo/bin/windows-amd64/demo.exe" {
		t.Errorf("binary = %q", got)
	}
}

func TestSpacesInAWindowsPathAreDecoded(t *testing.T) {
	out := runShim(t, []Target{{OS: "windows", Arch: "amd64"}}, "file:///C:/Users/Jane%20Doe/plug/hooks/register.js")
	if got := firstArg(t, out[0]); got != "C:/Users/Jane Doe/plug/bin/windows-amd64/demo.exe" {
		t.Errorf("binary = %q", got)
	}
}

func TestMultiTargetShimPicksWindowsByDriveLetter(t *testing.T) {
	targets := []Target{{OS: "linux", Arch: "amd64"}, {OS: "windows", Arch: "amd64"}, {OS: "windows", Arch: "arm64"}}
	for arch, want := range map[string]string{"AMD64": "windows-amd64", "ARM64": "windows-arm64"} {
		out := runShim(t, targets, "file:///D:/p/hooks/register.js", "FAKE_ARCH="+arch)
		if got := firstArg(t, out[0]); got != "cmd" {
			t.Fatalf("%s: first call %q, want cmd (no uname on Windows)", arch, got)
		}
		if got := firstArg(t, out[1]); got != "D:/p/bin/"+want+"/demo.exe" {
			t.Errorf("%s: binary = %q", arch, got)
		}
	}
}

func TestMultiTargetShimStillUsesUnameOffWindows(t *testing.T) {
	targets := []Target{{OS: "linux", Arch: "amd64"}, {OS: "windows", Arch: "amd64"}}
	out := runShim(t, targets, "file:///home/me/p/hooks/register.js", "FAKE_UNAME=Linux x86_64")
	if got := firstArg(t, out[0]); got != "uname" {
		t.Fatalf("first call %q, want uname", got)
	}
	if got := firstArg(t, out[1]); got != "/home/me/p/bin/linux-amd64/demo" {
		t.Errorf("binary = %q", got)
	}
}

func TestUnbuiltWindowsArchIsNamed(t *testing.T) {
	out := runShim(t, []Target{{OS: "linux", Arch: "amd64"}, {OS: "windows", Arch: "amd64"}}, "file:///C:/p/hooks/register.js", "FAKE_ARCH=ARM64")
	last, _ := out[len(out)-1].(string)
	if !strings.Contains(last, "no build for windows-arm64") {
		t.Errorf("got %v", out)
	}
}

func TestWindowsBinaryIsWrittenWithExe(t *testing.T) {
	dir, _ := gen(t, Once, Target{OS: "windows", Arch: "amd64"}, Target{OS: "linux", Arch: "amd64"})
	for _, f := range []string{"bin/windows-amd64/demo.exe", "bin/linux-amd64/demo"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("%s: %v", f, err)
		}
	}
}
