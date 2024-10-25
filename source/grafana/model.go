package grafana

type target struct {
	Expr string `json:"expr,omitempty"`
}

type panel struct {
	Type    string   `json:"type"`
	Title   string   `json:"title"`
	Panels  []panel  `json:"panels"`
	Targets []target `json:"targets"`
}

type row struct {
	Panels []panel `json:"panels"`
}

type templateVar struct {
	Name  string      `json:"name"`
	Type  string      `json:"type"`
	Query interface{} `json:"query"`
}

type simplifiedDashboard struct {
	UID        string  `json:"uid,omitempty"`
	Title      string  `json:"title"`
	Panels     []panel `json:"panels"`
	Rows       []row   `json:"rows"`
	Templating struct {
		List []templateVar `json:"list"`
	} `json:"templating"`
}

func extractTarget(panel panel) []target {
	var targets []target
	for _, p := range panel.Panels {
		targets = append(targets, extractTarget(p)...)
	}
	return append(targets, panel.Targets...)
}
