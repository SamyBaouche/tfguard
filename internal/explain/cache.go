package explain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func cachePath(dir string, contextJSON []byte) (string, error) {
	if dir == "" {
		home, err := os.UserCacheDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, "tfguard", "explain")
	}
	sum := sha256.Sum256(contextJSON)
	name := hex.EncodeToString(sum[:]) + ".json"
	return filepath.Join(dir, name), nil
}

func loadCache(path string) (Explanation, bool, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Explanation{}, false, nil
		}
		return Explanation{}, false, err
	}
	var e Explanation
	if err := json.Unmarshal(b, &e); err != nil {
		return Explanation{}, false, fmt.Errorf("read cache %s: %w", path, err)
	}
	if err := validateExplanation(e); err != nil {
		return Explanation{}, false, nil
	}
	return e, true, nil
}

func saveCache(path string, e Explanation) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
