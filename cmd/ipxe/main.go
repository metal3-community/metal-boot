package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	fmt.Println("Building iPXE for ARM64 EFI...")

	// Get the current working directory
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}

	// Build the Docker image
	fmt.Println("Building Docker image...")
	dockerfilePath := filepath.Join(wd, "hack", "Dockerfile.ipxe")

	buildCmd := exec.Command("docker", "build", "-f", dockerfilePath, "-t", "pibmc-ipxe", ".")
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	if err := buildCmd.Run(); err != nil {
		log.Fatalf("Failed to build Docker image: %v", err)
	}

	// Create output directory
	outputDir := filepath.Join(wd, "internal", "firmware", "edk2")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Extract the iPXE EFI file
	fmt.Println("Extracting iPXE EFI file...")

	// Create a temporary container
	ctx := context.Background()
	createCmd := exec.CommandContext(ctx, "docker", "create", "pibmc-ipxe")
	containerIDBytes, err := createCmd.Output()
	if err != nil {
		log.Fatalf("Failed to create container: %v", err)
	}
	containerID := string(containerIDBytes)[:12] // Take first 12 chars of container ID

	// Copy file from container
	outputFile := filepath.Join(outputDir, "snp-uboot.efi")
	copyCmd := exec.CommandContext(ctx, "docker", "cp", containerID+":/output/snp.efi", outputFile)
	if err := copyCmd.Run(); err != nil {
		// Clean up container
		exec.CommandContext(ctx, "docker", "rm", containerID).Run()
		log.Fatalf("Failed to copy iPXE EFI file: %v", err)
	}

	// Clean up container
	exec.CommandContext(ctx, "docker", "rm", containerID).Run()

	// Check file size
	fileInfo, err := os.Stat(outputFile)
	if err != nil {
		log.Fatalf("Failed to stat output file: %v", err)
	}

	fmt.Printf("Successfully copied snp-uboot.efi (%d bytes)\n", fileInfo.Size())
	fmt.Printf("iPXE build complete. EFI file available at: %s\n", outputFile)
}
