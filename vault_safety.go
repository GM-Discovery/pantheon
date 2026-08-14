package main

import (
	"fmt"
	"os"
	"syscall"
)

// openVaultMarkdown rejects symlinks at open time, closing the check/open race
// for the final path component. Callers also skip symlinks during traversal.
func openVaultMarkdown(path string) (*os.File, error) {
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), path)
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}
	if !info.Mode().IsRegular() {
		file.Close()
		return nil, fmt.Errorf("vault path is not a regular file: %s", path)
	}
	return file, nil
}
