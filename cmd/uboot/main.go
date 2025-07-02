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
	fmt.Println("Building U-Boot for Raspberry Pi...")

	// Get the current working directory
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}

	// Build the Docker image
	fmt.Println("Building Docker image...")
	dockerfilePath := filepath.Join(wd, "hack", "Dockerfile.uboot")

	buildCmd := exec.Command("docker", "build", "-f", dockerfilePath, "-t", "pibmc-uboot", ".")
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

	// Extract the U-Boot binary
	fmt.Println("Extracting U-Boot binary...")

	// Create a temporary container
	ctx := context.Background()
	createCmd := exec.CommandContext(ctx, "docker", "create", "pibmc-uboot")
	containerIDBytes, err := createCmd.Output()
	if err != nil {
		log.Fatalf("Failed to create container: %v", err)
	}
	containerID := string(containerIDBytes)[:12] // Take first 12 chars of container ID

	// Copy file from container
	outputFile := filepath.Join(outputDir, "u-boot.bin")
	copyCmd := exec.CommandContext(
		ctx,
		"docker",
		"cp",
		containerID+":/output/u-boot.bin",
		outputFile,
	)
	if err := copyCmd.Run(); err != nil {
		// Clean up container
		exec.CommandContext(ctx, "docker", "rm", containerID).Run()
		log.Fatalf("Failed to copy U-Boot binary: %v", err)
	}

	// Clean up container
	exec.CommandContext(ctx, "docker", "rm", containerID).Run()

	// Check file size
	fileInfo, err := os.Stat(outputFile)
	if err != nil {
		log.Fatalf("Failed to stat output file: %v", err)
	}

	fmt.Printf("Successfully copied u-boot.bin (%d bytes)\n", fileInfo.Size())
	fmt.Printf("U-Boot build complete. Binary available at: %s\n", outputFile)
}
