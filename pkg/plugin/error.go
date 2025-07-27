package plugin

import (
	"fmt"

	"github.com/mimuret/dtap/v3/pkg/types"
)

func PluginError(p types.Plugin, format string, args ...any) error {
	return &PluginErrorImpl{
		Plugin: p,
		Err:    fmt.Errorf(format, args...),
	}
}

type PluginErrorImpl struct {
	Plugin types.Plugin
	Err    error
}

func (e *PluginErrorImpl) Error() string {
	return fmt.Sprintf("plugin: %s err: %v", e.Plugin.GetFullName(), e.Err)
}

func (e *PluginErrorImpl) Unwrap() error {
	return e.Err
}

func (e *PluginErrorImpl) PluginName() string {
	return e.Plugin.GetFullName()
}
