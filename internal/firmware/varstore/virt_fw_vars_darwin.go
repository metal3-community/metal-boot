//go:build darwin

package varstore

import (
	"fmt"
	"os/exec"
)

func VirtFwVars(args ...string) (string, error) {
	cmd := exec.Command("virt-fw-vars", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error executing virt-fw-vars: %v\nOutput: %s", err, string(output))
	}
	return string(output), nil
}
