package users

import "github.com/kirsle/blog/core/models"

// User holds information about a user account.
type User struct {
	models.Base
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Role     string `json:"role"`
}

func (u *User) DocumentPath() string {
	return "users/by-id/%s"
}
