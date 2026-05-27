package crypto

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// OpenInEditor decrypts encryptedFilePath (starting with an empty buffer if
// the file does not yet exist), opens the plaintext in the user's $EDITOR,
// then re-encrypts the edited result and saves it back. The temporary plaintext
// file is securely overwritten with zeros before deletion.
func OpenInEditor(encryptedFilePath string) error {
	key, err := LoadOrCreateKey()
	if err != nil {
		return fmt.Errorf("editor: %w", err)
	}
	defer zeroBytes(key)

	plaintext, err := readAndDecrypt(encryptedFilePath, key)
	if err != nil {
		return err
	}

	tmpPath, err := writeTempFile(plaintext)
	zeroBytes(plaintext)
	if err != nil {
		return err
	}
	// secureDelete runs even if subsequent steps fail.
	defer secureDelete(tmpPath)

	if err := runEditor(tmpPath); err != nil {
		return err
	}

	newContents, err := os.ReadFile(tmpPath)
	if err != nil {
		return fmt.Errorf("editor: failed to read edited temp file: %w", err)
	}
	defer zeroBytes(newContents)

	encrypted, err := Encrypt(newContents, key)
	if err != nil {
		return fmt.Errorf("editor: failed to encrypt contents: %w", err)
	}

	if err := os.WriteFile(encryptedFilePath, encrypted, 0600); err != nil {
		return fmt.Errorf("editor: failed to write encrypted file: %w", err)
	}
	return nil
}

func readAndDecrypt(path string, key []byte) ([]byte, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return []byte{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("editor: failed to read %s: %w", path, err)
	}
	plaintext, err := Decrypt(data, key)
	if err != nil {
		return nil, fmt.Errorf("editor: failed to decrypt %s: %w", path, err)
	}
	return plaintext, nil
}

func writeTempFile(content []byte) (string, error) {
	f, err := os.CreateTemp("", "env-pull-*.tmp")
	if err != nil {
		return "", fmt.Errorf("editor: failed to create temp file: %w", err)
	}
	path := f.Name()
	if _, err := f.Write(content); err != nil {
		f.Close()
		secureDelete(path)
		return "", fmt.Errorf("editor: failed to write temp file: %w", err)
	}
	if err := f.Close(); err != nil {
		secureDelete(path)
		return "", fmt.Errorf("editor: failed to close temp file: %w", err)
	}
	return path, nil
}

func runEditor(filePath string) error {
	parts := editorArgs()
	cmdArgs := make([]string, 0, len(parts))
	cmdArgs = append(cmdArgs, parts[1:]...)
	cmdArgs = append(cmdArgs, filePath)

	cmd := exec.Command(parts[0], cmdArgs...)

	// GUI editors (e.g. notepad.exe) must not have stdin/stdout/stderr bound.
	// On Windows, attaching pipes to a GUI process causes the OS to wait for
	// the console pipe to be connected, introducing a ~3-second timeout before
	// the window appears. CLI editors (vim, nano) require the bindings so they
	// can take over the terminal.
	if !isGUIEditor(parts[0]) {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor: %s exited with error: %w", parts[0], err)
	}
	return nil
}

// isGUIEditor reports whether bin names a GUI editor that must not have
// stdin/stdout/stderr attached. The check is case-insensitive and strips
// the .exe extension so both "notepad" and "notepad.exe" are recognised.
func isGUIEditor(bin string) bool {
	name := strings.ToLower(filepath.Base(bin))
	name = strings.TrimSuffix(name, ".exe")
	switch name {
	case "notepad", "notepad++", "wordpad", "gedit", "kate":
		return true
	}
	return false
}

// editorArgs returns the editor binary and any flags parsed from $EDITOR.
// Supports multi-word values like "code --wait". Defaults to vim / notepad.
func editorArgs() []string {
	if e := os.Getenv("EDITOR"); e != "" {
		if parts := strings.Fields(e); len(parts) > 0 {
			return parts
		}
	}
	if runtime.GOOS == "windows" {
		// Use notepad.exe explicitly to bypass PATH lookup overhead and avoid
		// any shell-startup latency from routing through cmd.exe or powershell.
		return []string{"notepad.exe"}
	}
	return []string{"vim"}
}

// secureDelete overwrites path with zeros then removes it, limiting the chance
// that plaintext lingers on disk after deletion. On copy-on-write or
// log-structured filesystems (APFS, ZFS, most SSDs) physical overwrite is not
// guaranteed at the hardware level; this is best-effort at the OS layer.
func secureDelete(path string) {
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		os.Remove(path) //nolint:errcheck
		return
	}
	if info, err := f.Stat(); err == nil && info.Size() > 0 {
		zeros := make([]byte, info.Size())
		f.WriteAt(zeros, 0) //nolint:errcheck
		f.Sync()            //nolint:errcheck
	}
	f.Close()
	os.Remove(path) //nolint:errcheck
}
