package api

type CreateRequest struct {
	Name string `json:"name"`
}

type AddRequest struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
}

type PaginationRequest struct {
	Page  int
	Limit int
}
