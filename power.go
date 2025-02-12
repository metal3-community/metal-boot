package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ubiquiti-community/go-unifi/unifi"
)

var client *lazyClient = (*lazyClient)(nil)

func discover(ctx context.Context, mac string) error {
	activeClients, err := client.ListActiveClients(ctx)

	if err != nil {
		return fmt.Errorf("error listing active clients: %v", err)
	}

	for _, c := range activeClients {
		fmt.Printf("Client: %v\n", client)

		if c.ID == mac {
			fmt.Printf("Client found: %v\n", c)

			fmt.Printf("Client Port: %d\n", c.SwPort)

			return nil
		}
	}

	return fmt.Errorf("client with MAC Address %s not found", mac)
}

func getPort(ctx context.Context, macAddress string, portIdx string) (deviceId string, port unifi.DevicePortOverrides, err error) {
	deviceId = ""

	p, err := strconv.Atoi(portIdx)
	if err != nil {
		err = fmt.Errorf("error getting integer value from port %s: %v", portIdx, err)
		return
	}

	devices, err := client.ListDevice(ctx, "default")
	if err != nil {
		err = fmt.Errorf("error listing devices: %v", err)
		return
	}

	for _, dev := range devices {
		for _, pd := range dev.PortOverrides {
			if pd.PortIDX == p {
				port = pd
				break
			}
		}
	}

	dev, err := client.GetDeviceByMAC(ctx, "default", macAddress)
	if err != nil {
		err = fmt.Errorf("error getting device by MAC Address %s: %v", macAddress, err)
		return
	}

	deviceId = dev.ID

	for _, pd := range dev.PortOverrides {
		if pd.PortIDX == p {
			port = pd
			break
		}
	}

	return
}

func setPortPower(ctx context.Context, macAddress string, portIdx string, state string) error {
	p, err := strconv.Atoi(portIdx)
	if err != nil {
		return fmt.Errorf("error getting integer value from port %s: %v", portIdx, err)
	}

	dev, err := client.GetDeviceByMAC(ctx, "default", macAddress)
	if err != nil {
		return fmt.Errorf("error getting device by MAC Address %s: %v", macAddress, err)
	}

	for i, pd := range dev.PortOverrides {
		if pd.PortIDX == p {
			switch state {
			case "on":
				if pd.PoeMode == "auto" {
					return nil
				}
				dev.PortOverrides[i].PoeMode = "auto"
				break
			case "off":
				if pd.PoeMode == "off" {
					return nil
				}
				dev.PortOverrides[i].PoeMode = "off"
				break
			}
		}
	}

	_, err = client.UpdateDevice(ctx, "default", dev)

	if err != nil {
		return fmt.Errorf("error updating device: %v", err)
	}

	return nil
}

func GetPower(ctx context.Context, macAddress string, portIdx string) (state string, err error) {
	_, port, err := getPort(ctx, macAddress, portIdx)
	if err != nil {
		fmt.Printf("error setting power on for MAC Address %s, Port Index %s: %v", macAddress, portIdx, err)
		return
	}

	mode := port.PoeMode

	if mode == "auto" {
		state = "on"
	} else if mode == "off" {
		state = "off"
	}

	return
}
