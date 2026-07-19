package model

type AuthResponse struct {
	AccessToken            string `json:"access_token"`
	RefreshToken           string `json:"refresh_token"`
	TokenType              string `json:"token_type"`
	ExpiresIn              int    `json:"expires_in"`
	User                   User   `json:"user"`
	VerificationRequired   bool   `json:"verification_required"`
	VerificationPreviewURL string `json:"verification_preview_url,omitempty"`
}
