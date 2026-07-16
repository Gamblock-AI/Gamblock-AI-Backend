package model

type PortalOverview struct {
	ProtectedUsers        int    `json:"protected_users"`
	PartnerApprovals      int    `json:"partner_approvals"`
	HealthyDevicesPercent int    `json:"healthy_devices_percent"`
	OpenSupport           int    `json:"open_support"`
	ModelRelease          string `json:"model_release,omitempty"`
	RulesetRelease        string `json:"ruleset_release,omitempty"`
}
