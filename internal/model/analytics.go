package model

// AnalyticsTotals aggregates protection event counts over an analytics period.
// All values are privacy-preserving counts; no browsing content is ever
// represented here.
type AnalyticsTotals struct {
	Blocked           int `json:"blocked"`
	Interventions     int `json:"interventions"`
	TamperEvents      int `json:"tamper_events"`
	PermissionRevoked int `json:"permission_revoked"`
}

type AnalyticsDay struct {
	Date              string `json:"date"`
	Blocked           int    `json:"blocked"`
	Interventions     int    `json:"interventions"`
	TamperEvents      int    `json:"tamper_events"`
	PermissionRevoked int    `json:"permission_revoked"`
}

type AnalyticsHour struct {
	Hour  int `json:"hour"`
	Count int `json:"count"`
}

// AnalyticsSummary is the role-aware analytics payload served to partner and
// admin dashboards. Hourly is always a 24-slot histogram (0-23) of blocked
// events recorded in device-local time.
type AnalyticsSummary struct {
	PeriodDays       int             `json:"period_days"`
	Totals           AnalyticsTotals `json:"totals"`
	Daily            []AnalyticsDay  `json:"daily"`
	Hourly           []AnalyticsHour `json:"hourly"`
	DataState        string          `json:"data_state"`
	MemberCount      int             `json:"member_count"`
	SharedMemberCount int            `json:"shared_member_count"`
	ProtectedUsers   int             `json:"protected_users,omitempty"`
}
