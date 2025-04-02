# go-redfish-uefi

## TFTP Auto Serve UEFI Rpi based on mac prefix

## DHCP Proxy - Serve http ipxe config files from remote sidero metal

## IPMI/Redfish API for changing RPI.fd firmware vars in TFTP dir and power state

Serve read/write TFTP server with rpi4 uefi firmware for each device. Add Redfish api endpoints for fw var modification.

Use go-uefi to patch firmware: <https://github.com/Foxboron/go-uefi>

Optional: Provide power on/off via unifi poe switch.

# DESIGN

TFTP should exist here

Power state should be web hook

```mermaid
%% Set the layout to Left to Right (TB)
flowchart TB;
    
    %% Nova Boot Process
    subgraph Nova_System["Nova System"]
        direction TB;
        A["1. Nova boot"]:::step1 -->|Nova API| B["Message Queue"];
        B --> C["Nova Conductor"];
        C --> D["Nova Database"];
        
        B --> E["Nova Scheduler"];
        E --> F["2. Apply filters & find available compute host node"]:::step;
        
        F --> G["3. Compute Manager calls driver.spawn()"]:::step;
        G --> H["Nova Compute"];
        H --> I["4. Get info and claim bare metal node"]:::step;
        I --> J["7. Deploy bare metal node"]:::step;
    end

    %% Ironic Deployment Process
    subgraph Ironic_System["Ironic System"]
        direction TB;
        J --> K["Ironic API"];
        K --> L["Ironic Conductor"];
        L --> M["Ironic Database"];

        L --> P["Neutron"];
        P --> Q["6. Plug VIFs"]:::step;
        
        L --> R["Glance"];
        R --> S["5. Fetch images"]:::step;
        
        L --> N["8. Deploy (active boot loader)"]:::step;
        N --> O["10. Write image"]:::step;
    end

    %% Bare Metal Deployment Process
    subgraph Bare_Metal_System["Bare Metal Nodes"]
        direction TB;
        O --> T["PXE driver"];
        T --> U["Bare Metal Nodes"]:::baremetal;
        U --> V["IPMI driver"];

        O --> W["9. Power on bare metal node"]:::step;
        W --> X["11. Reboot"]:::step;
        X --> Y["12. Update status of bare metal node"]:::step;
    end

    %% Styling Definitions
    classDef step fill:#add8e6,stroke:#1c6ea4,stroke-width:2px;
    classDef step1 fill:#add8e6,stroke:#1c6ea4,stroke-width:2px,font-weight:bold;
    classDef baremetal fill:#d3d3d3,stroke:#7a7a7a,stroke-width:2px;
```

# What is this project?

Provides a complete netboot environment using DHCP proxy, TFTP, Http, Ipxe, EDK2 among many other things.

This project supplies DHCP bootp responses for a raspberry PI's initial DHCP broadcast. TFTP returns edk2 for a second stage bootloader. The RPI then boots into the firmware. We can control the firmware with EFI vars using the embedded `virt-fw-vars`. This allows you to set the boot sequence etc.

If the bootloader netboots, the server will provide `snp.efi`, an IPXE binary from the Tinkerbell project that supports embedding scripts on the fly.

It is possible to provide the boot order via IPXE using the `sanboot` command.

This project provides a fully featured Redfish server to manage multiple Raspberry pis. The server iterates through DHCP leases and discovers which POE ports are attached to which MAC address. This allows for automatic power management with zero configuration.

The `virt-fw-vars` binary supports the `--set-boot-uri` parameter, which allows easy iso loading for Redfish.

## Requirements

Should contain method for power on and off + state

## Other Options

Deploy sidero metal
