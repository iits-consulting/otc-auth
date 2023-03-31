package accesstoken

type TokenCreationResponse struct {
	Credential struct {
		Access        string `json:"access"`
		ExpiresAt     string `json:"expires_at"`
		Secret        string `json:"secret"`
		SecurityToken string `json:"securitytoken"`
	} `json:"credential"`
}
