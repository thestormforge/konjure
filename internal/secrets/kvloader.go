/*
Copyright 2020 GramLabs, Inc.

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

package secrets

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"sigs.k8s.io/kustomize/api/ifc"
	"sigs.k8s.io/kustomize/api/types"
)

type PassPhrase string

func (p PassPhrase) Read() ([]byte, error) {
	// TODO if p == "" prompt with echo off
	parts := strings.SplitN(string(p), ":", 2)
	if len(parts) == 1 {
		if parts[0] == "stdin" {
			return ioutil.ReadAll(os.Stdin)
		}
		return []byte(parts[0]), nil
	}

	switch parts[0] {
	case "pass":
		return []byte(parts[1]), nil
	case "env":
		return []byte(os.Getenv(parts[1])), nil
	case "file":
		return ioutil.ReadFile(parts[1])
	case "fd":
		fd, err := strconv.ParseUint(parts[1], 10, 32)
		if err != nil {
			return nil, err
		}
		f := os.NewFile(uintptr(fd), "pipe")
		return ioutil.ReadAll(f)
	}
	return nil, fmt.Errorf("unknown pass phrase ")
}

// NewKvLoader wraps the supplied key/value loader with additional support for potentially sensitive data.
func NewKvLoader(kvs ifc.KvLoader, pps map[string]PassPhrase) ifc.KvLoader {
	return &kvLoader{kvs: kvs, pps: pps}
}

type kvLoader struct {
	kvs ifc.KvLoader
	pps map[string]PassPhrase
}

func (k *kvLoader) Load(args types.KvPairSources) ([]types.Pair, error) {
	all, err := k.kvs.Load(args)
	for i := range all {
		if key := all[i].Key; strings.HasSuffix(key, ".gpg") {
			if err := k.handleGPG(&all[i]); err != nil {
				return nil, err
			}
		}
	}
	return all, err
}

func (k *kvLoader) Validator() ifc.Validator {
	return k.kvs.Validator()
}

func (k *kvLoader) handleGPG(pair *types.Pair) error {
	key := strings.TrimSuffix(pair.Key, ".gpg")

	// Get the passphrase
	pp, err := k.pps[key].Read()
	if err != nil {
		return err
	}

	// Execute GPG
	cmd := exec.Command("gpg", "--quiet", "--batch", "--yes", "--decrypt", "--passphrase", string(pp))
	cmd.Stdin = strings.NewReader(pair.Value)
	value, err := cmd.Output()
	if err != nil {
		return err
	}

	// Success, update the key/value pair
	pair.Key = key
	pair.Value = string(value)
	return nil
}
