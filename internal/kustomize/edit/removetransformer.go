// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package edit

import (
	"errors"

	"github.com/carbonrelay/konjure/internal/kustomize/edit/kustinternal"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/konfig"
)

type removeTransformerOptions struct {
	transformerFilePaths []string
}

// newCmdRemoveTransformer removes the name of a file containing a transformer from the kustomization file.
func newCmdRemoveTransformer(fSys filesys.FileSystem) *cobra.Command {
	var o removeTransformerOptions

	cmd := &cobra.Command{
		Use: "transformer",
		Short: "Removes one or more transformer file paths from " +
			konfig.DefaultKustomizationFileName(),
		Example: ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := o.Validate(args)
			if err != nil {
				return err
			}
			err = o.Complete(cmd, args)
			if err != nil {
				return err
			}
			return o.RunRemoveTransformer(fSys)
		},
	}
	return cmd
}

// Validate validates removeTransformer command.
func (o *removeTransformerOptions) Validate(args []string) error {
	if len(args) == 0 {
		return errors.New("must specify a transformer file")
	}
	o.transformerFilePaths = args
	return nil
}

// Complete completes removeTransformer command.
func (o *removeTransformerOptions) Complete(cmd *cobra.Command, args []string) error {
	return nil
}

// RunRemoveTransformer runs removeTransformer command (do real work).
func (o *removeTransformerOptions) RunRemoveTransformer(fSys filesys.FileSystem) error {

	mf, err := kustinternal.NewKustomizationFile(fSys)
	if err != nil {
		return err
	}

	m, err := mf.Read()
	if err != nil {
		return err
	}

	transformers, err := globPatterns(m.Transformers, o.transformerFilePaths)
	if err != nil {
		return err
	}

	if len(transformers) == 0 {
		return nil
	}

	newTransformers := make([]string, 0, len(m.Transformers))
	for _, transformer := range m.Transformers {
		if kustinternal.StringInSlice(transformer, transformers) {
			continue
		}
		newTransformers = append(newTransformers, transformer)
	}

	m.Transformers = newTransformers
	return mf.Write(m)
}
