package pipes

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type fileReadWriterOptions struct {
	// The reader used to parse the file contents.
	Reader *kio.ByteReader
	// The writer used to encode the file contents.
	Writer *kio.ByteWriter
	// The default permissions are 0666.
	WriteFilePerm os.FileMode
	// If non-zero, create the parent directories with these permissions using MkdirAll.
	MkdirAllPerm os.FileMode
}

// FileReadWriterOption represents a configuration option for the FileReader or FileWriter.
type FileReadWriterOption func(opts *fileReadWriterOptions) error

// FileWriterMkdirAll is an option that allows you to create all the parent directories on write.
func FileWriterMkdirAll(perm os.FileMode) FileReadWriterOption {
	return func(opts *fileReadWriterOptions) error {
		if opts.Writer == nil {
			return fmt.Errorf("mkdirAll requires a FileWriter")
		}
		opts.MkdirAllPerm = perm
		return nil
	}
}

// FileReader is a KIO reader that lazily loads a file.
type FileReader struct {
	// The file name to read.
	Name string
	// The file system to use for resolving file contents (defaults to the OS reader).
	FS fs.FS
	// Configuration options.
	Options []FileReadWriterOption
}

// Read opens the configured file and reads the contents.
func (r *FileReader) Read() ([]*yaml.RNode, error) {
	// TODO Should this open the file so we don't need the whole thing in memory to start?
	var data []byte
	var err error
	if r.FS != nil {
		data, err = fs.ReadFile(r.FS, r.Name)
	} else {
		data, err = os.ReadFile(r.Name)
	}
	if err != nil {
		return nil, err
	}

	// Wrap the reader in an options struct for configuration
	opts := fileReadWriterOptions{
		Reader: &kio.ByteReader{
			Reader: bytes.NewReader(data),
			SetAnnotations: map[string]string{
				kioutil.PathAnnotation: r.Name,
			},
		},
	}
	for _, opt := range r.Options {
		if err := opt(&opts); err != nil {
			return nil, err
		}
	}

	return opts.Reader.Read()
}

// FileWriter is a KIO writer that writes nodes to a file.
type FileWriter struct {
	// The file name to write.
	Name string
	// Configuration options.
	Options []FileReadWriterOption
}

// Write overwrites the file with encoded nodes.
func (w *FileWriter) Write(nodes []*yaml.RNode) error {
	// TODO Should this open the file so we don't need the whole thing in memory to start?
	var buf bytes.Buffer
	bw := &kio.ByteWriter{
		Writer: &buf,
	}

	// Wrap the reader in an options struct for configuration
	opts := fileReadWriterOptions{
		Writer: &kio.ByteWriter{
			Writer: &buf,
		},
	}
	for _, opt := range w.Options {
		if err := opt(&opts); err != nil {
			return err
		}
	}

	if err := bw.Write(nodes); err != nil {
		return err
	}

	if opts.MkdirAllPerm != 0 {
		if err := os.MkdirAll(filepath.Dir(w.Name), opts.MkdirAllPerm); err != nil {
			return err
		}
	}

	perm := opts.WriteFilePerm
	if perm == 0 {
		perm = 0666
	}

	return os.WriteFile(w.Name, buf.Bytes(), perm)
}
