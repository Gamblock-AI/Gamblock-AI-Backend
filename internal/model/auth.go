package model

type AuthResponse struct {
	AccessToken            string `json:"access_token,omitempty"`
	RefreshToken           string `json:"refresh_token,omitempty"`
	TokenType              string `json:"token_type,omitempty"`
	ExpiresIn              int    `json:"expires_in,omitempty"`
	User                   User   `json:"user,omitempty"`
	VerificationRequired   bool   `json:"verification_required"`
	VerificationPreviewURL string `json:"verification_preview_url,omitempty"`
	PasswordEnabled        bool   `json:"password_enabled"`
	GoogleLinked           bool   `json:"google_linked"`
	PasswordChangeRequired bool   `json:"password_change_required,omitempty"`
	PasswordChangeToken    string `json:"password_change_token,omitempty"`
}
