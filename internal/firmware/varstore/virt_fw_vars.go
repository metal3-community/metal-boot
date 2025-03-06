package varstore

func ReadVirtFwVars(firmwareFile string, fwVarsJsonFile string) (string, error) {
	_, err := VirtFwVars("-i", firmwareFile, "--output-json", fwVarsJsonFile)
	return fwVarsJsonFile, err
}

func SaveVirtFwVars(firmwareFile string, fwVarsJsonFile string) error {
	_, err := VirtFwVars("--inplace", firmwareFile, "--set-json", fwVarsJsonFile)
	return err
}
