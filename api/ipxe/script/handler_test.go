package script

import (
	"net"
	"testing"
)

func TestGetMAC(t *testing.T) {
	tests := []struct {
		name        string
		urlPath     string
		expected    string
		expectError bool
	}{
		{
			name:     "legacy path pattern",
			urlPath:  "/d8:3a:dd:5a:44:36/auto.ipxe",
			expected: "d8:3a:dd:5a:44:36",
		},
		{
			name:     "new v1/boot path pattern",
			urlPath:  "/v1/boot/d8:3a:dd:5a:44:36/boot.ipxe",
			expected: "d8:3a:dd:5a:44:36",
		},
		{
			name:     "new v1/boot path pattern with additional segments",
			urlPath:  "/v1/boot/d8-3a-dd-5a-44-36/boot.ipxe",
			expected: "d8-3a-dd-5a-44-36",
		},
		{
			name:        "invalid MAC in legacy pattern",
			urlPath:     "/invalid-mac/auto.ipxe",
			expectError: true,
		},
		{
			name:        "invalid MAC in v1/boot pattern",
			urlPath:     "/v1/boot/invalid-mac/boot.ipxe",
			expectError: true,
		},
		{
			name:        "malformed v1/boot path",
			urlPath:     "/v1/boot/",
			expectError: true,
		},
		{
			name:        "wrong v1 path",
			urlPath:     "/v1/other/d8:3a:dd:5a:44:36/boot.ipxe",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mac, err := getMAC(tt.urlPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for path %s, but got none", tt.urlPath)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for path %s: %v", tt.urlPath, err)
				return
			}

			expectedMAC, err := net.ParseMAC(tt.expected)
			if err != nil {
				t.Errorf("Invalid expected MAC in test case: %v", err)
				return
			}

			if mac.String() != expectedMAC.String() {
				t.Errorf("Expected MAC %s, got %s for path %s", expectedMAC.String(), mac.String(), tt.urlPath)
			}
		})
	}
}