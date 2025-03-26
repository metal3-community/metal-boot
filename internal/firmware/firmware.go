package firmware

type FirmwareManager interface {
	GetBootOrder() ([]string, error)
	SetBootOrder([]string) error

	UpdateFirmware() error
}
