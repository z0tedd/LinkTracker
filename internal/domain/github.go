package domain

import "time"

type GithubRepositoryInfo struct {
	CreatedAt   *time.Time  `json:"created_at,omitempty"`
	Description *string     `json:"description,omitempty"`
	FullName    *string     `json:"full_name,omitempty"`
	HTMLURL     *string     `json:"html_url,omitempty"`
	Name        *string     `json:"name,omitempty"`
	Owner       *GithubUser `json:"owner,omitempty"`
	UpdatedAt   *time.Time  `json:"updated_at,omitempty"`
}

type GithubUser struct {
	AvatarURL *string    `json:"avatar_url,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	HTMLURL   *string    `json:"html_url,omitempty"`
	ID        *int       `json:"id,omitempty"`
	Login     *string    `json:"login,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

type GithubListIssuesPulls struct {
	// Body Body of the issue
	Body *string `json:"body,omitempty"`
	// CreatedAt Creation date of the issue
	CreatedAt *time.Time `json:"created_at,omitempty"`
	// Title Title of the issue
	Title *string `json:"title,omitempty"`
	User  *struct {
		// Login Username of the creator
		Login *string `json:"login,omitempty"`
	} `json:"user,omitempty"`
}
