/*
Supervisorhttp provides the HTTP API for working with Supervisor.
*/
package supervisorhttp

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/LeonidVas/tvisor/supervisor/core"
)

// SupervisorHandler is used to communicate with the Supervisor over HTTP.
type SupervisorHandler struct {
	sv *core.Supervisor
}

// statusResult describes the result of the "status" command.
type statusResult struct {
	Status *core.InstanceStatus `json:"status"`
}

// startResult describes the result of the "start" command.
type startResult struct {
	ID int `json:"id"`
}

// listResult describes the result of the "list" command.
type listResult struct {
	Instances map[string]*core.InstanceStatus `json:"instances"`
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

// callCommand invokes the command and returns the execution result.
func callCommand(cmd *command, sv *core.Supervisor) interface{} {
	var res interface{}
	switch cmd.Name {
	case "start":
		id, err := sv.StartInstance(cmd.Params.Name, cmd.Params.Env,
			cmd.Params.Restartable)
		if err != nil {
			return &errorResult{`Can't start an Instance: "` + err.Error() + `"`}
		}
		res = &startResult{id}
	case "stop":
		if err := sv.StopInstance(cmd.Params.ID, cmd.Params.Force); err != nil {
			return &errorResult{`Can't stop the Instace: "` + err.Error() + `"`}
		}
		res = &doneResult{true}
	case "status":
		status, err := sv.GetInstanceStatus(cmd.Params.ID)
		if err != nil {
			return &errorResult{`Can't get the Instace: "` + err.Error() + `"`}
		}
		res = &statusResult{status}
	case "list":
		res = &listResult{sv.ListInstances()}
	}

	return res
}

// NewSupervisorHandler creates SupervisorHandler.
func NewSupervisorHandler(sv *core.Supervisor) *SupervisorHandler {
	return &SupervisorHandler{sv: sv}
}

// ServeHTTP handles requests to the Supervisor.
func (handler *SupervisorHandler) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	// Parse, check and call the command.
	var res interface{}
	var status int
	var cmd command
	if err := parseCommand(req.Body, &cmd); err != nil {
		status = http.StatusBadRequest
		res = &errorResult{err.Error()}
	} else {
		status = http.StatusOK
		res = callCommand(&cmd, handler.sv)
	}

	// Write the result.
	wr.Header().Set("Content-Type", "application/json")
	wr.WriteHeader(status)
	if err := json.NewEncoder(wr).Encode(res); err != nil {
		log.Printf("An error occurred while encoding the response: \"%v\"\n", err)
	}
}
