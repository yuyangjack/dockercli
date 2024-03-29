package formatter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/yuyangjack/moby/api/types"
	"github.com/yuyangjack/moby/pkg/stringid"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

func TestVolumeContext(t *testing.T) {
	volumeName := stringid.GenerateRandomID()

	var ctx volumeContext
	cases := []struct {
		volumeCtx volumeContext
		expValue  string
		call      func() string
	}{
		{volumeContext{
			v: types.Volume{Name: volumeName},
		}, volumeName, ctx.Name},
		{volumeContext{
			v: types.Volume{Driver: "driver_name"},
		}, "driver_name", ctx.Driver},
		{volumeContext{
			v: types.Volume{Scope: "local"},
		}, "local", ctx.Scope},
		{volumeContext{
			v: types.Volume{Mountpoint: "mountpoint"},
		}, "mountpoint", ctx.Mountpoint},
		{volumeContext{
			v: types.Volume{},
		}, "", ctx.Labels},
		{volumeContext{
			v: types.Volume{Labels: map[string]string{"label1": "value1", "label2": "value2"}},
		}, "label1=value1,label2=value2", ctx.Labels},
	}

	for _, c := range cases {
		ctx = c.volumeCtx
		v := c.call()
		if strings.Contains(v, ",") {
			compareMultipleValues(t, v, c.expValue)
		} else if v != c.expValue {
			t.Fatalf("Expected %s, was %s\n", c.expValue, v)
		}
	}
}

func TestVolumeContextWrite(t *testing.T) {
	cases := []struct {
		context  Context
		expected string
	}{

		// Errors
		{
			Context{Format: "{{InvalidFunction}}"},
			`Template parsing error: template: :1: function "InvalidFunction" not defined
`,
		},
		{
			Context{Format: "{{nil}}"},
			`Template parsing error: template: :1:2: executing "" at <nil>: nil is not a command
`,
		},
		// Table format
		{
			Context{Format: NewVolumeFormat("table", false)},
			`DRIVER              VOLUME NAME
foo                 foobar_baz
bar                 foobar_bar
`,
		},
		{
			Context{Format: NewVolumeFormat("table", true)},
			`foobar_baz
foobar_bar
`,
		},
		{
			Context{Format: NewVolumeFormat("table {{.Name}}", false)},
			`VOLUME NAME
foobar_baz
foobar_bar
`,
		},
		{
			Context{Format: NewVolumeFormat("table {{.Name}}", true)},
			`VOLUME NAME
foobar_baz
foobar_bar
`,
		},
		// Raw Format
		{
			Context{Format: NewVolumeFormat("raw", false)},
			`name: foobar_baz
driver: foo

name: foobar_bar
driver: bar

`,
		},
		{
			Context{Format: NewVolumeFormat("raw", true)},
			`name: foobar_baz
name: foobar_bar
`,
		},
		// Custom Format
		{
			Context{Format: NewVolumeFormat("{{.Name}}", false)},
			`foobar_baz
foobar_bar
`,
		},
	}

	for _, testcase := range cases {
		volumes := []*types.Volume{
			{Name: "foobar_baz", Driver: "foo"},
			{Name: "foobar_bar", Driver: "bar"},
		}
		out := bytes.NewBufferString("")
		testcase.context.Output = out
		err := VolumeWrite(testcase.context, volumes)
		if err != nil {
			assert.Error(t, err, testcase.expected)
		} else {
			assert.Check(t, is.Equal(testcase.expected, out.String()))
		}
	}
}

func TestVolumeContextWriteJSON(t *testing.T) {
	volumes := []*types.Volume{
		{Driver: "foo", Name: "foobar_baz"},
		{Driver: "bar", Name: "foobar_bar"},
	}
	expectedJSONs := []map[string]interface{}{
		{"Driver": "foo", "Labels": "", "Links": "N/A", "Mountpoint": "", "Name": "foobar_baz", "Scope": "", "Size": "N/A"},
		{"Driver": "bar", "Labels": "", "Links": "N/A", "Mountpoint": "", "Name": "foobar_bar", "Scope": "", "Size": "N/A"},
	}
	out := bytes.NewBufferString("")
	err := VolumeWrite(Context{Format: "{{json .}}", Output: out}, volumes)
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		msg := fmt.Sprintf("Output: line %d: %s", i, line)
		var m map[string]interface{}
		err := json.Unmarshal([]byte(line), &m)
		assert.NilError(t, err, msg)
		assert.Check(t, is.DeepEqual(expectedJSONs[i], m), msg)
	}
}

func TestVolumeContextWriteJSONField(t *testing.T) {
	volumes := []*types.Volume{
		{Driver: "foo", Name: "foobar_baz"},
		{Driver: "bar", Name: "foobar_bar"},
	}
	out := bytes.NewBufferString("")
	err := VolumeWrite(Context{Format: "{{json .Name}}", Output: out}, volumes)
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		msg := fmt.Sprintf("Output: line %d: %s", i, line)
		var s string
		err := json.Unmarshal([]byte(line), &s)
		assert.NilError(t, err, msg)
		assert.Check(t, is.Equal(volumes[i].Name, s), msg)
	}
}
