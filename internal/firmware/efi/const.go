package efi

const (
	EFI_VARIABLE_NON_VOLATILE                          = 0x00000001
	EFI_VARIABLE_BOOTSERVICE_ACCESS                    = 0x00000002
	EFI_VARIABLE_RUNTIME_ACCESS                        = 0x00000004
	EFI_VARIABLE_HARDWARE_ERROR_RECORD                 = 0x00000008
	EFI_VARIABLE_AUTHENTICATED_WRITE_ACCESS            = 0x00000010
	EFI_VARIABLE_TIME_BASED_AUTHENTICATED_WRITE_ACCESS = 0x00000020
	EFI_VARIABLE_APPEND_WRITE                          = 0x00000040

	LOAD_OPTION_ACTIVE          = 0x00000001
	LOAD_OPTION_FORCE_RECONNECT = 0x00000002
	LOAD_OPTION_HIDDEN          = 0x00000008
	LOAD_OPTION_CATEGORY        = 0x00001F00
	LOAD_OPTION_CATEGORY_BOOT   = 0x00000000
	LOAD_OPTION_CATEGORY_APP    = 0x00000100

	Boot                = "Boot"
	BootOrder           = "BootOrder"
	BootPrefix          = "Boot"
	BootNext            = "BootNext"
	EFI_GLOBAL_VARIABLE = "8be4df61-93ca-11d2-aa0d-00e098032b8c"

	Ffs          = "8c8ce578-8a3d-4f1c-9935-896185c32dd3"
	NvData       = "fff12b8d-7696-4c8b-a985-2747075b4f50"
	AuthVars     = "aaf32c78-947b-439a-a180-2e144ec37792"
	LzmaCompress = "ee4e5898-3914-4259-9d6e-dc7bd79403cf"
	ResetVector  = "1ba0062e-c779-4582-8566-336ae8f78f09"

	OvmfPeiFv = "6938079b-b503-4e3d-9d24-b28337a25806"
	OvmfDxeFv = "7cb8bdc9-f8eb-4f34-aaea-3ee4af6516a1"

	EfiGlobalVariable              = "8be4df61-93ca-11d2-aa0d-00e098032b8c"
	EfiImageSecurityDatabase       = "d719b2cb-3d3a-4596-a3bc-dad00e67656f"
	EfiSecureBootEnableDisable     = "f0a30bc7-af08-4556-99c4-001009c93a44"
	EfiCustomModeEnable            = "c076ec0c-7028-4399-a072-71ee5c448b9f"
	EfiDhcp6ServiceBindingProtocol = "9fb9a8a1-2f4a-43a6-889c-d0f7b6c47ad5"
	EfiIp6ConfigProtocol           = "937fe521-95ae-4d1a-8929-48bcd90ad31a"

	EfiCertX509   = "a5c059a1-94e4-4aa7-87b5-ab155c2bf072"
	EfiCertSha256 = "c1c41626-504c-4092-aca9-41f936934328"
	EfiCertPkcs7  = "4aafd29d-68df-49ee-8aa9-347d375665a7"

	MicrosoftVendor       = "77fa9abd-0359-4d32-bd60-28f4e78f784b"
	OvmfEnrollDefaultKeys = "a0baa8a3-041d-48a8-bc87-c36d121b5e3d"
	Shim                  = "605dab50-e046-4300-abb6-3dd810dd8b23"
	LoaderInfo            = "4a67b082-0a4c-41cf-b6c7-440b29bb8c4f"

	OvmfGuidList          = "96b582de-1fb2-45f7-baea-a366c55a082d"
	SevHashTableBlock     = "7255371f-3a3b-4b04-927b-1da6efa8d454"
	SevSecretBlock        = "4c2eb361-7d9b-4cc3-8081-127c90d3d294"
	SevProcessorReset     = "00f771de-1a7e-4fcb-890e-68c77e2fb44e"
	OvmfSevMetadataOffset = "dc886566-984a-4798-a75e-5585a7bf67cc"
	TdxMetadataOffset     = "e47a6535-984a-4798-865e-4685a7bf8ec2"

	FwMgrCapsule  = "6dcbd5ed-e82d-4c44-bda1-7194199ad92a"
	SignedCapsule = "4a3ca68b-7723-48fb-803d-578cc1fec44d"

	NotValid = "ffffffff-ffff-ffff-ffff-ffffffffffff"
)

// For getting categories
const LOAD_OPTION_CATEGORY_MASK uint32 = 0x1F000000

// EFI variable attributes constants
const (
	EfiAttrBootserviceAccess = EFI_VARIABLE_BOOTSERVICE_ACCESS
	EfiAttrRuntimeAccess     = EFI_VARIABLE_RUNTIME_ACCESS
	EfiAttrNonVolatile       = EFI_VARIABLE_NON_VOLATILE
)
