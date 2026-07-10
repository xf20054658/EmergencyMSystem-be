package dto

// --- Review DTOs ---

// CreateReviewRequest 创建评价
type CreateReviewRequest struct {
	Direction     string `json:"direction" binding:"required,oneof=requester_to_provider provider_to_requester"`
	ResponseSpeed int    `json:"response_speed" binding:"required,min=1,max=5"`
	Attitude      int    `json:"attitude" binding:"required,min=1,max=5"`
	Quality       int    `json:"quality" binding:"required,min=1,max=5"`
	TextContent   string `json:"text_content,omitempty"`
}

// ReviewResponse 评价响应
type ReviewResponse struct {
	ID            string  `json:"id"`
	MatchID       string  `json:"match_id"`
	Direction     string  `json:"direction"`
	ReviewerName  string  `json:"reviewer_name,omitempty"`
	ResponseSpeed int     `json:"response_speed"`
	Attitude      int     `json:"attitude"`
	Quality       int     `json:"quality"`
	OverallRating float64 `json:"overall_rating"`
	TextContent   string  `json:"text_content,omitempty"`
	CreatedAt     string  `json:"created_at"`
}
