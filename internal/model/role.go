package model

const (
	RoleUser    = "user"
	RolePartner = "partner"
	RoleAdmin   = "admin"
)

func IsAccountRole(role string) bool {
	return role == RoleUser || role == RolePartner || role == RoleAdmin
}
