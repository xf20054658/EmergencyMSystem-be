package dto

// --- Common DTOs ---

// PageQuery 通用分页查询
type PageQuery struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}

// Normalize 规范化分页参数
func (q *PageQuery) Normalize() {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 20
	}
}

// Offset 计算偏移量
func (q *PageQuery) Offset() int {
	return (q.Page - 1) * q.PageSize
}

// IDParam 通用ID路径参数
type IDParam struct {
	ID string `uri:"id" binding:"required,uuid"`
}
