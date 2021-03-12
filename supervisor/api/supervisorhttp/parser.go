package supervisorhttp

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/mitchellh/mapstructure"
)

// commandJSON describes the Supervisor command sent using the HTTP API.
type commandJSON struct {
	// Name - name of the command.
	// Now available: start, stop, status, list.
	Name string `json:"command_name"`
	// Params - command parameters.
	Params map[string]interface{} `json:"params"`
}

// paramSpec describes the requirements for the parameter.
type paramSpec struct {
	Required bool
	Default  interface{}
}

// cmdParamsSpec describes the parameter requirements
// for all available commands.
var cmdParamsSpec = map[string]map[string]paramSpec{
	"start": {
		"name":        {Required: true},
		"env":         {Required: false},
		"restartable": {Required: false, Default: true},
	},
	"stop": {
		"id":    {Required: true},
		"force": {Required: false, Default: true},
	},
	"status": {
		"id": {Required: true},
	},
	"list": {},
}

// commandParams structure contains all the parameters
// of all commands that can be passed through the HTTP API.
type commandParams struct {
	// ID - Instance ID.
	ID int
	// Name - Instance name.
	Name string
	// Env - environment variables for the starting Instance.
	Env []string
	// Restartable - the setting is responsible for the
	// need to restart the Instance on failure.
	// Default: true.
	Restartable bool
	// Force - the setting is responsible for "force" termination
	// the Instance in case of a graceful termination failure.
	// Default: true.
	Force bool
}

// command describes the Supervisor command
type command struct {
	// Name - name of the command.
	Name string
	// Params - command parameters.
	Params commandParams
}

// parseCommand decodes JSON, checks the parameters
// and parses them to a "command" struct.
func parseCommand(r io.Reader, cmd *command) error {
	// Decode JSON
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()

	var cmdJSON commandJSON
	if err := decoder.Decode(&cmdJSON); err != nil {
		return err
	}

	// Check command name.
	cmdSpec, ok := cmdParamsSpec[cmdJSON.Name]
	if !ok {
		return errors.New(`Unknown command name: "` + cmdJSON.Name + `".`)
	}

	// Check parameters.
	checkedParamsCount := 0
	for paramName, spec := range cmdSpec {
		_, ok := cmdJSON.Params[paramName]
		if !ok {
			if spec.Required {
				return errors.New(`A required parameter "` + paramName + `" is absent.`)
			} else if spec.Default != nil {
				cmdJSON.Params[paramName] = spec.Default
			} else {
				continue
			}
		}
		checkedParamsCount++
	}

	// If all input parameters have been checked, it is meaningless
	// to check for existence of unknown parameters.
	if len(cmdJSON.Params) != checkedParamsCount {
		for paramName := range cmdJSON.Params {
			if _, ok := cmdSpec[paramName]; !ok {
				return errors.New(`Unknown parameter "` + paramName + `".`)
			}
		}
	}

	// Parse cmdJSON to a "command" structure.
	// Additionally, all types of parameters will be checked.
	err := mapstructure.Decode(cmdJSON, cmd)
	if err != nil {
		err = fmt.Errorf(`Failed to parse command params: "%v"`, err.Error())
	}

	return err
}
