package accesstoken

import "time"

type TokenCreateArgs struct {
	DurationSeconds int
}

type TokenCreationResponse struct {
	Credential struct {
		Access        string    `json:"access"`
		ExpiresAt     time.Time `json:"expires_at"`
		Secret        string    `json:"secret"`
		SecurityToken string    `json:"securitytoken"`
	} `json:"credential"`
}
