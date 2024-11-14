/*
Copyright 2022 GramLabs, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pipes

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"github.com/thestormforge/konjure/pkg/konjure"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// CommandWriterFormatFlag is a global that can be overwritten to change the name
// of the flag used to get the output format from the command.
var CommandWriterFormatFlag = "output"

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
func CommandWriters(cmd *cobra.Command, overwriteFiles bool) []kio.Writer {
	var outputs []kio.Writer

	format, _ := cmd.Flags().GetString(CommandWriterFormatFlag)

	if overwriteFiles {
		outputs = append(outputs, &overwriteWriter{
			Format: format,
			ClearAnnotations: []string{
				kioutil.PathAnnotation,
				kioutil.LegacyPathAnnotation,
				filters.FmtAnnotation,
			},
		})
	} else {
		outputs = append(outputs, &konjure.Writer{
			Writer:               cmd.OutOrStdout(),
			InitialDocumentStart: true,
			Format:               format,
			ClearAnnotations: []string{
				kioutil.PathAnnotation,
				kioutil.LegacyPathAnnotation,
				filters.FmtAnnotation,
			},
		})
	}

	return outputs
}

// CommandEditor returns a filter which launches each of nodes into an editor for interactive edits.
func CommandEditor(cmd *cobra.Command) kio.Filter {
	return kio.FilterFunc(func(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
		for i := range nodes {
			if err := editNode(cmd, nodes[i]); err != nil {
				return nil, err
			}
		}
		return nodes, nil
	})
}

// editNode interactively edits a node in-place.
func editNode(cmd *cobra.Command, node *yaml.RNode) error {
	tmp, err := os.CreateTemp("", strings.ReplaceAll(cmd.CommandPath(), " ", "-")+"-*.yaml")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()

	// TODO We should support an option to edit in JSON?
	// TODO Should we support the option to force Windows line endings?
	err = yaml.NewEncoder(tmp).Encode(node.YNode())
	_ = tmp.Close()
	if err != nil {
		return err
	}

	editor := editorCmd(cmd.Context(), tmp.Name())
	editor.Stdout, editor.Stderr = cmd.OutOrStdout(), cmd.ErrOrStderr()
	if f, ok := cmd.InOrStdin().(*os.File); ok && isatty.IsTerminal(f.Fd()) {
		editor.Stdin = f
	} else if tty, err := os.Open("/dev/tty"); err == nil {
		defer tty.Close()
		editor.Stdin = tty
	} else {
		return fmt.Errorf("unable to open terminal")
	}

	if err := editor.Run(); err != nil {
		return err
	}

	result, err := yaml.ReadFile(tmp.Name())
	if err != nil {
		if errors.Is(err, io.EOF) {
			// If we went to empty, just ignore it
			return nil
		}
		return err
	}

	*node = *result
	return nil
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

// overwriteWriter is an alternative to the `kio.LocalPackageWriter` that does
// not make assumptions about "packages" or their shared base directories.
type overwriteWriter struct {
	Format           string
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

	return (&konjure.Writer{
		Writer:               f,
		InitialDocumentStart: true,
		Format:               w.Format,
		ClearAnnotations:     w.ClearAnnotations,
	}).Write(nodes)
}
