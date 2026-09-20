// Package update downloads pod-ssh releases from GitHub.
package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const latestReleaseURL = "https://api.github.com/repos/eznix86/pod-ssh/releases/latest"

// ErrNoAsset is returned when a release has no binary for the current platform.
var ErrNoAsset = errors.New("release has no compatible binary")

type release struct {
	TagName string  `json:"tag_name"`
	Name    string  `json:"name"`
	Assets  []asset `json:"assets"`
}

type asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

// Result describes an update attempt.
type Result struct {
	Previous string
	Current  string
	Changed  bool
}

// InstallLatest downloads and atomically installs the latest compatible release.
func InstallLatest(ctx context.Context, currentVersion string) (Result, error) {
	client := &http.Client{Timeout: 2 * time.Minute}
	latest, err := fetchRelease(ctx, client)
	if err != nil {
		return Result{}, err
	}
	result := Result{Previous: currentVersion, Current: latest.TagName}
	if normalizeVersion(currentVersion) == normalizeVersion(latest.TagName) {
		return result, nil
	}

	binaryAsset, ok := selectAsset(latest.Assets)
	if !ok {
		return Result{}, fmt.Errorf("%w for %s/%s in %s", ErrNoAsset, runtime.GOOS, runtime.GOARCH, latest.TagName)
	}
	executable, err := os.Executable()
	if err != nil {
		return Result{}, fmt.Errorf("locate current executable: %w", err)
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return Result{}, fmt.Errorf("resolve executable path: %w", err)
	}

	response, err := request(ctx, client, binaryAsset.URL)
	if err != nil {
		return Result{}, fmt.Errorf("download %s: %w", binaryAsset.Name, err)
	}
	defer response.Body.Close()

	directory := filepath.Dir(executable)
	temporary, err := os.CreateTemp(directory, ".pod-ssh-update-*")
	if err != nil {
		return Result{}, fmt.Errorf("create update beside %s: %w", executable, err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := extractBinary(binaryAsset.Name, response.Body, temporary); err != nil {
		temporary.Close()
		return Result{}, err
	}
	if err := temporary.Chmod(0o755); err != nil {
		temporary.Close()
		return Result{}, fmt.Errorf("make update executable: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return Result{}, fmt.Errorf("close downloaded update: %w", err)
	}
	if err := os.Rename(temporaryPath, executable); err != nil {
		return Result{}, fmt.Errorf("replace %s: %w", executable, err)
	}
	result.Changed = true
	return result, nil
}

func fetchRelease(ctx context.Context, client *http.Client) (release, error) {
	response, err := request(ctx, client, latestReleaseURL)
	if err != nil {
		return release{}, fmt.Errorf("check latest release: %w", err)
	}
	defer response.Body.Close()
	var latest release
	if err := json.NewDecoder(response.Body).Decode(&latest); err != nil {
		return release{}, fmt.Errorf("decode latest release: %w", err)
	}
	return latest, nil
}

func request(ctx context.Context, client *http.Client, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "pod-ssh")
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		response.Body.Close()
		return nil, fmt.Errorf("GitHub returned %s", response.Status)
	}
	return response, nil
}

func selectAsset(assets []asset) (asset, bool) {
	platform := runtime.GOOS + "_" + runtime.GOARCH
	for _, candidate := range assets {
		name := strings.ToLower(candidate.Name)
		if strings.Contains(name, platform) &&
			(strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".zip")) {
			return candidate, true
		}
	}
	return asset{}, false
}

func extractBinary(name string, source io.Reader, destination *os.File) error {
	switch {
	case strings.HasSuffix(strings.ToLower(name), ".tar.gz"):
		return extractTarGz(source, destination)
	case strings.HasSuffix(strings.ToLower(name), ".zip"):
		return extractZip(source, destination)
	default:
		return fmt.Errorf("unsupported release archive %s", name)
	}
}

func extractTarGz(source io.Reader, destination io.Writer) error {
	gzipReader, err := gzip.NewReader(source)
	if err != nil {
		return fmt.Errorf("open gzip archive: %w", err)
	}
	defer gzipReader.Close()
	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar archive: %w", err)
		}
		if header.Typeflag == tar.TypeReg && filepath.Base(header.Name) == "pod-ssh" {
			if _, err := io.Copy(destination, tarReader); err != nil {
				return fmt.Errorf("extract pod-ssh: %w", err)
			}
			return nil
		}
	}
	return fmt.Errorf("%w: archive does not contain pod-ssh", ErrNoAsset)
}

func extractZip(source io.Reader, destination io.Writer) error {
	temporary, err := os.CreateTemp("", "pod-ssh-*.zip")
	if err != nil {
		return fmt.Errorf("create temporary zip: %w", err)
	}
	path := temporary.Name()
	defer os.Remove(path)
	if _, err := io.Copy(temporary, source); err != nil {
		temporary.Close()
		return fmt.Errorf("download zip: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close zip: %w", err)
	}
	archive, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer archive.Close()
	for _, entry := range archive.File {
		if filepath.Base(entry.Name) != "pod-ssh" && filepath.Base(entry.Name) != "pod-ssh.exe" {
			continue
		}
		reader, err := entry.Open()
		if err != nil {
			return fmt.Errorf("open pod-ssh in zip: %w", err)
		}
		_, copyErr := io.Copy(destination, reader)
		closeErr := reader.Close()
		if copyErr != nil {
			return fmt.Errorf("extract pod-ssh: %w", copyErr)
		}
		if closeErr != nil {
			return fmt.Errorf("close pod-ssh archive entry: %w", closeErr)
		}
		return nil
	}
	return fmt.Errorf("%w: archive does not contain pod-ssh", ErrNoAsset)
}

func normalizeVersion(version string) string {
	return strings.TrimPrefix(strings.TrimSpace(version), "v")
}
