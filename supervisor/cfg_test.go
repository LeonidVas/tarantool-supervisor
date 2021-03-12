package main

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestCfgParse checks configuration parsing.
func TestCfgParse(t *testing.T) {
	assert := assert.New(t)
	// Config for the test.
	cfgStr := `{
  "instances_dir": "test_instances",
  "termination_timeout": 1
}
`
	// Create temporary cfg file.
	cfgFile, err := ioutil.TempFile("", "cfg")
	assert.Nilf(err, `Can't create test cfg. Error: "%v".`, err)
	defer os.Remove(cfgFile.Name())

	_, err = cfgFile.WriteString(cfgStr)
	assert.Nilf(err, `Can't write test cfg to file: "%v".`, err)
	cfgFile.Sync()

	// Parse and check the config.
	cfg, err := parseCfg(cfgFile.Name())
	assert.Nilf(err, `Failed to parse the config. Error: "%v".`, err)

	assert.True(cfg.TermTimeout == 1*time.Second &&
		cfg.InstancesDir == "test_instances", "Failed to parse the config.")
}
