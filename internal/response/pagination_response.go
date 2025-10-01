package response

type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int64 `json:"total_pages"`
	TotalItems int64 `json:"total_items"`
	HasMore    bool  `json:"has_more"`
	From       int   `json:"from"`
	To         int   `json:"to"`
}
