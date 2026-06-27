package main
import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"skyvern/internal/config"
	"strings"
)
type tgt struct {
	Name string
	OS   string
	Arch string
	Ext  string
}
func downloadUPXWindows(dstPath string) error {
	resp, err := http.Get("https://github.com/upx/upx/releases/download/v4.2.4/upx-4.2.4-win64.zip")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	tmpZip, err := os.CreateTemp("", "upx-*.zip")
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
		if strings.HasSuffix(f.Name, "upx.exe") {
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
			if _, err := io.Copy(outF, rc); err != nil {
				return err
			}
			return nil
		}
	}
	return fmt.Errorf("upx.exe not found in zip")
}
func downloadUPXLinux(dstPath string) error {
	arch := runtime.GOARCH
	var url string
	var dirName string
	if arch == "arm64" {
		url = "https://github.com/upx/upx/releases/download/v4.2.4/upx-4.2.4-arm64_linux.tar.xz"
		dirName = "upx-4.2.4-arm64_linux"
	} else {
		url = "https://github.com/upx/upx/releases/download/v4.2.4/upx-4.2.4-amd64_linux.tar.xz"
		dirName = "upx-4.2.4-amd64_linux"
	}

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	tmpFile, err := os.CreateTemp("", "upx-*.tar.xz")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", "upx-extract-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	c := exec.Command("tar", "-xf", tmpFile.Name(), "-C", tmpDir)
	if out, err := c.CombinedOutput(); err != nil {
		return fmt.Errorf("tar extract failed: %s (err: %v)", string(out), err)
	}

	extractedUpx := filepath.Join(tmpDir, dirName, "upx")
	outF, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer outF.Close()

	inF, err := os.Open(extractedUpx)
	if err != nil {
		return err
	}
	defer inF.Close()

	_, err = io.Copy(outF, inF)
	return err
}
func downloadFile(url, dstPath string) error {
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
func downloadAndExtractZip(url, targetFilename, dstPath string) error {
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
func downloadAndExtractTgz(url, targetFilename, dstPath string) error {
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
func getNgrokInfo(goos, goarch string) (url string, archiveName string, isZip bool) {
	switch goos {
	case "windows":
		archiveName = "ngrok.exe"
		isZip = true
		if goarch == "386" {
			url = "https://bin.equinox.io/c/bNyj1mQVY4c/ngrok-v3-stable-windows-386.zip"
		} else {
			url = "https://bin.equinox.io/c/bNyj1mQVY4c/ngrok-v3-stable-windows-amd64.zip"
		}
	case "darwin":
		archiveName = "ngrok"
		isZip = true
		if goarch == "arm64" {
			url = "https://bin.equinox.io/c/bNyj1mQVY4c/ngrok-v3-stable-darwin-arm64.zip"
		} else {
			url = "https://bin.equinox.io/c/bNyj1mQVY4c/ngrok-v3-stable-darwin-amd64.zip"
		}
	case "linux":
		archiveName = "ngrok"
		isZip = false
		if goarch == "arm64" {
			url = "https://bin.equinox.io/c/bNyj1mQVY4c/ngrok-v3-stable-linux-arm64.tgz"
		} else {
			url = "https://bin.equinox.io/c/bNyj1mQVY4c/ngrok-v3-stable-linux-amd64.tgz"
		}
	case "android":
		archiveName = "ngrok"
		isZip = false
		url = "https://bin.equinox.io/c/bNyj1mQVY4c/ngrok-v3-stable-linux-arm64.tgz"
	}
	return
}
func main() {
	tgts := []tgt{
		{"Windows (x64)", "windows", "amd64", ".exe"},
		{"Windows (32-bit)", "windows", "386", ".exe"},
		{"macOS (Apple Silicon)", "darwin", "arm64", ""},
		{"macOS (Intel)", "darwin", "amd64", ""},
		{"Linux (x64)", "linux", "amd64", ""},
		{"Android (Termux / arm64)", "android", "arm64", ""},
	}
	fmt.Println("SKYVERN")
	fmt.Println("Select your target build platform:")
	for i, t := range tgts {
		fmt.Printf("[%d] %s (GOOS=%s GOARCH=%s)\n", i+1, t.Name, t.OS, t.Arch)
	}
	fmt.Printf("[%d] Build for Current Host (%s/%s)\n", len(tgts)+1, runtime.GOOS, runtime.GOARCH)
	fmt.Printf("[%d] Build All Platforms\n", len(tgts)+2)
	fmt.Print("\nEnter choice (q to quit): ")
	rd := bufio.NewReader(os.Stdin)
	in, _ := rd.ReadString('\n')
	in = strings.TrimSpace(in)
	if strings.ToLower(in) == "q" {
		fmt.Println("Cancelled.")
		return
	}
	isBuildAll := in == fmt.Sprintf("%d", len(tgts)+2)
	var targetsToBuild []tgt

	t := tgt{
		Name: "Current Host",
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}
	if runtime.GOOS == "windows" {
		t.Ext = ".exe"
	}

	if isBuildAll {
		targetsToBuild = tgts
	} else {
		if in != "" && in != fmt.Sprintf("%d", len(tgts)+1) {
			var v int
			_, err := fmt.Sscanf(in, "%d", &v)
			if err != nil || v < 1 || v > len(tgts) {
				fmt.Println("invalid selection")
				return
			}
			t = tgts[v-1]
		}
		targetsToBuild = []tgt{t}
	}

	fmt.Print("Enable UPX compression? (y/N): ")
	upxIn, _ := rd.ReadString('\n')
	doUpx := strings.ToLower(strings.TrimSpace(upxIn)) == "y"

	upxPath := "upx"
	useUpx := false
	if doUpx {
		if _, err := exec.LookPath("upx"); err == nil {
			useUpx = true
		} else {
			localUpx := "./upx"
			if runtime.GOOS == "windows" {
				localUpx = "./upx.exe"
			}
			if _, err := os.Stat(localUpx); err == nil {
				upxPath = localUpx
				useUpx = true
			} else if runtime.GOOS == "windows" {
				fmt.Println("[*] upx not found on PATH. downloading UPX v4.2.4 automatically...")
				if err := downloadUPXWindows(localUpx); err == nil {
					upxPath = localUpx
					useUpx = true
					fmt.Println("[+] UPX downloaded successfully.")
				} else {
					fmt.Printf("[!] failed to download UPX: %v\n", err)
				}
			} else if runtime.GOOS == "linux" {
				fmt.Println("[*] upx not found on PATH. downloading UPX v4.2.4 automatically...")
				if err := downloadUPXLinux(localUpx); err == nil {
					upxPath = localUpx
					useUpx = true
					fmt.Println("[+] UPX downloaded successfully.")
				} else {
					fmt.Printf("[!] failed to download UPX: %v\n", err)
				}
			}
		}
	}

	for _, target := range targetsToBuild {
		out := fmt.Sprintf("skyvern-%s-%s-%s%s", config.Version, target.OS, target.Arch, target.Ext)
		fmt.Printf("\nBuilding for %s (%s/%s)...\n", target.Name, target.OS, target.Arch)
		c := exec.Command("go", "build", "-mod=mod", "-ldflags=-s -w", "-o", out, "main.go")
		c.Env = append(os.Environ(),
			"GOOS="+target.OS,
			"GOARCH="+target.Arch,
		)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			fmt.Printf("\n[!] build failed for %s: %v\n", target.Name, err)
			continue
		}
		fmt.Printf("[+] build successful: %s\n", out)

		ngrokDst := "ngrok-" + target.OS
		if target.OS == "windows" {
			ngrokDst = "ngrok-windows.exe"
		}
		if _, err := os.Stat(ngrokDst); os.IsNotExist(err) {
			ngrokURL, archName, isZip := getNgrokInfo(target.OS, target.Arch)
			if ngrokURL != "" {
				fmt.Printf("[*] ngrok for target (%s/%s) not found. downloading automatically...\n", target.OS, target.Arch)
				var dlErr error
				if isZip {
					dlErr = downloadAndExtractZip(ngrokURL, archName, ngrokDst)
				} else {
					dlErr = downloadAndExtractTgz(ngrokURL, archName, ngrokDst)
				}
				if dlErr == nil {
					fmt.Println("[+] ngrok downloaded successfully.")
				} else {
					fmt.Printf("[!] failed to download ngrok: %v\n", dlErr)
				}
			}
		}

		if doUpx && useUpx {
			fmt.Printf("[*] compressing %s using %s...\n", out, upxPath)
			uc := exec.Command(upxPath, "--best", out)
			uc.Stdout = os.Stdout
			uc.Stderr = os.Stderr
			if err := uc.Run(); err != nil {
				fmt.Printf("[!] upx compression failed for %s: %v\n", out, err)
			} else {
				fmt.Println("[+] upx compression complete.")
			}
		}
	}

	lavaDst := "lavalink/Lavalink.jar"
	if _, err := os.Stat(lavaDst); os.IsNotExist(err) {
		fmt.Println("[*] Lavalink.jar not found. downloading automatically...")
		_ = os.MkdirAll("lavalink", 0755)
		if err := downloadFile("https://github.com/lavalink-devs/Lavalink/releases/download/4.0.8/Lavalink.jar", lavaDst); err == nil {
			fmt.Println("[+] Lavalink.jar downloaded successfully.")
		} else {
			fmt.Printf("[!] failed to download Lavalink.jar: %v\n", err)
		}
	}
}