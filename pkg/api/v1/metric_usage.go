package v1

type RuleUsage struct {
	PromLink  string `json:"prom_link"`
	GroupName string `json:"group_name"`
	Name      string `json:"name"`
}

type MetricUsage struct {
	Dashboards     []string    `json:"dashboards,omitempty"`
	RecordingRules []RuleUsage `json:"recordingRules,omitempty"`
	AlertRules     []RuleUsage `json:"alertRules,omitempty"`
}

type Metric struct {
	Usage *MetricUsage `json:"usage,omitempty"`
}
