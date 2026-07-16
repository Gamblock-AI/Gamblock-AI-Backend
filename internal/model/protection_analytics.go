package model

type ProtectionAnalyticsTotals struct {
	Blocked           int `json:"blocked"`
	Interventions     int `json:"interventions"`
	TamperEvents      int `json:"tamper_events"`
	PermissionRevoked int `json:"permission_revoked"`
}

type ProtectionAnalyticsDay struct {
	Date              string `json:"date"`
	Blocked           int    `json:"blocked"`
	Interventions     int    `json:"interventions"`
	TamperEvents      int    `json:"tamper_events"`
	PermissionRevoked int    `json:"permission_revoked"`
}

type ProtectionAnalytics struct {
	DeviceID   string                    `json:"device_id"`
	PeriodDays int                       `json:"period_days"`
	Totals     ProtectionAnalyticsTotals `json:"totals"`
	Daily      []ProtectionAnalyticsDay  `json:"daily"`
	DataState  string                    `json:"data_state"`
}
