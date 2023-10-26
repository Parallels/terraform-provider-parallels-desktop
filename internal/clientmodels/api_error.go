package clientmodels

type APIErrorResponse struct {
	Message string `json:"message"`
	Code    int64  `json:"code"`
}
