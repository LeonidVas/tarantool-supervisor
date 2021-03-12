package core

import (
	"time"
)

// Cfg stores Supervisor settings.
type Cfg struct {
	// InstancesDir - directory that stores
	// executable files for running Instances.
	InstancesDir string `json:"instances_dir"`
	// TermTimeout - time to wait for the Instance to
	// terminate correctly. After this timeout expires,
	// the SIGKILL signal will be used to stop the
	// instance if the force option is true, else an
	// error will be returned.
	TermTimeout time.Duration `json:"termination_timeout"`
}
