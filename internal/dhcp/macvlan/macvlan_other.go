//go:build !linux

package macvlan

import (
	"errors"

	"github.com/go-logr/logr"
)

var errUnsupported = errors.New("macvlan interface setup is only supported on linux")

// InterfaceName is unsupported off linux.
func InterfaceName() (string, error) { return "", errUnsupported }

// Setup is unsupported off linux.
func Setup(_ logr.Logger, _, _ string) (func(), error) { return nil, errUnsupported }

// DefaultRouteInterface is unsupported off linux.
func DefaultRouteInterface() (string, error) { return "", errUnsupported }
