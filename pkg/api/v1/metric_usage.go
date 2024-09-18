package v1

type MetricUsage struct {
	Dashboards     []string `json:"dashboards"`
	RecordingRules []string `json:"recordingRules"`
	AlertRules     []string `json:"alertRules"`
}

type Metric struct {
	Usage MetricUsage `json:"usage"`
}
