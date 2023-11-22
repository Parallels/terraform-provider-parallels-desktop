package apimodels

import "errors"

type UserRequest struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r *UserRequest) Validate() error {
	if r.Username == "" {
		return errors.New("username is required")
	}
	if r.Name == "" {
		return errors.New("name is required")
	}
	if r.Email == "" {
		return errors.New("email is required")
	}
	if r.Password == "" {
		return errors.New("password is required")
	}

	return nil
}

type UserResponse struct {
	ID       string   `json:"id"`
	Username string   `json:"username"`
	Name     string   `json:"name"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
	Claims   []string `json:"claims"`
}

type ClaimRoleResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type AddUserClaimRoleRequest struct {
	Name string `json:"name"`
}
