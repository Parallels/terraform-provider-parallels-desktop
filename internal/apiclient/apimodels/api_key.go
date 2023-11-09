package apimodels

import "errors"

type ApiKeyRequest struct {
	Name   string `json:"name"`
	Key    string `json:"key"`
	Secret string `json:"secret"`
}

func (r *ApiKeyRequest) Validate() error {
	if r.Name == "" {
		return errors.New("name is required")
	}
	if r.Key == "" {
		return errors.New("key is required")
	}
	if r.Secret == "" {
		return errors.New("secret is required")
	}

	return nil
}

type ApiKeyResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Key       string `json:"key"`
	Revoked   bool   `json:"revoked"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	RevokedAt string `json:"revoked_at"`
}
