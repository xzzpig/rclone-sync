package handlers

// Page is a generic pagination response wrapper.
type Page[T any] struct {
	Data  T   `json:"data"`
	Total int `json:"total"`
}
