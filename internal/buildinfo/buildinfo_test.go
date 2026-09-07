package buildinfo

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrint(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		buildVersion string
		buildDate    string
		buildCommit  string
		want         string
	}{
		{
			name:         "all values are printed as intended",
			buildVersion: "1.2.3",
			buildDate:    "01.01.1970",
			buildCommit:  "a5b12b",
			want:         "Build version: 1.2.3\nBuild date: 01.01.1970\nBuild commit: a5b12b\n",
		},
		{
			name:         "there are default for every value",
			buildVersion: "N/A",
			buildDate:    "N/A",
			buildCommit:  "N/A",
			want:         "Build version: N/A\nBuild date: N/A\nBuild commit: N/A\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b bytes.Buffer
			Print(&b, tt.buildVersion, tt.buildDate, tt.buildCommit)
			assert.Equal(t, tt.want, b.String())
		})
	}
}
