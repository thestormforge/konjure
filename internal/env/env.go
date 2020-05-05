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

package env

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/carbonrelay/konjure/internal/kustomize"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/kustomize/api/resid"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/resource"
	"sigs.k8s.io/kustomize/api/types"
)

type command struct {
	// Shell allows you to override the shell to generate the environment for
	Shell string
	// Unset variables instead of setting them
	Unset bool

	// ConfigMapLabelSelector is a label selector to restrict which ConfigMaps to consider
	ConfigMapLabelSelector string
	// SecretLabelSelector is a label selector to restrict which Secrets to consider
	SecretLabelSelector string

	// env is the list of environment variables to output
	env []corev1.EnvVar
}

// NewCommand creates a new command for running the environment extraction from the CLI
func NewCommand() *cobra.Command {
	c := &command{}
	cmd := kustomize.NewPluginRunner(c, kustomize.WithTransformerFilenameFlag(), kustomize.WithPrinter(c.print))
	cmd.Use = "env"
	cmd.Short = "Extract environment mappings"
	cmd.Long = "Extracts config map and secret environment assignments"

	cmd.Flags().StringVar(&c.Shell, "shell", "", "force environment to be configured for a specific `shell`")
	cmd.Flags().BoolVarP(&c.Unset, "unset", "u", false, "unset variables instead of setting them")
	cmd.Flags().StringVar(&c.ConfigMapLabelSelector, "configmap-selector", "", "`selector` to filter ConfigMaps on")
	cmd.Flags().StringVar(&c.SecretLabelSelector, "secret-selector", "", "`selector` to filter Secrets on")

	return cmd
}

// Transform does not make any changes to the resource map, it just scans for environment variables to output
func (c *command) Transform(m resmap.ResMap) error {
	// This is over engineered to work off the list of EnvVars...e.g. why not just collect the values in one pass?

	// Add environment variables from ConfigMaps
	cms, err := m.Select(types.Selector{Gvk: resid.Gvk{Version: "v1", Kind: "ConfigMap"}, LabelSelector: c.ConfigMapLabelSelector})
	if err != nil {
		return err
	}
	for i := range cms {
		data, err := cms[i].GetStringMap("data")
		if err != nil {
			return err
		}
		for k, v := range data {
			if envVar := c.newEnvFrom(k, v); envVar != nil {
				envVar.ValueFrom.ConfigMapKeyRef = &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: cms[i].GetName()},
					Key:                  k,
				}
				c.env = append(c.env, *envVar)
			}
		}
	}

	// Add environment variables from Secrets
	ss, err := m.Select(types.Selector{Gvk: resid.Gvk{Version: "v1", Kind: "Secret"}, LabelSelector: c.SecretLabelSelector})
	if err != nil {
		return err
	}
	for i := range ss {
		data, err := ss[i].GetMap("data")
		if err != nil {
			return err
		}
		for k, v := range data {
			if envVar := c.newEnvFrom(k, v); envVar != nil {
				envVar.ValueFrom.SecretKeyRef = &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: ss[i].GetName()},
					Key:                  k,
				}
				c.env = append(c.env, *envVar)
			}
		}
	}

	return nil
}

func (c *command) print(w io.Writer, m resmap.ResMap) error {
	// Do nothing
	if len(c.env) == 0 {
		return nil
	}

	// To generate "unset" statements, we don't need the values
	if c.Unset {
		var names []string
		for i := range c.env {
			names = append(names, c.env[i].Name)
		}

		switch c.Shell {
		case "": // Default, just name=value
			for _, n := range names {
				_, _ = fmt.Fprintf(w, "%s=\n", n)
			}
		default: // bash, zsh, etc.
			_, _ = fmt.Fprintf(w, "unset %s\n", strings.Join(names, " "))
		}
		return nil
	}

	// Look up values and render each environment variable
	for i := range c.env {
		envVar := &c.env[i]
		value, err := GetEnvValue(envVar, m)
		if err != nil {
			return err
		}

		// TODO Should we print a comment with what object it came from?
		// TODO Should we look for `$(VARNAME)` and evaluate it? Is there Kube code we can reuse?

		switch c.Shell {
		case "": // Default, just name=value
			_, _ = fmt.Fprintf(w, "%s=%s\n", envVar.Name, value)
		default: // bash, zsh, etc.
			_, _ = fmt.Fprintf(w, "export %s=%s\n", envVar.Name, strconv.Quote(value))
		}
	}
	return nil
}

func (c *command) newEnvFrom(key string, value interface{}) *corev1.EnvVar {
	// NOTE: Even though the value is supplied, we only use it to filter out mappings we do not want, it MUST NOT be in the result

	// If key contains a "." assume it is meant to be a file name
	if strings.Contains(key, ".") {
		return nil
	}

	// If value contains a line break character assume it is meant to be file contents
	switch v := value.(type) {
	case string:
		if strings.ContainsAny(v, "\n\r") {
			return nil
		}
	case []byte:
		if bytes.ContainsAny(v, "\n\r") {
			return nil
		}
	}

	// TODO Filter out keys we do not want or transform the key
	return &corev1.EnvVar{Name: key, ValueFrom: &corev1.EnvVarSource{}}
}

// GetEnvValue returns the value of an environment variable, looking up the value from the resource map if necessary
func GetEnvValue(envVar *corev1.EnvVar, m resmap.ResMap) (string, error) {
	// No reference, just return the value directly (even if it is empty)
	if envVar.ValueFrom == nil {
		return envVar.Value, nil
	}

	// Determine how to look up the value based on the valueFrom reference
	switch {

	case envVar.ValueFrom.ConfigMapKeyRef != nil:
		id := resid.NewResId(resid.Gvk{Version: "v1", Kind: "ConfigMap"}, envVar.ValueFrom.ConfigMapKeyRef.Name)
		res, err := getByCurrentGvkn(m, id, envVar.ValueFrom.ConfigMapKeyRef.Optional)
		if err != nil {
			return "", err
		}
		return res.GetString("data." + envVar.ValueFrom.ConfigMapKeyRef.Key)

	case envVar.ValueFrom.SecretKeyRef != nil:
		id := resid.NewResId(resid.Gvk{Version: "v1", Kind: "Secret"}, envVar.ValueFrom.SecretKeyRef.Name)
		res, err := getByCurrentGvkn(m, id, envVar.ValueFrom.SecretKeyRef.Optional)
		if err != nil {
			return "", err
		}
		v, err := res.GetFieldValue("data." + envVar.ValueFrom.SecretKeyRef.Key)
		if err != nil {
			return "", err
		}
		vb, ok := v.([]byte)
		if !ok {
			vb, err = base64.StdEncoding.DecodeString(v.(string))
			if err != nil {
				return "", err
			}
		}
		return string(vb), nil

	}
	// TODO If we were to support the other types of references, we would need some type of context object reference to lookup first
	return "", fmt.Errorf("cannot get environment variable value")
}

func getByCurrentGvkn(m resmap.ResMap, id resid.ResId, optional *bool) (*resource.Resource, error) {
	result := m.GetMatchingResourcesByCurrentId(id.GvknEquals)
	if len(result) > 1 {
		return nil, fmt.Errorf("multiple matches for CurrentId %s", id)
	}
	if len(result) == 0 {
		if optional != nil && *optional {
			return nil, nil
		}
		return nil, fmt.Errorf("no matches for CurrentId %s", id)
	}
	return result[0], nil
}
