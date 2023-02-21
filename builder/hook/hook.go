package hook

import (
	"fmt"
	"sort"
	"strings"

	"github.com/honmaple/snow/builder/page"
	"github.com/honmaple/snow/builder/static"
	"github.com/honmaple/snow/builder/theme"
	"github.com/honmaple/snow/config"
)

type (
	pageHooks   = page.Hooks
	staticHooks = static.Hooks
	BaseHook    struct {
		pageHooks
		staticHooks
	}

	Hook interface {
		Name() string
		page.Hook
		static.Hook
	}
	Hooks       []Hook
	hookCreator func(config.Config, theme.Theme) Hook
)

var (
	_hooks = make(map[string]hookCreator)
)

func (hooks Hooks) PageHooks() (result []page.Hook) {
	for _, hook := range hooks {
		result = append(result, hook)
	}
	return
}

func (hooks Hooks) StaticHooks() (result []static.Hook) {
	for _, hook := range hooks {
		result = append(result, hook)
	}
	return
}

func New(conf config.Config, theme theme.Theme) Hooks {
	names := conf.GetStringSlice("hooks")
	hooks := make([]Hook, 0)
	// 设置默认的hook
	if creator, ok := _hooks["internal"]; ok {
		hooks = append(hooks, creator(conf, theme))
	}
	for _, name := range names {
		if name == "internal" {
			continue
		}
		if creator, ok := _hooks[name]; ok {
			hooks = append(hooks, creator(conf, theme))
		}else {
			conf.Log.Warnf("The hook %s not found", name)
		}
	}
	return hooks
}

func Print() {
	names := make([]string, 0)
	for name := range _hooks {
		names = append(names, name)
	}
	sort.Strings(names)
	fmt.Println(strings.Join(names, ", "))
}

func Register(name string, creator hookCreator) {
	if _, ok := _hooks[name]; ok {
		panic(fmt.Sprintf("The hook %s has been registered", name))
	}
	_hooks[name] = creator
}
