// Package modgen writes a Claude Code plugin around a Go mod binary: the
// plugin manifest, hooks.json and the generated JavaScript shim.
package modgen

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/taumatix/claude-mods-go/mods"
)

// Target is one built binary: GOOS, GOARCH and where the file is.
type Target struct {
	OS, Arch, Binary string
}

// Options configures Generate.
type Options struct {
	Manifest  mods.Manifest
	Dir       string
	Transport Transport
	Targets   []Target
	Author    string
}

var nameRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]*$`)

// Generate writes the plugin into o.Dir, replacing hooks/, .claude-plugin/ and bin/.
func Generate(o Options) error {
	m := o.Manifest
	if !nameRE.MatchString(m.Name) || len(m.Name) > 64 {
		return fmt.Errorf("modgen: %q is not a valid plugin name (letters, digits, hyphens)", m.Name)
	}
	if o.Transport == "" {
		o.Transport = Auto
	}
	if !o.Transport.Valid() {
		return fmt.Errorf("modgen: unknown transport %q (auto, daemon, once)", o.Transport)
	}
	if len(o.Targets) == 0 {
		return fmt.Errorf("modgen: no binaries to package")
	}
	if o.Author == "" {
		o.Author = "modgo"
	}
	if m.Version == "" {
		m.Version = "0.1.0"
	}
	for _, d := range []string{"hooks", ".claude-plugin", "bin"} {
		if err := os.RemoveAll(filepath.Join(o.Dir, d)); err != nil {
			return err
		}
	}
	plugin := map[string]any{
		"name": m.Name, "version": m.Version, "description": m.Description,
		"author": map[string]string{"name": o.Author},
	}
	if m.Description == "" {
		plugin["description"] = m.Name + " (a Go mod built with modgo)"
	}
	files := map[string][]byte{
		".claude-plugin/plugin.json": mustJSON(plugin),
		"hooks/hooks.json":           mustJSON(map[string]any{"description": m.Name, "modules": []string{"./register.js"}}),
		"hooks/register.js":          []byte(renderShim(m, o.Transport, o.Targets)),
	}
	for rel, b := range files {
		if err := write(filepath.Join(o.Dir, rel), b, 0o644); err != nil {
			return err
		}
	}
	for _, t := range o.Targets {
		b, err := os.ReadFile(t.Binary)
		if err != nil {
			return err
		}
		if err := write(filepath.Join(o.Dir, "bin", t.OS+"-"+t.Arch, m.Name), b, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func write(path string, b []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, b, mode)
}

func mustJSON(v any) []byte {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	return append(b, '\n')
}
