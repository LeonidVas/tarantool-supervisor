package supervisor

import (
	"regexp"
	"testing"
	"time"
)

// Test the basic Supervisor functionality.
func TestSupervisorBase(t *testing.T) {
	// Create config for Supervisor.
	cfg := new(Cfg)
	cfg.InstancesDir = "test_instances"
	cfg.TermTimeout = 100 * time.Millisecond

	// Create Supervisor.
	sv := NewSupervisor(cfg)

	// Terminate all Instances at the end of the test.
	defer sv.StopAllInstance()

	// Run several instances (some with additional env and some without).
	instName := "test_instance"
	id1, err := sv.StartInstance(instName, nil, false)
	if err != nil {
		t.Errorf("Can't start the Instance.\nError: \"%v\"", err)
	}
	id2, err := sv.StartInstance(instName, []string{"INSTSIGIGNORE=true"}, false)
	if err != nil {
		t.Errorf("Can't start the Instance.\nError: \"%v\"", err)
	}
	if _, err = sv.StartInstance(instName, nil, false); err != nil {
		t.Errorf("Can't start the Instance.\nError: \"%v\"", err)
	}
	if _, err = sv.StartInstance(instName, []string{"INSTSIGIGNORE=true"}, false); err != nil {
		t.Errorf("Can't start the Instance.\nError: \"%v\"", err)
	}

	// We need to wait for the new process to set handlers.
	// It is necessary to update for more correct synchronization.
	time.Sleep(100 * time.Millisecond)

	// Check  all instance are running.
	if len(sv.ListInstances()) != 4 {
		t.Errorf("Not all instances have been started.")
	}

	// Check the functionality of "StatusInstance".
	status, err := sv.StatusInstance(id2)
	if err != nil {
		t.Errorf("Can't get Instance status. Error: \"%v\"", err)
	}
	matched, err := regexp.MatchString("running.*INSTSIGIGNORE=true", status)
	if !matched {
		t.Errorf("Status of Instance is not correct. Error: \"%v\"", err)
	}

	// Check the functionality of "StopInstance".
	if err = sv.StopInstance(id1, false); err != nil {
		t.Errorf("Can't stop the Instance. Error: \"%v\"", err)
	}
	if len(sv.ListInstances()) != 3 {
		t.Errorf("Can't stop the Instance.")
	}

	// Check "StopInstance" on the Instance with "INSTSIGIGNORE=true".
	if err = sv.StopInstance(id2, false); err == nil {
		t.Errorf("The Instance with \"INSTSIGIGNORE=true\" has been terminated.")
	}
	// And now stop the Instance with "INSTSIGIGNORE=true" by using "force" = true.
	if err = sv.StopInstance(id2, true); err != nil {
		t.Errorf("Can't stop the Instance")
	}
	if len(sv.ListInstances()) != 2 {
		t.Errorf("Expected number of instances is 2, but now it's %v",
			len(sv.ListInstances()))
	}

	// Terminate all remaining instances.
	sv.StopAllInstance()
	if len(sv.ListInstances()) != 0 {
		t.Errorf("Expected number of instances is 0, but now it's %v",
			len(sv.ListInstances()))
	}
}
