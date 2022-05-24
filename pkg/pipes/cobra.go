package pipes

import (
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
)

// FileArguments returns KYAML readers for the supplied file name arguments.
func FileArguments(cmd *cobra.Command, args []string) []kio.Reader {
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
