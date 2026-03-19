package codexswitch

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func readTextFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func safeWriteFile(path string, data []byte, perm os.FileMode) error {
	if err := ensureDir(filepath.Dir(path)); err != nil {
		return err
	}

	tmpPath := fmt.Sprintf("%s.%d.tmp", path, time.Now().UnixNano())
	if err := os.WriteFile(tmpPath, data, perm); err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		_ = os.Remove(tmpPath)
		return err
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	return nil
}

func safeWriteText(path string, content string) error {
	return safeWriteFile(path, []byte(content), 0o600)
}

func safeWriteJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return safeWriteFile(path, data, 0o600)
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func computeContentHash(authRaw, configRaw string) string {
	return sha256Hex("auth.json\n" + authRaw + "\nconfig.toml\n" + configRaw)
}

func buildProfileID(profileType ProfileType, stableIdentity string) string {
	hash := sha256Hex(string(profileType) + ":" + stableIdentity)
	if len(hash) > 32 {
		return hash[:32]
	}
	return hash
}

func maskAPIKey(apiKey string) string {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return ""
	}
	if len(apiKey) <= 8 {
		return apiKey[:2] + "**********" + apiKey[max(2, len(apiKey)-2):]
	}
	if len(apiKey) <= 12 {
		return apiKey[:3] + "**********" + apiKey[len(apiKey)-3:]
	}
	return apiKey[:6] + "**********" + apiKey[len(apiKey)-4:]
}

func trimmedFirst(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
