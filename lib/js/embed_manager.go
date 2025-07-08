package js

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

//go:embed all:bundle
var embeddedBundle embed.FS

var (
	bundlePath string
	setupOnce  sync.Once
	setupErr   error
)

// GetNodeBundlePath ensures the embedded Node.js bundle is extracted to a temporary
// directory and returns the path to the bootstrap script. This is done only once.
func GetNodeBundlePath() (string, error) {
	setupOnce.Do(func() {
		// Define a path in the user's temp directory.
		tempDir := os.TempDir()
		targetPath := filepath.Join(tempDir, "go-cloudscraper-bundle")
		versionFile := filepath.Join(targetPath, "version.txt") // To check if extraction is needed.

		// A simple "version" for our bundle. Change this if you update bootstrap.js or dependencies.
		const bundleVersion = "1.0.0"

		// Check if the correct version is already extracted.
		existingVersion, err := os.ReadFile(versionFile)
		if err == nil && string(existingVersion) == bundleVersion {
			bundlePath = targetPath // It's already there and correct.
			return
		}

		// If not, extract it.
		// First, clean up any old directory.
		_ = os.RemoveAll(targetPath)

		// Walk the embedded FS and write files to the temp directory.
		err = fs.WalkDir(embeddedBundle, "bundle", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			// Get the relative path to create the correct structure on disk.
			relPath, _ := filepath.Rel("bundle", path)
			diskPath := filepath.Join(targetPath, relPath)

			if d.IsDir() {
				return os.MkdirAll(diskPath, 0755)
			}

			src, err := embeddedBundle.Open(path)
			if err != nil {
				return err
			}
			defer src.Close()

			dst, err := os.Create(diskPath)
			if err != nil {
				return err
			}
			defer dst.Close()

			_, err = io.Copy(dst, src)
			return err
		})

		if err != nil {
			setupErr = fmt.Errorf("failed to extract embedded node bundle: %w", err)
			return
		}

		// Write the version file so we don't extract again next time.
		if err := os.WriteFile(versionFile, []byte(bundleVersion), 0644); err != nil {
			setupErr = fmt.Errorf("failed to write bundle version file: %w", err)
			return
		}

		bundlePath = targetPath
	})

	return filepath.Join(bundlePath, "bootstrap.js"), setupErr
}
