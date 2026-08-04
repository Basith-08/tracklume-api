package issue

import "encoding/json"

type CreateRequest struct {
	Title       string  `json:"title" validate:"required,min=1,max=240"`
	Description string  `json:"description" validate:"max=20000"`
	Type        string  `json:"type" validate:"required,oneof=task bug feature"`
	Status      string  `json:"status" validate:"required,oneof=backlog todo in_progress done cancelled"`
	Priority    string  `json:"priority" validate:"required,oneof=low medium high urgent"`
	AssigneeID  *string `json:"assignee_id"`
	DueDate     *string `json:"due_date"`
}
type UpdateRequest struct {
	Title       *string         `json:"title" validate:"omitempty,min=1,max=240"`
	Description *string         `json:"description" validate:"omitempty,max=20000"`
	Type        *string         `json:"type" validate:"omitempty,oneof=task bug feature"`
	Status      *string         `json:"status" validate:"omitempty,oneof=backlog todo in_progress done cancelled"`
	Priority    *string         `json:"priority" validate:"omitempty,oneof=low medium high urgent"`
	AssigneeID  json.RawMessage `json:"assignee_id"`
	DueDate     json.RawMessage `json:"due_date"`
}
type StatusRequest struct {
	Status string `json:"status" validate:"required,oneof=backlog todo in_progress done cancelled"`
}
type PositionRequest struct {
	Status   string `json:"status" validate:"required,oneof=backlog todo in_progress done cancelled"`
	Position int64  `json:"position" validate:"gte=0"`
}

type CreateInput struct {
	Title, Description, Type, Status, Priority string
	AssigneeID                                 *string
	DueDate                                    *string
}
type UpdateInput struct {
	Title       *string
	Description *string
	Type        *string
	Status      *string
	Priority    *string
	AssigneeSet bool
	AssigneeID  *string
	DueDateSet  bool
	DueDate     *string
}
