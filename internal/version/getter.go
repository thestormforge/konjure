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
	"fmt"
	"net/url"
	"strings"

	"github.com/yujunz/go-getter"
)

var detectors []getter.Detector

func init() {
	detectors = []getter.Detector{
		new(gitHubDetector),
		new(getter.GitDetector),
		new(getter.BitBucketDetector),
		new(getter.FileDetector),
	}
}

// Workaround for a detector that does not preserve query parameters
type gitHubDetector struct{}

func (m *gitHubDetector) Detect(src, _ string) (string, bool, error) {
	if !strings.HasPrefix(src, "github.com/") {
		return "", false, nil
	}

	u := url.URL{
		Scheme: "https",
		Host:   "github.com",
		Path:   strings.TrimPrefix(src, "github.com"),
	}

	if pos := strings.IndexByte(u.Path, '?'); pos >= 0 {
		u.RawQuery = u.Path[pos+1:]
		u.Path = u.Path[0:pos]
	}

	p := strings.SplitN(u.Path, "/", 4)
	if len(p) < 3 {
		return "", false, fmt.Errorf("GitHub URLs should be github.com/username/repo")
	}

	if !strings.HasSuffix(p[2], ".git") {
		p[2] += ".git"
	}
	if len(p) > 3 {
		p[2] += "/"
	}
	u.Path = strings.Join(p, "/")

	return "git::" + u.String(), true, nil
}

func detect(src, pwd string) (force string, u *url.URL, err error) {
	src, err = getter.Detect(src, pwd, detectors)
	if err != nil {
		return force, u, err
	}

	p := strings.SplitN(src, "::", 2)
	if len(p) == 2 {
		force = p[0]
		src = p[1]
	}

	src, _ = getter.SourceDirSubdir(src)

	u, err = url.Parse(src)
	if err != nil {
		return force, u, err
	}
	if force == "" {
		force = u.Scheme
	}

	return force, u, err
}
