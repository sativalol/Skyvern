package lua_plugin

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"skyvern/internal/config"
	"skyvern/internal/manager"
	"skyvern/internal/plugins"
	"skyvern/internal/storage"
	"sync"

	lua "github.com/yuin/gopher-lua"
)

type LuaPlugin struct {
	db          *storage.DB
	mgr         *manager.Manager
	mu          sync.RWMutex
	cmdFiles    map[string]string // trigger -> script path
	registered  map[string]bool   // trigger -> registered with mgr
}

var instance *LuaPlugin

func init() {
	instance = &LuaPlugin{
		cmdFiles:   make(map[string]string),
		registered: make(map[string]bool),
	}
	plugins.Register(instance)
}

func (p *LuaPlugin) Name() string {
	return "lua_plugin"
}

func (p *LuaPlugin) Init(db *storage.DB, mgr *manager.Manager) error {
	p.db = db
	p.mgr = mgr

	dir := config.ResolvePath("plugins/lua")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Create a dummy example script if directory is empty
	files, _ := ioutil.ReadDir(dir)
	if len(files) == 0 {
		exampleCode := `-- Example Lua command
skyvern.register_command({
    trigger = "pinglua",
    aliases = {"pl"},
    name = "pinglua",
    description = "Replying from Lua!",
    category = "fun",
    execute = function(ctx)
        ctx:reply("Pong! Hello from Lua, kitten. 🐾")
    end
})
`
		_ = ioutil.WriteFile(filepath.Join(dir, "example.lua"), []byte(exampleCode), 0644)
	}

	p.Reload()

	// Register reload command
	mgr.AddCommands([]*manager.Command{
		{
			Trigger:     "reloadlua",
			Name:        "reloadlua",
			Description: "Reload all Lua scripts and dynamically register new commands.",
			Category:    "utility",
			Execute: func(ctx *manager.CommandContext) error {
				if err := p.Reload(); err != nil {
					return ctx.Reply(fmt.Sprintf("Error reloading Lua scripts: %v", err))
				}
				return ctx.Reply("Successfully reloaded Lua scripts, my sweet girl.")
			},
		},
	})

	return nil
}

func (p *LuaPlugin) Commands() []*manager.Command {
	// Manager calls Commands() at startup, but since we scan in Init, we can dynamically add them.
	return nil
}

func (p *LuaPlugin) Reload() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	dir := config.ResolvePath("plugins/lua")
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	p.cmdFiles = make(map[string]string)

	for _, f := range files {
		if filepath.Ext(f.Name()) == ".lua" {
			p.scanScript(filepath.Join(dir, f.Name()))
		}
	}

	return nil
}

func (p *LuaPlugin) scanScript(path string) {
	L := lua.NewState()
	defer L.Close()

	skyvern := L.NewTable()
	L.SetGlobal("skyvern", skyvern)

	L.SetField(skyvern, "register_command", L.NewFunction(func(L *lua.LState) int {
		tbl := L.CheckTable(1)
		trigger := tbl.RawGetString("trigger").String()
		if trigger == "" {
			return 0
		}

		p.cmdFiles[trigger] = path

		var aliases []string
		if alTbl, ok := tbl.RawGetString("aliases").(*lua.LTable); ok {
			alTbl.ForEach(func(_, v lua.LValue) {
				aliases = append(aliases, v.String())
			})
		}
		for _, al := range aliases {
			p.cmdFiles[al] = path
		}

		name := tbl.RawGetString("name").String()
		if name == "" {
			name = trigger
		}
		desc := tbl.RawGetString("description").String()
		category := tbl.RawGetString("category").String()
		if category == "" {
			category = "lua"
		}

		// Only register with manager if not already registered
		if !p.registered[trigger] {
			p.registered[trigger] = true
			cmd := &manager.Command{
				Trigger:     trigger,
				Aliases:     aliases,
				Name:        name,
				Description: desc,
				Category:    category,
				Execute: func(ctx *manager.CommandContext) error {
					return p.ExecuteLuaCommand(trigger, ctx)
				},
			}
			p.mgr.AddCommands([]*manager.Command{cmd})
		}

		return 0
	}))

	_ = L.DoFile(path)
}

func (p *LuaPlugin) ExecuteLuaCommand(trigger string, ctx *manager.CommandContext) error {
	p.mu.RLock()
	path, ok := p.cmdFiles[trigger]
	p.mu.RUnlock()

	if !ok {
		return fmt.Errorf("command not found")
	}

	L := lua.NewState()
	defer L.Close()

	registerContextType(L)

	skyvern := L.NewTable()
	L.SetGlobal("skyvern", skyvern)

	executed := false
	var execErr error

	L.SetField(skyvern, "register_command", L.NewFunction(func(L *lua.LState) int {
		tbl := L.CheckTable(1)
		currTrigger := tbl.RawGetString("trigger").String()
		var isMatch bool
		if currTrigger == trigger {
			isMatch = true
		} else {
			// Also check aliases
			if alTbl, ok := tbl.RawGetString("aliases").(*lua.LTable); ok {
				alTbl.ForEach(func(_, v lua.LValue) {
					if v.String() == trigger {
						isMatch = true
					}
				})
			}
		}

		if isMatch {
			fn := tbl.RawGetString("execute")
			if fn.Type() == lua.LTFunction {
				ud := L.NewUserData()
				ud.Value = ctx
				L.SetMetatable(ud, L.GetTypeMetatable("CommandContext"))

				L.Push(fn)
				L.Push(ud)
				executed = true
				if err := L.PCall(1, 0, nil); err != nil {
					execErr = err
				}
			}
		}
		return 0
	}))

	if err := L.DoFile(path); err != nil {
		return err
	}

	if !executed {
		return fmt.Errorf("lua trigger %s did not execute", trigger)
	}

	return execErr
}

func registerContextType(L *lua.LState) {
	mt := L.NewTypeMetatable("CommandContext")
	L.SetGlobal("CommandContext", mt)
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"reply": func(L *lua.LState) int {
			ud := L.CheckUserData(1)
			ctx, ok := ud.Value.(*manager.CommandContext)
			if !ok {
				L.ArgError(1, "expected CommandContext")
				return 0
			}
			text := L.CheckString(2)
			err := ctx.Reply(text)
			if err != nil {
				L.Push(lua.LString(err.Error()))
				return 1
			}
			L.Push(lua.LNil)
			return 1
		},
		"send_text": func(L *lua.LState) int {
			ud := L.CheckUserData(1)
			ctx, ok := ud.Value.(*manager.CommandContext)
			if !ok {
				L.ArgError(1, "expected CommandContext")
				return 0
			}
			text := L.CheckString(2)
			err := ctx.SendText(text)
			if err != nil {
				L.Push(lua.LString(err.Error()))
				return 1
			}
			L.Push(lua.LNil)
			return 1
		},
		"guild_id": func(L *lua.LState) int {
			ud := L.CheckUserData(1)
			ctx, ok := ud.Value.(*manager.CommandContext)
			if !ok {
				L.ArgError(1, "expected CommandContext")
				return 0
			}
			L.Push(lua.LString(ctx.GuildID()))
			return 1
		},
		"channel_id": func(L *lua.LState) int {
			ud := L.CheckUserData(1)
			ctx, ok := ud.Value.(*manager.CommandContext)
			if !ok {
				L.ArgError(1, "expected CommandContext")
				return 0
			}
			L.Push(lua.LString(ctx.ChanID()))
			return 1
		},
		"author_id": func(L *lua.LState) int {
			ud := L.CheckUserData(1)
			ctx, ok := ud.Value.(*manager.CommandContext)
			if !ok {
				L.ArgError(1, "expected CommandContext")
				return 0
			}
			L.Push(lua.LString(ctx.AuthorID()))
			return 1
		},
		"author_tag": func(L *lua.LState) int {
			ud := L.CheckUserData(1)
			ctx, ok := ud.Value.(*manager.CommandContext)
			if !ok {
				L.ArgError(1, "expected CommandContext")
				return 0
			}
			L.Push(lua.LString(ctx.AuthorTag()))
			return 1
		},
		"args": func(L *lua.LState) int {
			ud := L.CheckUserData(1)
			ctx, ok := ud.Value.(*manager.CommandContext)
			if !ok {
				L.ArgError(1, "expected CommandContext")
				return 0
			}
			tbl := L.NewTable()
			for _, arg := range ctx.Args {
				tbl.Append(lua.LString(arg))
			}
			L.Push(tbl)
			return 1
		},
	}))
}
