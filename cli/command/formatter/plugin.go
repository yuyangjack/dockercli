package formatter

import (
	"strings"

	"github.com/yuyangjack/moby/api/types"
	"github.com/yuyangjack/moby/pkg/stringid"
)

const (
	defaultPluginTableFormat = "table {{.ID}}\t{{.Name}}\t{{.Description}}\t{{.Enabled}}"

	pluginIDHeader    = "ID"
	descriptionHeader = "DESCRIPTION"
	enabledHeader     = "ENABLED"
)

// NewPluginFormat returns a Format for rendering using a plugin Context
func NewPluginFormat(source string, quiet bool) Format {
	switch source {
	case TableFormatKey:
		if quiet {
			return defaultQuietFormat
		}
		return defaultPluginTableFormat
	case RawFormatKey:
		if quiet {
			return `plugin_id: {{.ID}}`
		}
		return `plugin_id: {{.ID}}\nname: {{.Name}}\ndescription: {{.Description}}\nenabled: {{.Enabled}}\n`
	}
	return Format(source)
}

// PluginWrite writes the context
func PluginWrite(ctx Context, plugins []*types.Plugin) error {
	render := func(format func(subContext subContext) error) error {
		for _, plugin := range plugins {
			pluginCtx := &pluginContext{trunc: ctx.Trunc, p: *plugin}
			if err := format(pluginCtx); err != nil {
				return err
			}
		}
		return nil
	}
	pluginCtx := pluginContext{}
	pluginCtx.header = map[string]string{
		"ID":              pluginIDHeader,
		"Name":            nameHeader,
		"Description":     descriptionHeader,
		"Enabled":         enabledHeader,
		"PluginReference": imageHeader,
	}
	return ctx.Write(&pluginCtx, render)
}

type pluginContext struct {
	HeaderContext
	trunc bool
	p     types.Plugin
}

func (c *pluginContext) MarshalJSON() ([]byte, error) {
	return marshalJSON(c)
}

func (c *pluginContext) ID() string {
	if c.trunc {
		return stringid.TruncateID(c.p.ID)
	}
	return c.p.ID
}

func (c *pluginContext) Name() string {
	return c.p.Name
}

func (c *pluginContext) Description() string {
	desc := strings.Replace(c.p.Config.Description, "\n", "", -1)
	desc = strings.Replace(desc, "\r", "", -1)
	if c.trunc {
		desc = Ellipsis(desc, 45)
	}

	return desc
}

func (c *pluginContext) Enabled() bool {
	return c.p.Enabled
}

func (c *pluginContext) PluginReference() string {
	return c.p.PluginReference
}
