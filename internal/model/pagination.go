package model

type PaginationQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Limit    int    `form:"limit"`
	Status   string `form:"status"`
	Priority string `form:"priority"`
	Type     string `form:"type"`
	Role     string `form:"role"`
	Action   string `form:"action"`
	Actor    string `form:"actor"`
	Query    string `form:"q"`
}

func (p *PaginationQuery) Normalize(defaultPageSize int) (page, limit, offset int) {
	page = p.Page
	if page < 1 {
		page = 1
	}
	limit = p.Limit
	if limit < 1 {
		limit = p.PageSize
	}
	if limit < 1 {
		limit = defaultPageSize
	}
	if limit > 100 {
		limit = 100
	}
	offset = (page - 1) * limit
	return page, limit, offset
}

type PaginatedList[T any] struct {
	Items      []T  `json:"items"`
	TotalCount int  `json:"total_count"`
	Page       int  `json:"page"`
	PageSize   int  `json:"page_size"`
	TotalPages int  `json:"total_pages"`
	HasMore    bool `json:"has_more"`
}

func NewPaginatedList[T any](items []T, totalCount, page, pageSize int) PaginatedList[T] {
	if items == nil {
		items = []T{}
	}
	if pageSize < 1 {
		pageSize = 10
	}
	totalPages := (totalCount + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	return PaginatedList[T]{
		Items:      items,
		TotalCount: totalCount,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		HasMore:    page < totalPages,
	}
}
