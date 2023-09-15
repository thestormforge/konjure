package pipes

import (
	"bytes"
	"io/fs"
	"os"

	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// FileReader is a KIO reader that lazily loads a file.
type FileReader struct {
	// The file name to read.
	Name string
	// The file system to use for resolving file contents (defaults to the OS reader).
	FS fs.FS

	// Temporary hack for configuring the byte reader.
	HackHooks []func(*kio.ByteReader)
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

	// Build a byte reader to process the content
	br := &kio.ByteReader{
		Reader: bytes.NewReader(data),
		SetAnnotations: map[string]string{
			kioutil.PathAnnotation: r.Name,
		},
	}

	// TODO This is temporary while we figure out what is important
	for _, h := range r.HackHooks {
		h(br)
	}

	return br.Read()
}

// FileWriter is a KIO writer that writes nodes to a file.
type FileWriter struct {
	// The file name to write.
	Name string

	// Temporary hack for configuring the byte writer.
	HackHooks []func(*kio.ByteWriter)
}

// Write overwrites the file with encoded nodes.
func (w *FileWriter) Write(nodes []*yaml.RNode) error {
	// TODO Should this open the file so we don't need the whole thing in memory to start?
	var buf bytes.Buffer
	bw := &kio.ByteWriter{
		Writer: &buf,
	}

	// TODO This is temporary while we figure out what is important
	for _, h := range w.HackHooks {
		h(bw)
	}

	if err := bw.Write(nodes); err != nil {
		return err
	}

	return os.WriteFile(w.Name, buf.Bytes(), 0666)
}
