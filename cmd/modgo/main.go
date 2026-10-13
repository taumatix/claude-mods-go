// Command modgo builds a Go program written with package mods into a Claude
// Code plugin.
//
//	modgo build [-o dir] [-transport auto|daemon|once] [-targets os/arch,...] [package]
//	modgo validate [dir]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/taumatix/claude-mods-go/modgen"
	"github.com/taumatix/claude-mods-go/mods"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	var err error
	switch os.Args[1] {
	case "build":
		err = build(os.Args[2:])
	case "validate":
		err = validate(os.Args[2:])
	default:
		usage()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "modgo:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: modgo build [-o dir] [-transport auto|daemon|once] [-targets os/arch,...] [package]\n       modgo validate [dir]")
	os.Exit(2)
}

func build(argv []string) error {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	out := fs.String("o", "plugin", "output plugin directory")
	tr := fs.String("transport", "auto", "auto, daemon or once")
	targets := fs.String("targets", runtime.GOOS+"/"+runtime.GOARCH, "comma-separated os/arch list")
	author := fs.String("author", "", "plugin author name")
	_ = fs.Parse(argv)
	pkg := "."
	if fs.NArg() > 0 {
		pkg = fs.Arg(0)
	}

	tmp, err := os.MkdirTemp("", "modgo-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	host := filepath.Join(tmp, "host")
	if err := goBuild(pkg, host, runtime.GOOS, runtime.GOARCH); err != nil {
		return err
	}
	raw, err := exec.Command(host, "manifest").Output()
	if err != nil {
		return fmt.Errorf("reading the manifest: %w", err)
	}
	var m mods.Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return err
	}

	var ts []modgen.Target
	for _, t := range strings.Split(*targets, ",") {
		osName, arch, ok := strings.Cut(strings.TrimSpace(t), "/")
		if !ok {
			return fmt.Errorf("target %q: want os/arch", t)
		}
		bin := filepath.Join(tmp, osName+"-"+arch)
		if err := goBuild(pkg, bin, osName, arch); err != nil {
			return err
		}
		ts = append(ts, modgen.Target{OS: osName, Arch: arch, Binary: bin})
	}
	if err := modgen.Generate(modgen.Options{Manifest: m, Dir: *out, Transport: modgen.Transport(*tr), Targets: ts, Author: *author}); err != nil {
		return err
	}
	fmt.Printf("modgo: wrote plugin %s to %s\n", m.Name, *out)
	return nil
}

func goBuild(pkg, out, goos, goarch string) error {
	cmd := exec.Command("go", "build", "-o", out, pkg)
	cmd.Env = append(os.Environ(), "GOOS="+goos, "GOARCH="+goarch, "CGO_ENABLED=0")
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func validate(argv []string) error {
	dir := "plugin"
	if len(argv) > 0 {
		dir = argv[0]
	}
	cmd := exec.Command("claude", "plugin", "validate", dir)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}
