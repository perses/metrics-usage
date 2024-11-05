package prometheus

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractMetricNamesFromPromQL(t *testing.T) {
	result, err := ExtractMetricNamesFromPromQL("service_status{env=~\"$env\",region=~\"$region\"}")
	assert.NoError(t, err)
	assert.Equal(t, []string{"service_status"}, result)
}
