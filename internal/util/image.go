package util

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

const defaultImageRef = "ghcr.io/metal3-community/ironic-python-agent-image:latest"

func DownloadIpaImages(rootpath string) error {
	architectures := []string{"amd64", "arm64"}
	for _, arch := range architectures {
		if err := downloadImage(rootpath, arch); err != nil {
			return fmt.Errorf("failed to download image for %s: %w", arch, err)
		}
	}
	return nil
}

func downloadImage(rootpath string, arch string) error {
	imageRef := defaultImageRef

	plt := v1.Platform{
		Architecture: arch,
		OS:           "linux",
	}

	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return fmt.Errorf("failed to parse image reference: %w", err)
	}

	// Pull image for specific platform
	img, err := crane.Pull(ref.Name(), crane.WithPlatform(&plt))
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}

	// Extract to rootfs directory
	rootfs := filepath.Join(rootpath, arch)
	if err := os.MkdirAll(rootfs, 0o755); err != nil {
		return fmt.Errorf("failed to create rootfs directory: %w", err)
	}

	if err := extractImage(img, rootfs); err != nil {
		return fmt.Errorf("failed to extract image: %w", err)
	}

	return nil
}

func extractImage(img v1.Image, root string) error {
	layers, err := img.Layers()
	if err != nil {
		return err
	}

	for _, layer := range layers {
		r, err := layer.Uncompressed()
		if err != nil {
			return err
		}
		defer r.Close()

		if err := untar(r, root); err != nil {
			return err
		}
	}
	return nil
}

func untar(r io.Reader, dest string) error {
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, hdr.Name)
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(hdr.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			if err := os.Symlink(hdr.Linkname, target); err != nil && !os.IsExist(err) {
				return err
			}
		}
	}
	return nil
}
