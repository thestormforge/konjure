/*
Copyright 2022 CloudBolt, Inc.

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

package filters

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitHelmChart(t *testing.T) {
	cases := []struct {
		desc            string
		chart           string
		expectedName    string
		expectedVersion string
	}{
		{
			desc:            "simple",
			chart:           "foo-1.0.0",
			expectedName:    "foo",
			expectedVersion: "1.0.0",
		},
		{
			desc:            "prerelease",
			chart:           "foo-1.0.0-beta.1",
			expectedName:    "foo",
			expectedVersion: "1.0.0-beta.1",
		},
		{
			desc:            "hyphenated name",
			chart:           "foo-bar-1.0.0",
			expectedName:    "foo-bar",
			expectedVersion: "1.0.0",
		},
		{
			desc:            "no version",
			chart:           "foo-bar",
			expectedName:    "foo-bar",
			expectedVersion: "",
		},
		{
			desc:            "invalid version",
			chart:           "foo-bar-01.0.0",
			expectedName:    "foo-bar-01.0.0",
			expectedVersion: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			name, version := splitHelmChart(tc.chart)
			assert.Equal(t, tc.expectedName, name, "name")
			assert.Equal(t, tc.expectedVersion, version, "version")
		})
	}
}
