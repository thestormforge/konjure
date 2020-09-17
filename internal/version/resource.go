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

package version

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
)

var (
	// Matches reference that look like a version tag (i.e. `v*`)
	versionTagRef = regexp.MustCompile(`^v([[:digit:]]+\.[[:digit:]]+\.[[:digit:]]+)$`)
	// Matches references that look like a commit hash (40 hexadecimal characters)
	hashRef = regexp.MustCompile(`^([[:xdigit:]]{8})[[:xdigit:]]{32}$`)
)

// Resource represents a target in the list of Kustomize resources, plus associated version information.
type Resource struct {
	// The reference used by Kustomize
	Target string `json:"target"`
	// The version of the resources
	Version string `json:"version,omitempty"`
	// The names of the images that should be re-tagged
	ImageNames []string `json:"images,omitempty"`
	// The new image tag to use
	ImageTag string `json:"imageTag,omitempty"`

	// TODO Should we also have a commit hash?
}

// NewResource returns a new resource for the supplied target reference. If possible, version details
// will be extracted from the reference.
func NewResource(target string) *Resource {
	v := &Resource{Target: target}
	v.parseTarget()
	return v
}

// Empty returns true if the version resource does not contain any version information
func (r *Resource) Empty() bool {
	return r == nil || (r.Version == "" && len(r.ImageNames) == 0 && r.ImageTag == "")
}

// MatchedImageName returns the image name (registry + repository) if it is matched by this versioned resource.
func (r *Resource) MatchedImageName(image string) string {
	// Ensure we do not consider the registry which may contain ":" for a port number
	i := strings.LastIndexByte(image, '/')
	if i < 0 {
		i = 0
	}

	for _, imageName := range r.ImageNames {
		// Check to see if the supplied image has a digest
		digest := strings.IndexByte(image[i:], '@')
		if digest > 0 {
			if imageName != image[0:i+digest] {
				continue
			}
			return imageName
		}

		// Check to see if the supplied image has a tag
		tag := strings.IndexByte(image[i:], ':')
		if tag > 0 {
			if imageName != image[0:i+tag] {
				continue
			}
			return imageName
		}

		// Check to see if the image is an exact match
		if imageName != image {
			continue
		}
		return imageName
	}

	return ""
}

// Unmarshal reads a JSON encoded resource: a string will be treated as reference.
func (r *Resource) UnmarshalJSON(b []byte) error {
	if b[0] == '"' {
		err := json.Unmarshal(b, &r.Target)
		if err != nil {
			return err
		}
		r.parseTarget()
		return nil
	}

	// Hide this implementation to prevent infinite recursion
	type R *Resource
	return json.Unmarshal(b, R(r))
}

// Marshal writes a JSON encoded representation of the resource.
func (r *Resource) MarshalJSON() ([]byte, error) {
	if r.Empty() {
		return json.Marshal(r.Target)
	}

	type R *Resource
	return json.Marshal(R(r))
}

func (r *Resource) parseTarget() {
	if r.Target == "" {
		return
	}

	// Use the go-getter logic to resolve the repository URL
	pwd, err := os.Getwd()
	if err != nil {
		return
	}
	force, u, err := detect(r.Target, pwd)
	if err != nil || force != "git" {
		return
	}
	ref := u.Query().Get("ref")

	// Fill in the missing values

	if r.Version == "" {
		if ms := versionTagRef.FindStringSubmatch(ref); ms != nil {
			r.Version = ref
		}
	}

	if len(r.ImageNames) == 0 {
		if u.Host == "github.com" {
			if p := strings.Split(strings.TrimPrefix(u.Path, "/"), "/"); len(p) >= 2 {
				r.ImageNames = append(r.ImageNames, strings.TrimSuffix(strings.ToLower(p[0]+"/"+p[1]), ".git"))
			}
		}
	}

	if r.ImageTag == "" {
		if ref == "master" || ref == "" {
			r.ImageTag = "edge"
		} else if ms := versionTagRef.FindStringSubmatch(ref); ms != nil {
			r.ImageTag = ms[1]
		} else if ms := hashRef.FindStringSubmatch(ref); ms != nil {
			r.ImageTag = "sha-" + ms[1]
		} else {
			r.ImageTag = strings.ReplaceAll(ref, "/", "-")
		}
	}
}
