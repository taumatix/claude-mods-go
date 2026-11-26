// Reload is a mod whose daemon vanishes between session.start and the command
// that follows: the first command a daemon that saw session.start receives
// kills it, so the shim must start a new daemon and have that one answer. The
// command reports whether the process that answered still remembered
// session.start. Dying on the request, not on a timer, keeps the test from
// depending on how fast Claude Code gets from one hook to the next.
package main

import (
	"fmt"
	"os"

	"github.com/taumatix/claude-mods-go/mods"
)

func main() {
	app := mods.New("reload").Uses(mods.CommandRegister)
	sawStart := false
	app.OnSessionStart(func(c *mods.Ctx, e mods.SessionStartEvent) (any, error) {
		sawStart = true
		if err := c.Command.Register(mods.CommandSpec{Name: "pid", Description: "daemon process id"}); err != nil {
			return nil, err
		}
		return c.Pass()
	})
	app.OnCommand("pid", func(c *mods.Ctx, e mods.CommandRunEvent) (any, error) {
		if sawStart {
			os.Exit(1)
		}
		return mods.Text(fmt.Sprintf("pid=%d kept-memory=%t", os.Getpid(), sawStart)), nil
	})
	app.Run()
}
