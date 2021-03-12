/*
Supervisorhttp provides the HTTP API for working with Supervisor.
*/
package supervisorhttp

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/LeonidVas/tarantool-supervisor/service/supervisor"
)

// SupervisorHandler is used to communicate with the Supervisor over HTTP.
type SupervisorHandler struct {
	sv *supervisor.Supervisor
}

// params structure contains all the command parameters
// that can be passed through the HTTP API.
type params struct {
	// Inst - Instance name.
	Inst string `json:"inst"`
	// Env - environment variables for the starting Instance.
	Env []string `json:"env"`
	// IsRestart - the setting is responsible for the
	// need to restart the Instance on failure.
	// Default: true.
	IsRestart bool `json:"is_restart"`
	// id - Instance ID.
	Id int `json:"id"`
	// Force - the setting is responsible for "force" termination
	// the Instance in case of a graceful termination failure.
	// Default: true.
	Force bool `json:"force"`
}

// command describes the Supervisor command sent using the HTTP API.
type command struct {
	// Name - name of the command.
	// Now available: start, stop, status, list.
	Name string `json:"name"`
	// Params - command parameters.
	Params params `json:"params"`
}

// statusResult describes the result of the "status" command.
type statusResult struct {
	Status string `json:"status"`
}

// startResult describes the result of the "start" command.
type startResult struct {
	Id int `json:"id"`
}

// listResult describes the result of the "list" command.
type listResult struct {
	Instances []string
}

// doneResult describes the success of the command
// execution if there is no return value.
type doneResult struct {
	Done bool `json:"done"`
}

// errorResult describes a failure during the command execution.
type errorResult struct {
	Err string `json:"err"`
}

// call checks the parametrs, invokes the command
// and returns the execution result.
func call(cmd *command, sv *supervisor.Supervisor) interface{} {
	var res interface{}
	switch cmd.Name {
	case "start":
		if cmd.Params.Inst == "" {
			return &errorResult{"The Instance name is empty."}
		}
		id, err := sv.StartInstance(cmd.Params.Inst, cmd.Params.Env,
			cmd.Params.IsRestart)
		if err != nil {
			return &errorResult{"Can't start an Instance: \"" + err.Error() + "\""}
		}
		res = &startResult{id}
	case "stop":
		if cmd.Params.Id <= 0 {
			return &errorResult{"Instance ID is invalid."}
		}

		if err := sv.StopInstance(cmd.Params.Id, cmd.Params.Force); err != nil {
			return &errorResult{"Can't stop the Instace: \"" + err.Error() + "\""}
		}

		res = &doneResult{true}
	case "status":
		if cmd.Params.Id <= 0 {
			return &errorResult{"Instance ID is invalid."}
		}

		status, err := sv.StatusInstance(cmd.Params.Id)
		if err != nil {
			return &errorResult{"Can't get the Instace: \"" + err.Error() + "\""}
		}

		res = &statusResult{status}
	case "list":
		res = &listResult{sv.ListInstances()}
	default:
		res = &errorResult{"Unknown command name : \"" + cmd.Name + "\""}
	}

	return res
}

// NewSupervisorHandler creates SupervisorHandler.
func NewSupervisorHandler(sv *supervisor.Supervisor) *SupervisorHandler {
	return &SupervisorHandler{sv: sv}
}

// ServeHTTP handles requests to the Supervisor.
func (handler *SupervisorHandler) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	// Set command parameters to default.
	cmd := command{Params: params{IsRestart: true, Force: true}}

	// Parse and call the command.
	decoder := json.NewDecoder(req.Body)
	var res interface{}
	if err := decoder.Decode(&cmd); err != nil {
		res = &errorResult{err.Error()}
	} else {
		res = call(&cmd, handler.sv)
	}

	// Write the result.
	wr.Header().Set("Content-Type", "application/json")
	wr.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(wr).Encode(res); err != nil {
		log.Printf("An error occurred while encoding the response: \"%v\"\n", err)
	}
}
