//go:build linux && amd64

package varstore

import (
	_ "embed"

	"codeberg.org/msantos/embedexe/exec"
)

//go:embed virt-fw-vars-linux-amd64
var virt_fw_vars_bin []byte

func VirtFwVars(args ...string) ([]byte, error) {
	cmd := exec.Command(virt_fw_vars_bin, args...)
	return cmd.CombinedOutput()
}
