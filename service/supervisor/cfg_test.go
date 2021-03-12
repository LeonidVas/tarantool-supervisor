package supervisor

import (
	"testing"
	"time"
)

func TestCfgParse(t *testing.T) {
	path := "test_cfg.json"

	cfg, err := ParseCfg(path)
	if err != nil {
		t.Errorf("Failed to parse the config. Error: \"%v\"", err)
	}

	if cfg.TermTimeout != 1*time.Second || cfg.InstancesDir != "test_instances" {
		t.Errorf("Failed to parse the config.")
	}
}
