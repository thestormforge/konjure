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
