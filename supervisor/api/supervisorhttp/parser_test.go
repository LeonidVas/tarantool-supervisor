package supervisorhttp

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

// parse parses the command from JSON into a "command"
// struct with a check for success.
func parse(t *testing.T, jsonByte []byte, cmd *command) {
	assert := assert.New(t)
	err := parseCommand(bytes.NewReader(jsonByte), cmd)
	assert.Nilf(err, `Can't parse a command: "%v".`, err)
}

// assertParseFails parses the command from JSON into a "command"
// struct with a check for failure.
func assertParseFails(t *testing.T, jsonByte []byte) {
	assert := assert.New(t)
	var cmd command
	err := parseCommand(bytes.NewReader(jsonByte), &cmd)
	assert.NotNil(err, "Successfully parsed an invalid command.")
}

// TestParser tests positive cases of command parsing.
func TestParser(t *testing.T) {
	assert := assert.New(t)
	// Start command parsing check.
	jsonStart := []byte(`{
  "command_name": "start",
  "params": {
    "name": "test_inst",
    "env": [
      "TRYAM=true"
    ]
  }
}
`)

	var cmd command
	parse(t, jsonStart, &cmd)
	// Check parsing result.
	assert.Equal(cmd.Name, "start")
	assert.Equal(cmd.Params.Name, "test_inst")
	assert.Equal(cmd.Params.Restartable, true)
	assert.Equal(cmd.Params.Env[0], "TRYAM=true")

	// Stop command parsing check.
	jsonStop := []byte(`{
  "command_name": "stop",
  "params": {
    "id": 1,
    "force": false
  }
}
`)

	parse(t, jsonStop, &cmd)
	// Check parsing result.
	assert.Equal(cmd.Name, "stop")
	assert.Equal(cmd.Params.ID, 1)
	assert.Equal(cmd.Params.Force, false)

	// Status command parsing check.
	jsonStatus := []byte(`{
  "command_name": "status",
  "params": {
    "id": 1
  }
}
`)

	parse(t, jsonStatus, &cmd)
	// Check parsing result.
	assert.Equal(cmd.Name, "status")
	assert.Equal(cmd.Params.ID, 1)

	// Status command parsing check.
	jsonList := []byte(`{
  "command_name": "list"
}
`)

	parse(t, jsonList, &cmd)
	// Check parsing result.
	assert.Equal(cmd.Name, "list")
}

// TestParserNegative tests negative cases of command parsing.
func TestParserNegative(t *testing.T) {
	// Check type validation.
	jsonBadCmd := []byte(`{
  "command_name": "start",
  "params": {
    "name": true,
    "env": [
      "TRYAM=true"
    ]
  }
}
`)
	assertParseFails(t, jsonBadCmd)

	// Check params validation (required parameter("name") is missing).
	jsonBadCmd = []byte(`{
  "command_name": "start",
  "params": {
    "env": [
      "TRYAM=true"
    ]
  }
}
`)
	assertParseFails(t, jsonBadCmd)

	// Check params validation (unknown parameter is present).
	jsonBadCmd = []byte(`{
  "command_name": "start",
  "params": {
    "name": "test_inst",
    "unknown_param": 5,
    "env": [
      "TRYAM=true"
    ]
  }
}
`)
	assertParseFails(t, jsonBadCmd)
}
