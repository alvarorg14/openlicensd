package openlicensd

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Fingerprint returns a stable machine identifier persisted under the OS user
// config directory at <UserConfigDir>/<appName>/machine-id.
func Fingerprint(appName string) (string, error) {
	appName = strings.TrimSpace(appName)
	if appName == "" {
		return "", fmt.Errorf("openlicensd: app name is required")
	}

	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("openlicensd: resolve config dir: %w", err)
	}

	path := filepath.Join(dir, appName, "machine-id")
	return FingerprintAt(path)
}

// FingerprintAt reads or creates a stable machine identifier at path.
func FingerprintAt(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("openlicensd: fingerprint path is required")
	}

	data, err := os.ReadFile(path)
	if err == nil {
		id := strings.TrimSpace(string(data))
		if id != "" {
			return id, nil
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("openlicensd: read fingerprint: %w", err)
	}

	id, err := newRandomID()
	if err != nil {
		return "", err
	}
	if err := writeFingerprintFile(path, id); err != nil {
		return "", err
	}
	return id, nil
}

func newRandomID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("openlicensd: generate fingerprint: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

func writeFingerprintFile(path, id string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("openlicensd: create fingerprint dir: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".machine-id-*")
	if err != nil {
		return fmt.Errorf("openlicensd: create temp fingerprint file: %w", err)
	}
	tmpPath := tmp.Name()

	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}

	if _, err := tmp.WriteString(id + "\n"); err != nil {
		cleanup()
		return fmt.Errorf("openlicensd: write fingerprint: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		cleanup()
		return fmt.Errorf("openlicensd: chmod fingerprint: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("openlicensd: close fingerprint: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		cleanup()
		return fmt.Errorf("openlicensd: persist fingerprint: %w", err)
	}
	return nil
}
