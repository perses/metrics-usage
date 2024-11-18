package prometheus

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnalyzePromQLExpression(t *testing.T) {
	result, _, err := AnalyzePromQLExpression("service_status{env=~\"$env\",region=~\"$region\"}")
	assert.NoError(t, err)
	assert.Equal(t, []string{"service_status"}, result.TransformAsSlice())
}
