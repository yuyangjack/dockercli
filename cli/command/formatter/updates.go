package formatter

import (
	clitypes "github.com/yuyangjack/dockercli/types"
)

const (
	defaultUpdatesTableFormat = "table {{.Type}}\t{{.Version}}\t{{.Notes}}"
	defaultUpdatesQuietFormat = "{{.Version}}"

	updatesTypeHeader = "TYPE"
	versionHeader     = "VERSION"
	notesHeader       = "NOTES"
)

// NewUpdatesFormat returns a Format for rendering using a updates context
func NewUpdatesFormat(source string, quiet bool) Format {
	switch source {
	case TableFormatKey:
		if quiet {
			return defaultUpdatesQuietFormat
		}
		return defaultUpdatesTableFormat
	case RawFormatKey:
		if quiet {
			return `update_version: {{.Version}}`
		}
		return `update_version: {{.Version}}\ntype: {{.Type}}\nnotes: {{.Notes}}\n`
	}
	return Format(source)
}

// UpdatesWrite writes the context
func UpdatesWrite(ctx Context, availableUpdates []clitypes.Update) error {
	render := func(format func(subContext subContext) error) error {
		for _, update := range availableUpdates {
			updatesCtx := &updateContext{trunc: ctx.Trunc, u: update}
			if err := format(updatesCtx); err != nil {
				return err
			}
		}
		return nil
	}
	updatesCtx := updateContext{}
	updatesCtx.header = map[string]string{
		"Type":    updatesTypeHeader,
		"Version": versionHeader,
		"Notes":   notesHeader,
	}
	return ctx.Write(&updatesCtx, render)
}

type updateContext struct {
	HeaderContext
	trunc bool
	u     clitypes.Update
}

func (c *updateContext) MarshalJSON() ([]byte, error) {
	return marshalJSON(c)
}

func (c *updateContext) Type() string {
	return c.u.Type
}

func (c *updateContext) Version() string {
	return c.u.Version
}

func (c *updateContext) Notes() string {
	return c.u.Notes
}
