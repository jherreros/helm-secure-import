package main

import (
	"encoding/json"
	"os"
	"os/exec"
)

func execCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func checkVulnerabilities(jsonFile string) (bool, error) {
	data, err := os.ReadFile(jsonFile)
	if err != nil {
		return false, err
	}

	var result TrivyResult
	if err := json.Unmarshal(data, &result); err != nil {
		return false, err
	}

	for _, r := range result.Results {
		if len(r.Vulnerabilities) > 0 {
			return true, nil
		}
	}

	return false, nil
}
