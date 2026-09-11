package api

type CreateRequest struct {
	TeamID      string `json:"team_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type UpdateRequest struct {
	TeamID      string `json:"team_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type AssignRequest struct {
	AssigneeID string `json:"assignee_id"`
}

type PaginationRequest struct {
	Page   int
	Limit  int
	Status string
	TeamID string
	Search string
}
