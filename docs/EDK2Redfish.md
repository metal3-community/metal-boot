# UEFI Redfish EDK2 Implementation

[](#uefi-redfish-edk2-implementation)[](#uefi-redfish-edk2-implementation)UEFI Redfish EDK2 Implementation
==========================================================================================================

[](#introduction)[](#introduction)Introduction
----------------------------------------------

UEFI Redfish EDK2 solution is an efficient and secure solution for the end-users to remote configure (in Out-of-band) UEFI platform configurations by leveraging the Redfish RESTful API. It's simple for end-users to access the configurations of UEFI firmware which have the equivalent properties defined in Redfish schema.

Below are the block diagrams of UEFI Redfish EDK2 Implementation. _**[EDK2 Redfish Foundation\[1\]](#[0])**_ in the lower part of the figure refers to the EDK2 Redfish Foundation, which provides the fundamental EDK2 drivers to communicate with Redfish service _**([\[19\]](#[0]) in the above figure)**_. The Redfish service could be implemented in BMC to manage the system, or on the network for managing multiple systems.

_**[EDK2 Redfish Client\[2\]](#[0])**_ in the upper part of the figure refers to the EDK2 Redfish client, which is the EDK2 Redfish application used to configure platform configurations by consuming the Redfish properties. The EDK2 Redfish client can also provision the UEFI platform-owned Redfish properties, consume and update Redfish properties as well. The _**[EDK2 Redfish Feature DXE Drivers \[17\]](https://github.com/tianocore/edk2-redfish-client/blob/main/RedfishClientPkg/Readme.md)**_ is the project in tianocore/edk2-redfish-client repository. Each EDK2 Redfish Feature DXE Driver is designed to communicate with the particular Redfish data model defined in the Redfish schema _(e.g. Edk2RedfishBiosDxe driver manipulates the properties defined in Redfish BIOS data model)_.

[](#edk2-redfish-implementation-diagrams)[](#edk2-redfish-implementation-diagrams)EDK2 Redfish Implementation Diagrams
----------------------------------------------------------------------------------------------------------------------

![UEFI Redfish Implementation](https://github.com/tianocore/edk2/blob/master/RedfishPkg/Documents/Media/RedfishDriverStack.svg?raw=true)

[](#efi-edk2-redfish-driver-stack)[](#efi-edk2-redfish-driver-stack)EFI EDK2 Redfish Driver Stack
-------------------------------------------------------------------------------------------------

Below are the EDK2 drivers implemented on EDK2,

### [](#edk2-redfish-host-interface-dxe-driver-6)[](#edk2-redfish-host-interface-dxe-driver-6)EDK2 Redfish Host Interface DXE Driver _**[\[6\]](#[0])**_

The abstract EDK2 DXE driver to create SMBIOS type 42 record through EFI SMBIOS protocol according to the device descriptor and protocol type data (defined in SMBIOS type 42h _**[\[7\]](#[0])**_) provided by platform level Redfish host interface library. On EDK2 open source implementation (**EmulatorPkg**), SMBIOS type 42 data is retrieved from EFI variables created by RedfishPlatformConfig.efi _**[\[20\]](#[0])**_ under EFI shell. OEM may provide its own PlatformHostInterfaceLib _**[\[11\]](#[0])**_ instance for the platform-specific implementation.

### [](#edk2-refish-credential-dxe-driver-5)[](#edk2-refish-credential-dxe-driver-5)EDK2 Refish Credential DXE Driver _**[\[5\]](#[0])**_

The abstract DXE driver which incorporates with RedfishPlatformCredentialLib _**[\[10\]](#[0])**_ to acquire the credential of Redfish service. On edk2 EmulatorPkg implementation, the credential is hardcoded using the fixed Account/Password in order to connect to Redfish service established by [Redfish Profile Simulator](https://github.com/DMTF/Redfish-Profile-Simulator). OEM may provide its own RedfishPlatformCredentialLib instance for the platform-specific implementation.

### [](#efi-rest-ex-uefi-driver-for-redfish-service-4)[](#efi-rest-ex-uefi-driver-for-redfish-service-4)EFI REST EX UEFI Driver for Redfish service _**[\[4\]](#[0])**_

This is the network-based driver instance of EFI\_REST\_EX protocol [(UEFI spec 2.8, section 29.7.2)](http://uefi.org/specifications) for communicating with Redfish service using the HTTP protocol. OEM may have its own EFI REST EX UEFI Driver instance on which the underlying transport to Redfish service could be proprietary.

### [](#efi-redfish-discover-uefi-driver-3)[](#efi-redfish-discover-uefi-driver-3)EFI Redfish Discover UEFI Driver _**[\[3\]](#[0])**_

EFI Redfish Discover Protocol implementation (UEFI spec 2.8, section 31.1). Only support Redfish service discovery through Redfish Host Interface. The Redfish service discovery using SSDP over UDP _**[\[18\]](#[0])**_ is not implemented at the moment.

### [](#efi-rest-json-structure-dxe-driver-9)[](#efi-rest-json-structure-dxe-driver-9)EFI REST JSON Structure DXE Driver _**[\[9\]](#[0])**_

EFI REST JSON Structure DXE implementation (UEFI spec 2.8, section 29.7.3). This could be used by EDK2 Redfish Feature DXE Drivers _**[\[17\]](#[0])**_. The EDK2 Redfish feature drivers manipulate platform-owned Redfish properties in C structure format and convert them into the payload in JSON format through this protocol. This driver leverages the effort of [Redfish Schema to C Generator](https://github.com/DMTF/Redfish-Schema-C-Struct-Generator) to have the “C Structure” <-> “JSON” conversion.

### [](#edk2-redfish-config-handler-uefi-driver-15)[](#edk2-redfish-config-handler-uefi-driver-15)EDK2 Redfish Config Handler UEFI Driver _**[\[15\]](#[0])**_

This is the centralized manager of EDK2 Redfish feature drivers, it initiates EDK2 Redfish feature drivers by invoking init() function of EDK2 Redfish Config Handler Protocol _**[\[16\]](#[0])**_ installed by each EDK2 Redfish feature driver. EDK2 Redfish Config Handler driver is an UEFI driver which has the dependency with EFI REST EX protocol and utilizes EFI Redfish Discover protocol to discover Redfish service that manages this system.

### [](#edk2-content-coding-library-12)[](#edk2-content-coding-library-12)EDK2 Content Coding Library _**[\[12\]](#[0])**_

The library is incorporated with RedfishLib _**[\[13\]](#[0])**_ to encode and decode Redfish JSON payload. This is the platform library to support HTTP Content-Encoding/Accept-Encoding headers. EumlatorPkg use the NULL instance of this library because [Redfish Profile Simulator](https://github.com/DMTF/Redfish-Profile-Simulator) supports neither HTTP Content-Encoding header on the payload returned to Redfish client nor HTTP Accept-Encoding header.

[](#other-open-source-projects)[](#other-open-source-projects)Other Open Source Projects
----------------------------------------------------------------------------------------

The following libraries are the wrappers of other open source projects used in RedfishPkg

* **RedfishPkg\\PrivateLibrary\\RedfishLib** _**[\[13\]](#[0])**_ This is the wrapper of open source project _**[libredfish](https://github.com/DMTF/libredfish)**_, which is the library to initialize the connection to Redfish service with the proper credential and execute Create/Read/Update/Delete (CRUD) HTTP methods on Redfish properties.

* **RedfishPkg\\Library\\JsonLib** _**[\[14\]](#[0])**_ This is the wrapper of open source project _**[Jansson](https://digip.org/jansson)**_, which is the library that provides APIs to manipulate JSON payload.

[](#platform-components)[](#platform-components)Platform Components
-------------------------------------------------------------------

### [](#edk2-emulatorpkg)[](#edk2-emulatorpkg)**EDK2 EmulatorPkg**

![EDK2 EmulatorPkg Figure](https://github.com/tianocore/edk2/blob/master/RedfishPkg/Documents/Media/EmualtorPlatformLibrary.svg?raw=true)

* **RedfishPlatformCredentialLib**  
    The EDK2 Emulator platform implementation of acquiring credential to build up the communication between UEFI firmware and Redfish service. _**[\[10\]](#[0])**_

    The Redfish credential is hardcoded in the EmulatorPkg RedfishPlatformCredentialLib. The credential is used to access to the Redfish service hosted by [Redfish Profile Simulator](https://github.com/DMTF/Redfish-Profile-Simulator).

* **RedfishPlatformHostInterfaceLib**  
    EDK2 Emulator platform implementation which provides the information of building up SMBIOS type 42h record. _**[\[11\]](#[0])**_

    EmulatorPkg RedfishPlatformHostInterfaceLib library consumes the EFI Variable which is created by [RedfishPlatformConfig EFI application](https://github.com/tianocore/edk2/tree/master/EmulatorPkg/Application/RedfishPlatformConfig). RedfishPlatformConfig EFI application stores not all of SMBIOS Type42 record information but the necessary network properties of Redfish Host Interface in EFI Variable.

### [](#Platform-with-BMC-and-the-BMC_Exposed-USB-Network-Device)[](#platform-with-bmc-and-the-bmc_exposed-usb-network-device)**Platform with BMC and the BMC-Exposed USB Network Device**

![Platform with BMC Figure](https://github.com/tianocore/edk2/blob/master/RedfishPkg/Documents/Media/BmcExposedUsbNic.svg?raw=true)

Server platform with BMC as the server management entity may expose the [USB Network Interface Device (NIC)](https://www.usb.org/document-library/class-definitions-communication-devices-12) to the platform, which is so called the in-band host-BMC transport interface. The USB NIC exposed by BMC is usually a link-local network device which has two network endpoints at host and BMC ends. The endpoint at host side is connected to the platform USB port, and it is enumerated by edk2 USB BUS driver. The edk2 USB NIC driver then produces the **EFI Network Interface Identifier Protocol** _(EFI\_NETWORK\_INTERFACE\_IDENTIFIER\_PROTOCOL\_GUID\_31)_ and connected by EFI Simple Network Protocol (SNP) driver and the upper layer edk2 network drivers. The applications can then utilize the network stack built up on top of USB NIC to communicate with Redfish service hosted by BMC.  
BMC-exposed USB NIC is mainly designed for the communication between host and BMC-hosted Redfish service. BMC-exposed USB NIC can be public to host through [Redfish Host Interface Specification](https://www.dmtf.org/sites/default/files/standards/documents/DSP0270_1.3.0.pdf) and discovered by edk2 Redfish discovery driver. The [Redfish Host Interface Specification](https://www.dmtf.org/sites/default/files/standards/documents/DSP0270_1.3.0.pdf) describes the dedicated host interface between host and BMC. The specification follows the [SMBIOS Type 42 format](https://www.dmtf.org/sites/default/files/standards/documents/DSP0134_3.6.0.pdf) and defines the host interface as:

* Network Host Interface type (40h)
* Redfish over IP Protocol (04h)

![Platform BMC Library Figure](https://github.com/tianocore/edk2/blob/master/RedfishPkg/Documents/Media/PlatformWihtBmcLibrary.svg?raw=true)

* **RedfishPlatformCredentialLib**  
    RedfishPlatformCredentialLib library instance on the platform with BMC uses Redfish Credential Bootstrapping IPMI commands defined in [Redfish Host Interface Specification](https://www.dmtf.org/sites/default/files/standards/documents/DSP0270_1.3.0.pdf) to acquire the Redfish credential from BMC. edk2 Redfish firmware then uses this credential to access to Redfish service hosted by BMC.

* **RedfishPlatformHostInterfaceLib**  
    BMC-exposed USB NIC, a IPMI Message channel is reported by BMC as a “IPMB-1.0” protocol and “802.3 LAN” medium channel. edk2 firmware can issue a series of IPMI commands to acquire the channel application information (NetFn App), transport network properties (NetFn Transport) and other necessary information to build up the SMBIOS type 42h record. In order to recognize the specific BMC-exposed USB NIC in case the platform has more than one USB NIC devices attached, the MAC address specified in the EFI Device Path Protocol of SNP EFI handle is used to match with the MAC address of IPMI message channel. Due to the network information such as the MAC address, IP address, Subnet mask and Gateway are assigned by BMC, edk2 Redfish implementation needs a programmatic way to recognize the BMC-exposed USB NIC.

  * **MAC address:** Searching for the BMC-exposed USB NIC  
        The last byte of host-end USB NIC MAC address is the last byte of BMC-end USB NIC MAC address minus 1. RedfishPlatformHostInterfaceLib issues the NetFn Transport IPMI command to get the MAC address of each channel and checks them with the MAC address specified in the EFI Device Path Protocol.  

        **_For example:_**  
        BMC-end USB NIC MAC address: 11-22-33-44-55-00  
        Host-end USB NIC MAC address: 11-22-33-44-55-ff

  * **IP Address:** Acquiring the host-end USB NIC IP Address  
        The last byte of host-end USB NIC IPv4 address is the last byte of BMC-end USB NIC IPv4 address minus 1.

        **_For example:_**  
        BMC-end USB NIC IPv4 address: 165.10.0.10  
        Host-end USB NIC IPv4 address: 165.10.0.9

  * **Other Network Properties**:  
        Due to the host-end USB NIC and BMC-end USB NIC is a link-local network. Both of them have the same network properties such as subnet mask and gateway. RedfishPlatformHostInterfaceLib issues the NetFn Transport IPMI command to get the network properties of BMC-end USB NIC and apply it on host-end USB NIC.

* **IPMI Commands that Used to Build up Redfish Host Interface**

    Standard IPMI commands those are used to build up the Redfish Host Interface for BMC-exposed USB NIC. The USB NIC exposed by BMC must be reported as one of the message channels as 802.3 LAN/IPMB 1.0 message channel.

    IPMI NetFn

    IPMI Command

    Purpose

    Corresponding Host Interface Field

    App  
    (0x06)

    0x42

    Check the message channel's medium type and protocol.  
    Medium: 802.3 LAN  
    Protocol: IPMB 1.0

    None

    Transport  
    (0x0C)

    0x02

    Get MAC address of message channel. Used to match with the MAC address populated in EFI Device Path of network device

    None

    Group Ext  
    (0x2C)

    Group Extension ID: 0x52  
    Command: 0x02

    Check if Redfish bootstrap credential is supported or not.

    In Device Descriptor Data, Credential Bootstrapping Handle

    Transport  
    (0x0C)

    Command: 0x02  
    Parameter: 0x04

    Get BMC-end message channel IP address source

    In Protocol Specific Record Data  
    \- Host IP Assignment Type  
    \- Redfish Service IP Discovery Type  
    \- Generate the Host-side IP address

    Transport  
    (0x0C)

    Command: 0x02  
    Parameter: 0x03

    Get BMC-end message channel IPv4 address

    In Protocol Specific Record Data  
    \- Host IP Address Format  
    \- Host IP Address

    Transport  
    (0x0C)

    Command: 0x02  
    Parameter: 0x06

    Get BMC-end message channel IPv4 subnet mask

    In Protocol Specific Record Data  
    \- Host IP Mask  
    \- Redfish Service IP Mask

    Transport  
    (0x0C)

    Command: 0x02  
    Parameter: 0x12

    Get BMC-end message channel gateway IP address

    None, used to configure edk2 network configuration

    Transport  
    (0x0C)

    Command: 0x02  
    Parameter: 0x14

    Get BMC-end message channel VLAN ID

    In Protocol Specific Record Data  
    Redfish Service VLAN ID

****NOTE****

Current RedfishPlatformHostInterfaceLib implementation of BMC-exposed USB NIC can only support IPv4 address format.

[](#miscellaneous)[](#miscellaneous)Miscellaneous
--------------------------------------------------

* **EFI Shell Application** RedfishPlatformConfig.exe is an EFI Shell application used to set up the Redfish service information for the EDK2 Emulator platform. The information such as IP address, subnet, and port.

 For example, run shell command "RedfishPlatformConfig.efi -s 192.168.10.101 255.255.255.0 192.168.10.123 255.255.255.0", which means
   the source IP address is 192.168.10.101, and the Redfish Server IP address is 192.168.10.123.

* **Redfish Profile Simulator** Refer to [Redfish Profile Simulator](https://github.com/DMTF/Redfish-Profile-Simulator) to set up the Redfish service. We are also in the progress to contribute bug fixes and enhancements to the mainstream Redfish Profile Simulator in order to incorporate with EDK2 Redfish solution.

[](#connect-to-redfish-service-on-edk2-emulator-platform)[](#connect-to-redfish-service-on-edk2-emulator-platform)Connect to Redfish Service on EDK2 Emulator Platform
----------------------------------------------------------------------------------------------------------------------------------------------------------------------

1. Install the WinpCap and copy [SnpNt32Io.dll](https://github.com/tianocore/edk2-NetNt32Io) to the building directory of the Emulator platform. This is the emulated network interface for EDK2 Emulator Platform.

e.g. %WORKSPACE%/Build/EmulatorX64/DEBUG\_VS2015x86/X64

2. Enable below macros in EmulatorPkg.dsc

NETWORK\_SNP\_ENABLE \= TRUE
NETWORK\_HTTP\_ENABLE \= TRUE
NETWORK\_IP6\_ENABLE \= TRUE
SECURE\_BOOT\_ENABLE \= TRUE
REDFISH\_ENABLE \= TRUE

3. Allow HTTP connection Enable below macro to allow HTTP connection on EDK2 network stack for connecting to [Redfish Profile Simulator](https://github.com/DMTF/Redfish-Profile-Simulator) becasue Redfish Profile Simulator doesn't support HTTPS.

NETWORK\_ALLOW\_HTTP\_CONNECTIONS \= TRUE

4. Assign the correct MAC Address Assign the correct MAC address of the network interface card emulated by WinpCap.

* Rebuild EmulatorPkg and boot to EFI shell once SnpNt32Io.dll is copied to the building directory and the macros mentioned in #2 are all set to TURE.
* Execute the EFI shell command “ifconfig -l” under EFI shell and look for MAC address information, then assign the MAC address to below PCD.

gEfiRedfishPkgTokenSpaceGuid.PcdRedfishRestExServiceDevicePath.DevicePath|{DEVICE\_PATH("MAC(000000000000,0x1)")}

* Assign the network adapter instaleld on the host (working machine) that will be emulated as the network interface in edk2 Emulator.

#

 \# For Windows based host, use a number to refer to network adapter

#

 gEmulatorPkgTokenSpaceGuid.PcdEmuNetworkInterface|L"1"
 or

#

 \# For Linux based host, use the device name of network adapter

#

 gEmulatorPkgTokenSpaceGuid.PcdEmuNetworkInterface|L"en0"

5. Configure the Redfish service on the EDK2 Emulator platform

Execute RedfishPlatformConfig.efi under EFI shell to configure the Redfish service information. The EFI variables are created for storing Redfish service information and is consumed by RedfishPlatformHostInterfaceLib under EmulatorPkg.

[](#related-materials)[](#related-materials)Related Materials
-------------------------------------------------------------

1. [DSP0270](https://www.dmtf.org/sites/default/files/standards/documents/DSP0270_1.3.0.pdf) - Redfish Host Interface Specification, 1.3.0
2. [DSP0266](https://www.dmtf.org/sites/default/files/standards/documents/DSP0266_1.12.0.pdf) - Redfish Specification, 1.12.0
3. Redfish Schemas - [https://redfish.dmtf.org/schemas/v1/](https://redfish.dmtf.org/schemas/v1/)
4. SMBIOS - [https://www.dmtf.org/sites/default/files/standards/documents/DSP0134\_3.6.0.pdf](https://www.dmtf.org/sites/default/files/standards/documents/DSP0134_3.6.0.pdf)
5. USB CDC - [https://www.usb.org/document-library/class-definitions-communication-devices-12](https://www.usb.org/document-library/class-definitions-communication-devices-12)
6. UEFI Specification - [http://uefi.org/specifications](http://uefi.org/specifications)

[](#The-Contributors)[](#the-contributors)The Contributors
