package mods

// Method names a mods API method as "namespace.method", the way Claude Code
// names the event each call raises: $.fs.read is "fs.read".
type Method string

// The mods API methods a Go mod can call. A mod declares the ones it uses with
// App.Uses; the generated shim gets a case for each and for no others.
//
// Not bridged: methods that take a function ($.clock.after, $.clock.every,
// $.state), $.env (its name must be a string literal), and the drawing calls
// that need element trees ($.ui.open, $.ui.resolve, $.ui.blit).
const (
	UIInvalidate Method = "ui.invalidate"
	UIClose      Method = "ui.close"
	UIPanes      Method = "ui.panes"
	UIFocus      Method = "ui.focus"
	UIScroll     Method = "ui.scroll"
	UIToast      Method = "ui.toast"
	UIStatus     Method = "ui.status"
	UILog        Method = "ui.log"
	UINotice     Method = "ui.notice"
	UIAsk        Method = "ui.ask"
	UICopy       Method = "ui.copy"
	UISelection  Method = "ui.selection"

	CommandRegister Method = "command.register"
	CommandRun      Method = "command.run"
	CommandList     Method = "command.list"

	ToolRegister Method = "tool.register"
	ToolCall     Method = "tool.call"
	ToolCheck    Method = "tool.check"
	ToolList     Method = "tool.list"

	AgentRegister Method = "agent.register"
	AgentSpawn    Method = "agent.spawn"
	AgentList     Method = "agent.list"

	ModelComplete Method = "model.complete"
	ModelFork     Method = "model.fork"
	ModelClassify Method = "model.classify"

	PromptSubmit  Method = "prompt.submit"
	PromptRead    Method = "prompt.read"
	PromptFill    Method = "prompt.fill"
	PromptSuggest Method = "prompt.suggest"
	PromptCompose Method = "prompt.compose"

	TurnAbort Method = "turn.abort"

	SessionMessages  Method = "session.messages"
	SessionCwd       Method = "session.cwd"
	SessionRoot      Method = "session.root"
	SessionModel     Method = "session.model"
	SessionTurns     Method = "session.turns"
	SessionID        Method = "session.id"
	SessionRepo      Method = "session.repo"
	SessionSurfaces  Method = "session.surfaces"
	SessionUsage     Method = "session.usage"
	SessionVersion   Method = "session.version"
	SessionCompact   Method = "session.compact"
	SessionSend      Method = "session.send"
	SessionAppend    Method = "session.append"
	SessionAuthorize Method = "session.authorize"

	ConfigList Method = "config.list"
	ConfigSet  Method = "config.set"

	SettingsRead Method = "settings.read"

	FSRead      Method = "fs.read"
	FSWrite     Method = "fs.write"
	FSList      Method = "fs.list"
	FSExists    Method = "fs.exists"
	FSStat      Method = "fs.stat"
	FSAncestors Method = "fs.ancestors"

	StoreGet    Method = "store.get"
	StoreSet    Method = "store.set"
	StoreDelete Method = "store.delete"
	StoreKeys   Method = "store.keys"

	ClockNow   Method = "clock.now"
	ClockSleep Method = "clock.sleep"

	HTTPFetch Method = "http.fetch"

	ProcessRun Method = "process.run"

	MCPCall    Method = "mcp.call"
	MCPConnect Method = "mcp.connect"

	AudioPlay  Method = "audio.play"
	AudioSpeak Method = "audio.speak"

	TelemetryLog  Method = "telemetry.log"
	TelemetryMark Method = "telemetry.mark"
)

// KnownMethods is every method the generator can bridge, in a stable order.
var KnownMethods = []Method{
	UIInvalidate, UIClose, UIPanes, UIFocus, UIScroll, UIToast, UIStatus, UILog, UINotice, UIAsk, UICopy, UISelection,
	CommandRegister, CommandRun, CommandList,
	ToolRegister, ToolCall, ToolCheck, ToolList,
	AgentRegister, AgentSpawn, AgentList,
	ModelComplete, ModelFork, ModelClassify,
	PromptSubmit, PromptRead, PromptFill, PromptSuggest, PromptCompose,
	TurnAbort,
	SessionMessages, SessionCwd, SessionRoot, SessionModel, SessionTurns, SessionID, SessionRepo, SessionSurfaces,
	SessionUsage, SessionVersion, SessionCompact, SessionSend, SessionAppend, SessionAuthorize,
	ConfigList, ConfigSet,
	SettingsRead,
	FSRead, FSWrite, FSList, FSExists, FSStat, FSAncestors,
	StoreGet, StoreSet, StoreDelete, StoreKeys,
	ClockNow, ClockSleep,
	HTTPFetch,
	ProcessRun,
	MCPCall, MCPConnect,
	AudioPlay, AudioSpeak,
	TelemetryLog, TelemetryMark,
}

// IsKnown reports whether m can be bridged.
func IsKnown(m Method) bool {
	for _, k := range KnownMethods {
		if k == m {
			return true
		}
	}
	return false
}
