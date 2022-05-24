package pipes

import (
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// CommandReaders returns KYAML readers for the supplied file name arguments.
func CommandReaders(cmd *cobra.Command, args []string) []kio.Reader {
	var inputs []kio.Reader

	var hasDefault bool
	for _, filename := range args {
		r := &kio.ByteReader{
			Reader:         &fileReader{Filename: filename},
			SetAnnotations: map[string]string{kioutil.PathAnnotation: filename},
		}

		// Handle the process relative "default" stream
		if filename == "-" {
			// Only take stdin once
			if hasDefault {
				continue
			}
			hasDefault = true

			r.Reader = cmd.InOrStdin()

			delete(r.SetAnnotations, kioutil.PathAnnotation)
			if path, err := filepath.Abs("stdin"); err == nil {
				r.SetAnnotations[kioutil.PathAnnotation] = path
			}
		}

		inputs = append(inputs, r)
	}

	return inputs
}

// CommandWriters returns KYAML writers for the supplied command.
func CommandWriters(cmd *cobra.Command, overwrite bool) []kio.Writer {
	var outputs []kio.Writer

	if overwrite {
		outputs = append(outputs, &overwriteWriter{
			ClearAnnotations: []string{
				kioutil.PathAnnotation,
				filters.FmtAnnotation,
			},
		})
	} else {
		outputs = append(outputs, &kio.ByteWriter{
			Writer: &prefixWriter{w: cmd.OutOrStdout()},
			ClearAnnotations: []string{
				kioutil.PathAnnotation,
				filters.FmtAnnotation,
			},
		})
	}

	return outputs
}

// fileReader is a reader the lazily opens a file for reading and automatically
// closes it when it hits EOF.
type fileReader struct {
	sync.Once
	io.ReadCloser
	Filename string
}

// Read performs the lazy open on first read and closes on EOF.
func (r *fileReader) Read(p []byte) (n int, err error) {
	r.Once.Do(func() {
		r.ReadCloser, err = os.Open(r.Filename)
	})
	if err != nil {
		return
	}

	n, err = r.ReadCloser.Read(p)
	if err != io.EOF {
		return
	}
	if closeErr := r.ReadCloser.Close(); closeErr != nil {
		err = closeErr
	}
	return
}

// prefixWriter is a writer that emits a document separator prefix on the first
// attempt to write output.
type prefixWriter struct {
	sync.Once
	w io.Writer
}

// Write emits "---" on first write.
func (w *prefixWriter) Write(p []byte) (n int, err error) {
	w.Once.Do(func() {
		n, err = w.w.Write([]byte("---\n"))
	})
	if err != nil {
		return
	}
	return w.w.Write(p)
}

// overwriteWriter is an alternative to the `kio.LocalPackageWriter` that does
// not make assumptions about "packages" or their shared base directories.
type overwriteWriter struct {
	ClearAnnotations []string
}

// Write overwrites the supplied nodes back into the files they came from.
func (w *overwriteWriter) Write(nodes []*yaml.RNode) error {
	if err := kioutil.DefaultPathAndIndexAnnotation("", nodes); err != nil {
		return err
	}

	pathIndex := make(map[string][]*yaml.RNode, len(nodes))
	for _, n := range nodes {
		if path, err := n.Pipe(yaml.GetAnnotation(kioutil.PathAnnotation)); err == nil {
			pathIndex[path.YNode().Value] = append(pathIndex[path.YNode().Value], n)
		}
	}
	for k := range pathIndex {
		_ = kioutil.SortNodes(pathIndex[k])
	}

	for k, v := range pathIndex {
		if err := w.writeToPath(k, v); err != nil {
			return err
		}
	}

	return nil
}

// writeToPath writes the supplied nodes into the specified file.
func (w *overwriteWriter) writeToPath(path string, nodes []*yaml.RNode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return kio.ByteWriter{
		Writer:           f,
		ClearAnnotations: w.ClearAnnotations,
	}.Write(nodes)
}
