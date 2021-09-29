/*
Copyright 2021 GramLabs, Inc.

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

package readers

import (
	"io"
	"path/filepath"

	"sigs.k8s.io/kustomize/kyaml/kio"
)

// WithDefaultInputStream overrides the default input stream of stdin.
func WithDefaultInputStream(defaultReader io.Reader) Option {
	return func(r kio.Reader) kio.Reader {
		if rr, ok := r.(*ResourceReader); ok && rr.Reader == nil {
			rr.Reader = defaultReader
		}
		return r
	}
}

// WithWorkingDirectory sets the base directory to resolve relative paths against.
func WithWorkingDirectory(dir string) Option {
	abs := func(path string) (string, error) {
		if filepath.IsAbs(path) {
			return filepath.Clean(path), nil
		}
		return filepath.Join(dir, path), nil
	}

	return func(r kio.Reader) kio.Reader {
		if fr, ok := r.(*FileReader); ok {
			fr.Abs = abs
		}
		return r
	}
}

// WithRecursiveDirectories controls the behavior for traversing directories.
func WithRecursiveDirectories(recurse bool) Option {
	return func(r kio.Reader) kio.Reader {
		if fr, ok := r.(*FileReader); ok {
			fr.Recurse = recurse
		}
		return r
	}
}

// WithKubeconfig controls the default path of the kubeconfig file.
func WithKubeconfig(kubeconfig string) Option {
	return func(r kio.Reader) kio.Reader {
		if kr, ok := r.(*KubernetesReader); ok {
			kr.Kubeconfig = kubeconfig
		}
		return r
	}
}

// WithKubectlExecutor controls the alternate executor for kubectl.
func WithKubectlExecutor(executor Executor) Option {
	return func(r kio.Reader) kio.Reader {
		if kr, ok := r.(*KubernetesReader); ok {
			kr.Executor = executor
		}
		return r
	}
}

// WithKustomizeExecutor controls the alternate executor for kustomize.
func WithKustomizeExecutor(executor Executor) Option {
	return func(r kio.Reader) kio.Reader {
		if kr, ok := r.(*KustomizeReader); ok {
			kr.Executor = executor
		}
		return r
	}
}
