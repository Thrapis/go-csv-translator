package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
)

// node is a source directory: its files and its sub-directories.
type node struct {
	name    string
	files   []fileRef
	folders []node
}

// fileRef is one source file.
type fileRef struct {
	name string // base name, reused verbatim for the output file
	path string // absolute/relative path to read
}

// scanFolder reads dir recursively into a node tree.
func scanFolder(dir string) (*node, error) {
	n := &node{name: filepath.Base(dir)}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", dir, err)
	}
	for _, entry := range entries {
		full := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			sub, err := scanFolder(full)
			if err != nil {
				return nil, err
			}
			n.folders = append(n.folders, *sub)
		} else {
			n.files = append(n.files, fileRef{name: entry.Name(), path: full})
		}
	}
	return n, nil
}
