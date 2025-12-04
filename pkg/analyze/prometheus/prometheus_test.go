package prometheus

import (
	"testing"

	"github.com/perses/metrics-usage/pkg/analyze/expr"
	"github.com/stretchr/testify/assert"
)

func TestAnalyzePromQLExpression(t *testing.T) {
	analyzer, err := expr.NewAnalyzer(expr.EnginePromQL)
	if !assert.NoError(t, err) {
		return
	}

	result, _, err := AnalyzePromQLExpression("service_status{env=~\"$env\",region=~\"$region\"}", analyzer)
	assert.NoError(t, err)
	assert.Equal(t, []string{"service_status"}, result.TransformAsSlice())
}
