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
	wd = filepath.Join(wd, "..", "..")

	// Build the Docker image
	fmt.Println("Building Docker image...")
	dockerfilePath := filepath.Join(wd, "internal", "ipxe", "generate", "Dockerfile")

	buildCmd := exec.Command(
		"docker",
		"build",
		"--platform",
		"linux/arm64",
		"-f",
		dockerfilePath,
		"-t",
		"metal-boot-ipxe",
		"../../",
	)
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	if err := buildCmd.Run(); err != nil {
		log.Fatalf("Failed to build Docker image: %v", err)
	}

	// Create output directory
	outputDir := filepath.Join(wd, "internal", "ipxe", "binary")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Extract the iPXE EFI file
	fmt.Println("Extracting iPXE EFI file...")

	// Create a temporary container
	ctx := context.Background()
	createCmd := exec.CommandContext(ctx, "docker", "create", "metal-boot-ipxe")
	containerIDBytes, err := createCmd.Output()
	if err != nil {
		log.Fatalf("Failed to create container: %v", err)
	}
	containerID := string(containerIDBytes)[:12] // Take first 12 chars of container ID
	// Clean up container
	defer exec.CommandContext(ctx, "docker", "rm", containerID).Run()

	// Copy file from container
	outputFiles := []string{
		"undionly.kpxe",
		"ipxe.lkrn",
		"snp.efi",
		"ipxe.efi",
		"ipxe.iso",
	}

	for _, file := range outputFiles {
		outputFile := filepath.Join(outputDir, file)
		copyCmd := exec.CommandContext(
			ctx,
			"docker",
			"cp",
			containerID+":/output/"+file,
			outputFile,
		)
		if err := copyCmd.Run(); err != nil {
			// Clean up container
			log.Fatalf("Failed to copy iPXE EFI file: %v", err)
			break
		}

		// Check file size
		fileInfo, err := os.Stat(outputFile)
		if err != nil {
			log.Fatalf("Failed to stat output file: %v", err)
		}
		fmt.Printf("Successfully copied %s (%d bytes)\n", file, fileInfo.Size())
		fmt.Printf("iPXE build complete. EFI file available at: %s\n", outputFile)
	}
}
