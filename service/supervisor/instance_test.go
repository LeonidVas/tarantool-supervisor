package supervisor

import (
	"os"
	"os/exec"
	"path"
	"regexp"
	"testing"
	"time"
)

const testInstPath = "test_instances"
const testInstName = "test_instance.lua"

// commonStartTest run an Instance and checks its status.
func commonStartTest(t *testing.T, isSignalIgnore bool) *Instance {
	instPath := path.Join(testInstPath, testInstName)
	if _, err := exec.LookPath(instPath); err != nil {
		t.Errorf("Unknown instance:\"%v\"\nError: \"%v\"", instPath, err)
	}

	// Create an Instance.
	cmd := exec.Command(instPath)
	var env []string
	if isSignalIgnore {
		env = append(env, "INSTSIGIGNORE=true")
	}
	cmd.Env = append(os.Environ(), env...)
	inst := NewInstance(testInstName, cmd, env, false)

	// Start the Instance.
	if err := inst.Start(); err != nil {
		t.Errorf("Can't start an Instance.Error: \"%v\"", err)
	}

	// Check status of the Instance.
	matched, err := regexp.MatchString("running", inst.Status())
	if !matched || err != nil {
		t.Errorf("The Instance is not running. Error: %v", err)
	}

	// We need to wait for the new process to set handlers.
	// It is necessary to update for more correct synchronization.
	time.Sleep(100 * time.Millisecond)

	return inst
}

// A simple test that starts an Instance, terminates it,
// and checks the status of the Instance.
func TestInstance(t *testing.T) {
	inst := commonStartTest(t, false)

	if err := inst.Stop(100*time.Millisecond, false); err != nil {
		t.Errorf("Can't stop the Instance. Error: %v", err)
	}

	matched, err := regexp.MatchString("terminated", inst.Status())
	if !matched || err != nil {
		t.Errorf("The Instance hasn't been terminated. Error: %v", err)
	}
}

// A test in which the Instance ignores the "SIGINT" signal.
func TestSignalIgnoreInstance(t *testing.T) {
	inst := commonStartTest(t, true)

	// Try to stop the Instance using only the "SIGINT" signal.
	if err := inst.Stop(100*time.Millisecond, false); err == nil {
		t.Errorf("Can't stop the Instance. Error: %v", err)
	}

	// Check that the Instance is still running.
	matched, err := regexp.MatchString("running", inst.Status())
	if !matched || err != nil {
		t.Errorf("The Instance doesn't ignore the signal. Error: %v", err)
	}

	// Let's try to stop the Instance using force == true.
	if err := inst.Stop(100*time.Millisecond, true); err != nil {
		t.Errorf("Can't stop the Instance. Error: %v", err)
	}

	// Check that the Instance has been terminated.
	matched, err = regexp.MatchString("terminated", inst.Status())
	if !matched || err != nil {
		t.Errorf("The Instance hasn't been terminated. Error: %v", err)
	}
}
