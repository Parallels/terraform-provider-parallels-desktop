package clientmodels

type TokenLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type TokenLoginResponse struct {
	Token string `json:"token"`
}
