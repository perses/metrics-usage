package v1

type MetricUsage struct {
	Dashboards     []string `json:"dashboards,omitempty"`
	RecordingRules []string `json:"recordingRules,omitempty"`
	AlertRules     []string `json:"alertRules,omitempty"`
}

type Metric struct {
	Usage *MetricUsage `json:"usage,omitempty"`
}
