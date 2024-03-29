// +build !windows

package connhelper

import (
	"context"
	"io"
	"testing"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

// For https://github.com/yuyangjack/dockercli/pull/1014#issuecomment-409308139
func TestCommandConnEOFWithError(t *testing.T) {
	ctx := context.TODO()
	cmd := "sh"
	args := []string{"-c", "echo hello; echo some error >&2; exit 42"}
	c, err := newCommandConn(ctx, cmd, args...)
	assert.NilError(t, err)
	b := make([]byte, 32)
	n, err := c.Read(b)
	assert.Check(t, is.Equal(len("hello\n"), n))
	assert.NilError(t, err)
	n, err = c.Read(b)
	assert.Check(t, is.Equal(0, n))
	assert.ErrorContains(t, err, "some error")
	assert.ErrorContains(t, err, "42")
}

func TestCommandConnEOFWithoutError(t *testing.T) {
	ctx := context.TODO()
	cmd := "sh"
	args := []string{"-c", "echo hello; echo some debug log >&2; exit 0"}
	c, err := newCommandConn(ctx, cmd, args...)
	assert.NilError(t, err)
	b := make([]byte, 32)
	n, err := c.Read(b)
	assert.Check(t, is.Equal(len("hello\n"), n))
	assert.NilError(t, err)
	n, err = c.Read(b)
	assert.Check(t, is.Equal(0, n))
	assert.Check(t, is.Equal(io.EOF, err))
}
