package pantheon

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"path/filepath"
)

func buildNexusFingerprintState(vaultPath string) (weaveFingerprintState, error) {
	state := weaveFingerprintState{Files: map[string]string{}}
	root := filepath.Join(vaultPath, "Nexus")
	err := filepath.WalkDir(root, func(path string, e fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if e.Type()&fs.ModeSymlink != 0 || e.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		f, err := openVaultMarkdown(path)
		if err != nil {
			return err
		}
		h := sha256.New()
		_, err = io.Copy(h, f)
		f.Close()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(vaultPath, path)
		if err != nil {
			return err
		}
		state.Files[filepath.ToSlash(rel)] = hex.EncodeToString(h.Sum(nil))
		return nil
	})
	return state, err
}
