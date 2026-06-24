package downloader

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Download fetches a raw URL and writes it to dstPath with 0755 permissions.
func Download(url, dstPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	outF, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer outF.Close()

	_, err = io.Copy(outF, resp.Body)
	return err
}

// ExtractZipFile downloads a zip and extracts a single target file to dstPath.
func ExtractZipFile(url, targetFilename, dstPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	tmpZip, err := os.CreateTemp("", "download-*.zip")
	if err != nil {
		return err
	}
	defer os.Remove(tmpZip.Name())
	defer tmpZip.Close()

	if _, err := io.Copy(tmpZip, resp.Body); err != nil {
		return err
	}

	r, err := zip.OpenReader(tmpZip.Name())
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		parts := strings.Split(f.Name, "/")
		base := parts[len(parts)-1]
		if strings.EqualFold(base, targetFilename) {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			outF, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
			if err != nil {
				return err
			}
			defer outF.Close()

			_, err = io.Copy(outF, rc)
			return err
		}
	}
	return fmt.Errorf("file %q not found in zip", targetFilename)
}

// ExtractZipToDir downloads a zip and extracts all of its files into dstDir.
func ExtractZipToDir(url, dstDir string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	tmpZip, err := os.CreateTemp("", "download-*.zip")
	if err != nil {
		return err
	}
	defer os.Remove(tmpZip.Name())
	defer tmpZip.Close()

	if _, err := io.Copy(tmpZip, resp.Body); err != nil {
		return err
	}

	r, err := zip.OpenReader(tmpZip.Name())
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		base := filepath.Base(f.Name)
		if base == "" || base == "." || base == ".." || base == "/" || base == "\\" {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		outPath := filepath.Join(dstDir, base)
		outF, err := os.OpenFile(outPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(outF, rc)
		outF.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// ExtractTgzFile downloads a .tar.gz and extracts a single target file to dstPath.
func ExtractTgzFile(url, targetFilename, dstPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	gzr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		parts := strings.Split(header.Name, "/")
		base := parts[len(parts)-1]
		if strings.EqualFold(base, targetFilename) {
			outF, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
			if err != nil {
				return err
			}
			defer outF.Close()

			_, err = io.Copy(outF, tr)
			return err
		}
	}
	return fmt.Errorf("file %q not found in tar.gz", targetFilename)
}
