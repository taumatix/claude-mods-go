// Probe exercises several bridged calls in one handler for the e2e tests.
package main

import (
	"fmt"
	"strings"

	"github.com/taumatix/claude-mods-go/mods"
)

func main() {
	app := mods.New("probe").
		Uses(mods.CommandRegister, mods.FSWrite, mods.FSRead, mods.ProcessRun, mods.SessionCwd, mods.UIToast)

	app.OnSessionStart(func(c *mods.Ctx, e mods.SessionStartEvent) (any, error) {
		if err := c.Command.Register(mods.CommandSpec{Name: "probe", Description: "bridge probe"}); err != nil {
			return nil, err
		}
		if err := c.Command.Register(mods.CommandSpec{Name: "boom", Description: "fails on purpose"}); err != nil {
			return nil, err
		}
		return c.Pass()
	})

	app.OnCommand("probe", func(c *mods.Ctx, e mods.CommandRunEvent) (any, error) {
		path := strings.TrimSpace(e.Args)
		if err := c.FS.Write(path, "from go"); err != nil {
			return nil, err
		}
		got, err := c.FS.Read(path)
		if err != nil {
			return nil, err
		}
		r, err := c.Process.Run([]string{"echo", "ran"})
		if err != nil {
			return nil, err
		}
		return mods.Text(fmt.Sprintf("read=%s proc=%s", got, strings.TrimSpace(r.Stdout))), nil
	})

	app.OnCommand("boom", func(c *mods.Ctx, e mods.CommandRunEvent) (any, error) {
		_, err := c.FS.Read("/definitely/not/here")
		return nil, err
	})

	app.Run()
}
