package dashboard

import "github.com/Basith-08/tracklume-api/internal/issue"

type Summary struct {
	TotalActive        int            `json:"total_active"`
	ByStatus           map[string]int `json:"by_status"`
	ByPriority         map[string]int `json:"by_priority"`
	ByType             map[string]int `json:"by_type"`
	Overdue            int            `json:"overdue"`
	DueNext7Days       int            `json:"due_next_7_days"`
	RecentlyUpdated    []any          `json:"recently_updated"`
	ProgressPercentage float64        `json:"progress_percentage"`
}

func presentRecent(items []issue.Issue) []any {
	out := make([]any, 0, len(items))
	for _, item := range items {
		out = append(out, issue.Present(item))
	}
	return out
}
