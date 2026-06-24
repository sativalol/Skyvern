package obfuscator

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed tools/*
var ToolsFS embed.FS

// RestoreTools writes the embedded tools directory to the given destination directory.
func RestoreTools(destDir string) error {
	return fs.WalkDir(ToolsFS, "tools", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Calculate target path relative to the destination directory
		relPath, err := filepath.Rel("tools", path)
		if err != nil {
			return err
		}
		
		targetPath := filepath.Join(destDir, "tools", relPath)

		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		data, err := ToolsFS.ReadFile(path)
		if err != nil {
			return err
		}

		// Only write the file if it doesn't already exist or has different size
		if info, err := os.Stat(targetPath); err == nil {
			if info.Size() == int64(len(data)) {
				return nil
			}
		}

		return os.WriteFile(targetPath, data, 0644)
	})
}
