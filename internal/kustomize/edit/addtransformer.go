// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package edit

import (
	"errors"
	"log"

	"github.com/carbonrelay/konjure/internal/kustomize/edit/kustinternal"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/v3/pkg/fs"
)

type addTransformerOptions struct {
	transformerFilePaths []string
}

// newCmdAddTransformer adds the name of a file containing a transformer to the kustomization file.
func newCmdAddTransformer(fSys fs.FileSystem) *cobra.Command {
	var o addTransformerOptions

	cmd := &cobra.Command{
		Use:   "transformer",
		Short: "Add the name of a file containing a transformer to the kustomization file.",
		Example: `
		add transformer {filepath}`,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := o.Validate(args)
			if err != nil {
				return err
			}
			err = o.Complete(cmd, args)
			if err != nil {
				return err
			}
			return o.RunAddTransformer(fSys)
		},
	}
	return cmd
}

// Validate validates addTransformer command.
func (o *addTransformerOptions) Validate(args []string) error {
	if len(args) == 0 {
		return errors.New("must specify a transformer file")
	}
	o.transformerFilePaths = args
	return nil
}

// Complete completes addTransformer command.
func (o *addTransformerOptions) Complete(cmd *cobra.Command, args []string) error {
	return nil
}

// RunAddTransformer runs addTransformer command (do real work).
func (o *addTransformerOptions) RunAddTransformer(fSys fs.FileSystem) error {
	transformers, err := kustinternal.GlobPatterns(fSys, o.transformerFilePaths)
	if err != nil {
		return err
	}
	if len(transformers) == 0 {
		return nil
	}

	mf, err := kustinternal.NewKustomizationFile(fSys)
	if err != nil {
		return err
	}

	m, err := mf.Read()
	if err != nil {
		return err
	}

	for _, transformer := range transformers {
		if kustinternal.StringInSlice(transformer, m.Transformers) {
			log.Printf("transformer %s is already in kustomization file", transformer)
			continue
		}
		m.Transformers = append(m.Transformers, transformer)
	}

	return mf.Write(m)
}
