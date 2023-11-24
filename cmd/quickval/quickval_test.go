package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
)

func testSG() *cli.App {
	tapp := *app
	return &tapp
}

func TestAppRun(t *testing.T) {
	sg := testSG()

	var out, err bytes.Buffer
	sg.Writer = &out
	sg.ErrWriter = &err
	// check app launches up correctly
	assert.NoError(t, sg.Run([]string{
		"dcf",
		"-h",
	}))
	// check that the output looks more or less correct
	assert.Contains(t, out.String(), "Perform quick valuations using the QuickFS API")
	assert.NotContains(t, out.String(), "error")
	assert.NotContains(t, out.String(), "panic")
	assert.Empty(t, err.String())
}
