package githubclient

import (
	"context"

	githubAPI "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/github"
	"github.com/central-university-dev/go-z0tedd/internal/domain"
)

type HTTPGithubClient struct {
	client githubAPI.ClientWithResponsesInterface
}

func NewHTTPGithubClient(client githubAPI.ClientWithResponsesInterface) *HTTPGithubClient {
	return &HTTPGithubClient{client: client}
}

// Аналогично для других методов...
func (c *HTTPGithubClient) GetRepositoryInfo(ctx context.Context, owner, repo string) (*domain.GithubRepositoryInfo, error) {
	rsp, err := c.client.GetReposOwnerRepoWithResponse(ctx, owner, repo)
	if err != nil {
		return nil, &domain.ClientError{Code: 500, Message: err.Error()}
	}

	if rsp.StatusCode() != 200 {
		return nil, &domain.ClientError{Code: rsp.StatusCode()}
	}

	if rsp.JSON200 == nil {
		return nil, &domain.ClientError{Code: rsp.StatusCode(), Message: "200 response is nil"}
	}

	return &domain.GithubRepositoryInfo{
		CreatedAt:   rsp.JSON200.CreatedAt,
		Description: rsp.JSON200.Description,
		FullName:    rsp.JSON200.FullName,
		Name:        rsp.JSON200.Name,
		HTMLURL:     rsp.JSON200.HtmlUrl,
		Owner: &domain.GithubUser{
			AvatarURL: rsp.JSON200.Owner.AvatarUrl,
			CreatedAt: rsp.JSON200.Owner.CreatedAt,
			HTMLURL:   rsp.JSON200.HtmlUrl,
			ID:        rsp.JSON200.Owner.Id,
			Login:     rsp.JSON200.Owner.Login,
			UpdatedAt: rsp.JSON200.Owner.CreatedAt,
		},
		UpdatedAt: rsp.JSON200.UpdatedAt,
	}, nil
}

func (c *HTTPGithubClient) GetPullRequests(ctx context.Context, owner, repo string) ([]domain.GithubListIssuesPulls, error) {
	rsp, err := c.client.ListPullRequestsWithResponse(ctx, owner, repo)
	if err != nil {
		return nil, &domain.ClientError{Code: 500, Message: err.Error()}
	}

	if rsp.StatusCode() != 200 {
		return nil, &domain.ClientError{Code: rsp.StatusCode(), Message: string(rsp.Body)}
	}

	if rsp.JSON200 == nil {
		return nil, &domain.ClientError{Code: rsp.StatusCode(), Message: "200 response is nil"}
	}

	return c.ConvertToGithubListIssuesPullsSlice(*rsp.JSON200), nil
}

func (c *HTTPGithubClient) GetIssues(ctx context.Context, owner, repo string) ([]domain.GithubListIssuesPulls, error) {
	rsp, err := c.client.ListIssuesWithResponse(ctx, owner, repo)
	if err != nil {
		return nil, &domain.ClientError{Code: 500, Message: err.Error()}
	}

	if rsp.StatusCode() != 200 {
		return nil, &domain.ClientError{Code: rsp.StatusCode(), Message: string(rsp.Body)}
	}

	if rsp.JSON200 == nil {
		return nil, &domain.ClientError{Code: rsp.StatusCode(), Message: "200 response is nil"}
	}

	return c.ConvertToGithubListIssuesPullsSlice(*rsp.JSON200), nil
}

// Conversion function for a slice of structs.
func (c *HTTPGithubClient) ConvertToGithubListIssuesPullsSlice(input []githubAPI.ListIssuesPulls) []domain.GithubListIssuesPulls {
	if input == nil {
		return nil
	}

	result := make([]domain.GithubListIssuesPulls, len(input))

	for i, item := range input {
		converted := c.ConvertToGithubListIssuesPulls(&item)
		if converted != nil {
			result[i] = *converted
		}
	}

	return result
}

// Conversion function.
func (c *HTTPGithubClient) ConvertToGithubListIssuesPulls(input *githubAPI.ListIssuesPulls) *domain.GithubListIssuesPulls {
	if input == nil {
		return nil
	}

	return &domain.GithubListIssuesPulls{
		Body:      input.Body,
		CreatedAt: input.CreatedAt,
		Title:     input.Title,
		User: &struct {
			Login *string `json:"login,omitempty"`
		}{
			Login: input.User.Login,
		},
	}
}

// func (c *HTTPGithubClient) ListPullRequests(ctx context.Context, owner, repo string) ([]domain.PullRequest, error) {
// 	rsp, err := c.client.ListPullRequestsWithResponse(ctx, owner, repo)
// 	if err != nil || rsp.StatusCode() != 200 || rsp.JSON200 == nil {
// 		return nil, fmt.Errorf("failed to fetch pull requests: %w", err)
// 	return mapToDomainPullRequests(*rsp.JSON200), nil
// }
//
// func (c *HTTPGithubClient) ListIssues(ctx context.Context, owner, repo string) ([]domain.Issue, error) {
// 	rsp, err := c.client.ListIssuesWithResponse(ctx, owner, repo)
// 	if err != nil || rsp.StatusCode() != 200 || rsp.JSON200 == nil {
// 		return nil, fmt.Errorf("failed to fetch issues: %w", err)
// 	}
// 	return mapToDomainIssues(*rsp.JSON200), nil
// }
