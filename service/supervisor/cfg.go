package supervisor

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"time"
)

// Cfg stores Supervisor settings.
type Cfg struct {
	// Instances_dir - directory that stores
	// executable files for running Instances.
	InstancesDir string `json:"inst_dir"`
	// TermTimeout - time to wait for the Instance to
	// terminate correctly. After this timeout expires,
	// the SIGKILL signal will be used to stop the
	// instance if the force option is true, else an
	// error will be returned.
	TermTimeout time.Duration `json:"termination_timeout"`
}

// ParseCfg parse the Supervisor JSON config.
func ParseCfg(path string) (*Cfg, error) {
	// Check is the file exists.
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	// Open the file.
	jsonFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()

	// Read and parse config.
	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}
	var cfg Cfg
	if err = json.Unmarshal(byteValue, &cfg); err != nil {
		return nil, err
	}

	// In the config, the time is indicated in seconds. Convert the value.
	cfg.TermTimeout = cfg.TermTimeout * time.Second

	return &cfg, nil
}
