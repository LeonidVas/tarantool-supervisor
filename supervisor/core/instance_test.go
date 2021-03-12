package core

import (
	"os"
	"os/exec"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const testInstPath = "../../test_instances"
const testInstName = "test_instance.lua"

// startTestInstance run an Instance and checks its status.
func startTestInstance(t *testing.T, isSignalIgnore bool) *Instance {
	assert := assert.New(t)

	instPath := path.Join(testInstPath, testInstName)
	_, err := exec.LookPath(instPath)
	assert.Nilf(err, `Unknown instance:"%v".Error: "%v"`, instPath, err)

	// Create an Instance.
	cmd := exec.Command(instPath)
	var env []string
	if isSignalIgnore {
		env = append(env, "INSTSIGIGNORE=true")
	}
	cmd.Env = append(os.Environ(), env...)
	inst := NewInstance(testInstName, cmd, env, false)

	// Start the Instance.
	err = inst.Start()
	assert.Nilf(err, `Can't start an Instance.Error: "%v"`, err)

	// Check status of the Instance.
	assert.Equal(inst.Status().State, stateRunning,
		"The Instance is not running.")

	// We need to wait for the new process to set handlers.
	// It is necessary to update for more correct synchronization.
	time.Sleep(100 * time.Millisecond)

	return inst
}

// cleanupTestInstances sends a SIGKILL signal to all test
// Instances that remain alive after the test done.
func cleanupTestInstances(insts []*Instance) {
	for _, inst := range insts {
		if inst.IsAlive() {
			inst.Cmd.Process.Kill()
		}
	}
}

// A simple test that starts an Instance, terminates it,
// and checks the status of the Instance.
func TestInstance(t *testing.T) {
	assert := assert.New(t)
	// Set a cleanup callback.
	var insts []*Instance
	t.Cleanup(func() { cleanupTestInstances(insts) })

	inst := startTestInstance(t, false)
	insts = append(insts, inst)

	err := inst.Stop(100*time.Millisecond, false)
	assert.Nilf(err, `Can't stop the Instance. Error: "%v"`, err)

	assert.Equal(inst.Status().State, stateTerminated,
		"The Instance hasn't been terminated.")
	// Check that Instance has been terminated.
	assert.False(inst.IsAlive(), "The Instance hasn't been terminated.")
}

// A test in which the Instance ignores the "SIGINT" signal.
func TestSignalIgnoreInstance(t *testing.T) {
	assert := assert.New(t)
	// Set a cleanup callback.
	var insts []*Instance
	t.Cleanup(func() { cleanupTestInstances(insts) })

	inst := startTestInstance(t, true)
	insts = append(insts, inst)

	// Try to stop the Instance using only the "SIGINT" signal.
	err := inst.Stop(100*time.Millisecond, false)
	assert.NotNilf(err, "Instance shouldn't be stopped.")

	// Check that the Instance is still running.
	assert.Equal(inst.Status().State, stateRunning,
		"The Instance doesn't ignore the signal.")

	// Let's try to stop the Instance using force == true.
	err = inst.Stop(100*time.Millisecond, true)
	assert.Nilf(err, `Can't stop the Instance. Error: "%v"`, err)

	// Check that the Instance has been terminated.
	assert.Equal(inst.Status().State, stateTerminated,
		"The Instance hasn't been terminated.")
}
