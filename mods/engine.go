package mods

import (
	"encoding/json"
	"fmt"
)

// execute runs the handler for one event and returns its final Step.
func (a *App) execute(drv driver, run string, st Start) (final Step) {
	fail := func(format string, args ...any) Step {
		return Step{Run: run, Op: OpError, Message: fmt.Sprintf(format, args...)}
	}
	if st.Reg < 0 || st.Reg >= len(a.regs) {
		return fail("mods: no hook registered at index %d", st.Reg)
	}
	reg := a.regs[st.Reg]
	if reg.hook.Event != st.Event {
		return fail("mods: hook %d handles %q, not %q: the shim and the binary are from different builds", st.Reg, reg.hook.Event, st.Event)
	}
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(suspendSignal); ok {
				final = Step{Run: run, Op: "suspended"}
				return
			}
			final = fail("panic in %s handler: %v", st.Event, r)
		}
	}()
	v, err := reg.h(a.newCtx(drv, run, st.E), st.E)
	if err != nil {
		return fail("%v", err)
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fail("mods: encoding the %s result: %v", st.Event, err)
	}
	return Step{Run: run, Op: OpDone, Value: b}
}
