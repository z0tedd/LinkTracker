package domain

type StackOverflowComment struct {
	Body           *string            `json:"body,omitempty"`
	CommentID      *int               `json:"comment_id,omitempty"`
	ContentLicense *string            `json:"content_license,omitempty"`
	CreationDate   *int               `json:"creation_date,omitempty"`
	Edited         *bool              `json:"edited,omitempty"`
	Owner          *StackOverflowUser `json:"owner,omitempty"`
	PostID         *int               `json:"post_id,omitempty"`
	ReplyToUser    *StackOverflowUser `json:"reply_to_user,omitempty"`
	Score          *int               `json:"score,omitempty"`
}

type StackOverflowAnswer struct {
	AnswerID         *int               `json:"answer_id,omitempty"`
	Body             *string            `json:"body,omitempty"`
	ContentLicense   *string            `json:"content_license,omitempty"`
	CreationDate     *int               `json:"creation_date,omitempty"`
	IsAccepted       *bool              `json:"is_accepted,omitempty"`
	LastActivityDate *int               `json:"last_activity_date,omitempty"`
	LastEditDate     *int               `json:"last_edit_date,omitempty"`
	Owner            *StackOverflowUser `json:"owner,omitempty"`
	QuestionID       *int               `json:"question_id,omitempty"`
	Score            *int               `json:"score,omitempty"`
}

type StackOverflowQuestion struct {
	AcceptedAnswerID *int               `json:"accepted_answer_id,omitempty"`
	AnswerCount      *int               `json:"answer_count,omitempty"`
	ClosedDate       *int               `json:"closed_date,omitempty"`
	ClosedReason     *string            `json:"closed_reason,omitempty"`
	ContentLicense   *string            `json:"content_license,omitempty"`
	CreationDate     *int               `json:"creation_date,omitempty"`
	IsAnswered       *bool              `json:"is_answered,omitempty"`
	LastActivityDate *int               `json:"last_activity_date,omitempty"`
	LastEditDate     *int               `json:"last_edit_date,omitempty"`
	Link             *string            `json:"link,omitempty"`
	Owner            *StackOverflowUser `json:"owner,omitempty"`
	ProtectedDate    *int               `json:"protected_date,omitempty"`
	QuestionID       *int               `json:"question_id,omitempty"`
	Score            *int               `json:"score,omitempty"`
	Tags             *[]string          `json:"tags,omitempty"`
	Title            *string            `json:"title,omitempty"`
	ViewCount        *int               `json:"view_count,omitempty"`
}

type StackOverflowUser struct {
	AcceptRate   *int    `json:"accept_rate,omitempty"`
	AccountID    *int    `json:"account_id,omitempty"`
	DisplayName  *string `json:"display_name,omitempty"`
	Link         *string `json:"link,omitempty"`
	ProfileImage *string `json:"profile_image,omitempty"`
	Reputation   *int    `json:"reputation,omitempty"`
	UserID       *int    `json:"user_id,omitempty"`
	UserType     *string `json:"user_type,omitempty"`
}
