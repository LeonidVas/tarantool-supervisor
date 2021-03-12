package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Test the basic Supervisor functionality.
func TestSupervisorBase(t *testing.T) {
	assert := assert.New(t)
	// Create config for Supervisor.
	cfg := new(Cfg)
	cfg.InstancesDir = "../../test_instances"
	cfg.TermTimeout = 100 * time.Millisecond

	// Create Supervisor.
	sv := NewSupervisor(cfg)

	// Set a cleanup callback.
	t.Cleanup(func() { sv.StopAllInstances() })

	// Run several instances (some with additional env and some without).
	instName := "test_instance"

	id1, err := sv.StartInstance(instName, nil, false)
	assert.Nilf(err, `Can't start the Instance.Error: "%v"`, err)

	id2, err := sv.StartInstance(instName, []string{"INSTSIGIGNORE=true"}, false)
	assert.Nilf(err, `Can't start the Instance.Error: "%v"`, err)

	_, err = sv.StartInstance(instName, nil, false)
	assert.Nilf(err, `Can't start the Instance. Error: "%v"`, err)

	_, err = sv.StartInstance(instName, []string{"INSTSIGIGNORE=true"}, false)
	assert.Nilf(err, `Can't start the Instance. Error: "%v"`, err)

	// We need to wait for the new process to set handlers.
	// It is necessary to update for more correct synchronization.
	time.Sleep(100 * time.Millisecond)

	// Check  all instance are running.
	assert.Equal(len(sv.ListInstances()), 4,
		"Not all instances have been started.")

	// Check the functionality of "GetInstanceStatus".
	status, err := sv.GetInstanceStatus(id2)
	assert.Nilf(err, `Can't get Instance status. Error: "%v"`, err)

	assert.True(
		status.State == "running" && status.Env[0] == "INSTSIGIGNORE=true",
		"Status of Instance is not correct.")

	// Check the functionality of "StopInstance".
	assert.Nilf(sv.StopInstance(id1, false),
		`Can't stop the Instance. Error: "%v"`, err)
	assert.Equal(len(sv.ListInstances()), 3, "Can't stop the Instance.")

	// Check "StopInstance" on the Instance with "INSTSIGIGNORE=true".
	assert.NotNil(sv.StopInstance(id2, false),
		`The Instance with "INSTSIGIGNORE=true" has been terminated.`)
	// And now stop the Instance with "INSTSIGIGNORE=true" by using "force" = true.
	assert.Nil(sv.StopInstance(id2, true), "Can't stop the Instance")
	assert.Equalf(len(sv.ListInstances()), 2,
		"Expected number of instances is 2, but now it's %v",
		len(sv.ListInstances()))

	// Terminate all remaining instances.
	sv.StopAllInstances()
	assert.Equalf(len(sv.ListInstances()), 0,
		"Expected number of instances is 0, but now it's %v",
		len(sv.ListInstances()))
}
