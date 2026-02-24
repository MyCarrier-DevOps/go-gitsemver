package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/MyCarrier-DevOps/go-gitsemver/internal/git"
)

// GraphQL query to fetch all branches with tip commit details.
const branchesQuery = `
query($owner: String!, $name: String!, $cursor: String) {
  repository(owner: $owner, name: $name) {
    refs(refPrefix: "refs/heads/", first: 100, after: $cursor) {
      nodes {
        name
        target {
          ... on Commit {
            oid
            message
            committedDate
            parents(first: 10) {
              nodes { oid }
            }
          }
        }
      }
      pageInfo {
        hasNextPage
        endCursor
      }
    }
  }
}
`

// GraphQL query to fetch all tags with peel information.
// For annotated tags, resolves through to the target commit.
const tagsQuery = `
query($owner: String!, $name: String!, $cursor: String) {
  repository(owner: $owner, name: $name) {
    refs(refPrefix: "refs/tags/", first: 100, after: $cursor) {
      nodes {
        name
        target {
          __typename
          oid
          ... on Tag {
            target {
              __typename
              oid
              ... on Commit {
                oid
                message
                committedDate
                parents(first: 10) {
                  nodes { oid }
                }
              }
            }
          }
          ... on Commit {
            oid
            message
            committedDate
            parents(first: 10) {
              nodes { oid }
            }
          }
        }
      }
      pageInfo {
        hasNextPage
        endCursor
      }
    }
  }
}
`

// graphQL response types.

type graphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []graphQLError  `json:"errors"`
}

type graphQLError struct {
	Message string `json:"message"`
}

type refsResponse struct {
	Repository struct {
		Refs refConnection `json:"refs"`
	} `json:"repository"`
}

type refConnection struct {
	Nodes    []refNode `json:"nodes"`
	PageInfo pageInfo  `json:"pageInfo"`
}

type refNode struct {
	Name   string    `json:"name"`
	Target refTarget `json:"target"`
}

type refTarget struct {
	TypeName      string     `json:"__typename"`
	OID           string     `json:"oid"`
	Message       string     `json:"message"`
	CommittedDate string     `json:"committedDate"`
	Parents       parentList `json:"parents"`
	// For annotated tags: nested target.
	Target *refTarget `json:"target"`
}

type parentList struct {
	Nodes []struct {
		OID string `json:"oid"`
	} `json:"nodes"`
}

type pageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

// executeGraphQL sends a GraphQL query using the client's HTTP transport.
func (r *GitHubRepository) executeGraphQL(query string, variables map[string]interface{}) (json.RawMessage, error) {
	reqBody := graphQLRequest{
		Query:     query,
		Variables: variables,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling GraphQL request: %w", err)
	}

	graphqlURL := "https://api.github.com/graphql"
	if r.baseURL != "" {
		graphqlURL = deriveGraphQLURL(r.baseURL)
	}

	httpReq, err := http.NewRequestWithContext(r.ctx, http.MethodPost, graphqlURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("creating GraphQL request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := r.client.Client().Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("executing GraphQL request: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading GraphQL response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GraphQL request failed with status %d: %s", httpResp.StatusCode, string(respBody))
	}

	var resp graphQLResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("parsing GraphQL response: %w", err)
	}

	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL error: %s", resp.Errors[0].Message)
	}

	return resp.Data, nil
}

// fetchAllBranchesGraphQL fetches all branches with tip commit details via GraphQL.
func (r *GitHubRepository) fetchAllBranchesGraphQL() ([]git.Branch, error) {
	var branches []git.Branch
	var cursor *string

	for {
		vars := map[string]interface{}{
			"owner": r.owner,
			"name":  r.repo,
		}
		if cursor != nil {
			vars["cursor"] = *cursor
		}

		data, err := r.executeGraphQL(branchesQuery, vars)
		if err != nil {
			return nil, fmt.Errorf("fetching branches via GraphQL: %w", err)
		}

		var resp refsResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, fmt.Errorf("parsing branches response: %w", err)
		}

		for _, node := range resp.Repository.Refs.Nodes {
			// Skip branches with empty OID (e.g., unborn branches).
			if node.Target.OID == "" {
				continue
			}

			commit := commitFromRefTarget(node.Target)
			r.cache.putCommit(commit)

			branches = append(branches, git.Branch{
				Name:     git.NewBranchReferenceName(node.Name),
				Tip:      &commit,
				IsRemote: false,
			})
		}

		if !resp.Repository.Refs.PageInfo.HasNextPage {
			break
		}
		cursor = &resp.Repository.Refs.PageInfo.EndCursor
	}

	return branches, nil
}

// fetchAllTagsGraphQL fetches all tags with peel info via GraphQL.
// Pre-populates the tagPeels cache for instant PeelTagToCommit lookups.
func (r *GitHubRepository) fetchAllTagsGraphQL() ([]git.Tag, error) {
	var tags []git.Tag
	var cursor *string

	for {
		vars := map[string]interface{}{
			"owner": r.owner,
			"name":  r.repo,
		}
		if cursor != nil {
			vars["cursor"] = *cursor
		}

		data, err := r.executeGraphQL(tagsQuery, vars)
		if err != nil {
			return nil, fmt.Errorf("fetching tags via GraphQL: %w", err)
		}

		var resp refsResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, fmt.Errorf("parsing tags response: %w", err)
		}

		for _, node := range resp.Repository.Refs.Nodes {
			tagSha := node.Target.OID
			var commitSha string

			switch node.Target.TypeName {
			case "Commit":
				// Lightweight tag: target is the commit directly.
				commitSha = tagSha
				commit := commitFromRefTarget(node.Target)
				r.cache.putCommit(commit)

			case "Tag":
				// Annotated tag: peel through to the commit.
				if node.Target.Target != nil && node.Target.Target.OID != "" {
					commitSha = node.Target.Target.OID
					commit := commitFromRefTarget(*node.Target.Target)
					r.cache.putCommit(commit)
				}
			}

			if commitSha != "" {
				r.cache.putTagPeel(tagSha, commitSha)
			}

			tags = append(tags, git.Tag{
				Name:      git.NewReferenceName("refs/tags/" + node.Name),
				TargetSha: tagSha,
			})
		}

		if !resp.Repository.Refs.PageInfo.HasNextPage {
			break
		}
		cursor = &resp.Repository.Refs.PageInfo.EndCursor
	}

	return tags, nil
}

// deriveGraphQLURL converts a GitHub REST API base URL to the corresponding
// GraphQL endpoint. For GitHub Enterprise, the REST base URL is typically
// "https://ghe.example.com/api/v3" and the GraphQL endpoint is
// "https://ghe.example.com/api/graphql" (not "/api/v3/graphql").
func deriveGraphQLURL(baseURL string) string {
	if strings.HasSuffix(baseURL, "/api/v3") {
		return baseURL[:len(baseURL)-len("/api/v3")] + "/api/graphql"
	}
	if strings.HasSuffix(baseURL, "/api/v3/") {
		return baseURL[:len(baseURL)-len("/api/v3/")] + "/api/graphql"
	}
	return strings.TrimRight(baseURL, "/") + "/graphql"
}

// commitFromRefTarget converts a GraphQL ref target to a git.Commit.
func commitFromRefTarget(target refTarget) git.Commit {
	parents := make([]string, 0, len(target.Parents.Nodes))
	for _, p := range target.Parents.Nodes {
		parents = append(parents, p.OID)
	}

	var when time.Time
	if target.CommittedDate != "" {
		when, _ = time.Parse(time.RFC3339, target.CommittedDate)
	}

	return git.Commit{
		Sha:     target.OID,
		Parents: parents,
		When:    when,
		Message: target.Message,
	}
}
