// Reload is a mod whose daemon dies right after session.start, as an idle exit
// would between two hooks of a session. The command that follows reports
// whether the process that answered it still remembered session.start.
package main

import (
	"fmt"
	"os"
	"time"

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
		go func() { time.Sleep(50 * time.Millisecond); os.Exit(0) }()
		return c.Pass()
	})
	app.OnCommand("pid", func(c *mods.Ctx, e mods.CommandRunEvent) (any, error) {
		return mods.Text(fmt.Sprintf("pid=%d kept-memory=%t", os.Getpid(), sawStart)), nil
	})
	app.Run()
}
