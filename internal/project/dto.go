package project

type CreateRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=160"`
	Key         string `json:"key" validate:"required,min=2,max=10"`
	Description string `json:"description" validate:"max=5000"`
}
type UpdateRequest struct {
	Name        *string `json:"name" validate:"omitempty,min=1,max=160"`
	Description *string `json:"description" validate:"omitempty,max=5000"`
}
type AddMemberRequest struct {
	Email string `json:"email" validate:"required,email"`
	Role  string `json:"role" validate:"required,oneof=admin member viewer"`
}
type UpdateMemberRequest struct {
	Role string `json:"role" validate:"required,oneof=admin member viewer"`
}
