package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gh "github.com/google/go-github/v68/github"
	"github.com/stretchr/testify/require"
)

func TestExecuteGraphQL(t *testing.T) {
	t.Run("successful query", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, http.MethodPost, r.Method)
			require.Equal(t, "application/json", r.Header.Get("Content-Type"))

			resp := graphQLResponse{
				Data: json.RawMessage(`{"test": "value"}`),
			}
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client, err := gh.NewClient(nil).WithEnterpriseURLs(server.URL+"/", server.URL+"/")
		require.NoError(t, err)

		repo := NewGitHubRepository(client, "owner", "repo")
		repo.ctx = context.Background()
		repo.baseURL = server.URL

		data, err := repo.executeGraphQL("query { test }", nil)
		require.NoError(t, err)
		require.Contains(t, string(data), "test")
	})

	t.Run("HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("server error"))
		}))
		defer server.Close()

		client, err := gh.NewClient(nil).WithEnterpriseURLs(server.URL+"/", server.URL+"/")
		require.NoError(t, err)

		repo := NewGitHubRepository(client, "owner", "repo")
		repo.ctx = context.Background()
		repo.baseURL = server.URL

		_, err = repo.executeGraphQL("query { test }", nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "status 500")
	})

	t.Run("GraphQL error in response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := graphQLResponse{
				Errors: []graphQLError{{Message: "something went wrong"}},
			}
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client, err := gh.NewClient(nil).WithEnterpriseURLs(server.URL+"/", server.URL+"/")
		require.NoError(t, err)

		repo := NewGitHubRepository(client, "owner", "repo")
		repo.ctx = context.Background()
		repo.baseURL = server.URL

		_, err = repo.executeGraphQL("query { test }", nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "something went wrong")
	})
}

func TestFetchAllBranchesGraphQL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := graphQLResponse{
			Data: json.RawMessage(`{
				"repository": {
					"refs": {
						"nodes": [
							{
								"name": "main",
								"target": {
									"__typename": "Commit",
									"oid": "abc123",
									"message": "initial commit",
									"committedDate": "2024-01-01T00:00:00Z",
									"parents": {"nodes": []}
								}
							},
							{
								"name": "empty-branch",
								"target": {
									"__typename": "Commit",
									"oid": "",
									"message": "",
									"committedDate": "",
									"parents": {"nodes": []}
								}
							}
						],
						"pageInfo": {"hasNextPage": false, "endCursor": ""}
					}
				}
			}`),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := gh.NewClient(nil).WithEnterpriseURLs(server.URL+"/", server.URL+"/")
	require.NoError(t, err)

	repo := NewGitHubRepository(client, "owner", "repo")
	repo.ctx = context.Background()
	repo.baseURL = server.URL

	branches, err := repo.fetchAllBranchesGraphQL()
	require.NoError(t, err)
	// Empty OID branch should be skipped.
	require.Len(t, branches, 1)
	require.Equal(t, "main", branches[0].Name.Friendly)

	// Commit should be cached.
	cached, ok := repo.cache.getCommit("abc123")
	require.True(t, ok)
	require.Equal(t, "initial commit", cached.Message)
}

func TestFetchAllTagsGraphQL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := graphQLResponse{
			Data: json.RawMessage(`{
				"repository": {
					"refs": {
						"nodes": [
							{
								"name": "v1.0.0",
								"target": {
									"__typename": "Commit",
									"oid": "commit-sha",
									"message": "release 1.0",
									"committedDate": "2024-01-01T00:00:00Z",
									"parents": {"nodes": []}
								}
							},
							{
								"name": "v2.0.0",
								"target": {
									"__typename": "Tag",
									"oid": "tag-sha",
									"target": {
										"__typename": "Commit",
										"oid": "annotated-commit-sha",
										"message": "release 2.0",
										"committedDate": "2024-06-01T00:00:00Z",
										"parents": {"nodes": [{"oid": "parent1"}]}
									}
								}
							}
						],
						"pageInfo": {"hasNextPage": false, "endCursor": ""}
					}
				}
			}`),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := gh.NewClient(nil).WithEnterpriseURLs(server.URL+"/", server.URL+"/")
	require.NoError(t, err)

	repo := NewGitHubRepository(client, "owner", "repo")
	repo.ctx = context.Background()
	repo.baseURL = server.URL

	tags, err := repo.fetchAllTagsGraphQL()
	require.NoError(t, err)
	require.Len(t, tags, 2)

	// Lightweight tag.
	require.Equal(t, "v1.0.0", tags[0].Name.Friendly)
	require.Equal(t, "commit-sha", tags[0].TargetSha)

	// Annotated tag.
	require.Equal(t, "v2.0.0", tags[1].Name.Friendly)
	require.Equal(t, "tag-sha", tags[1].TargetSha)

	// Tag peel should be cached.
	peeled, ok := repo.cache.getTagPeel("tag-sha")
	require.True(t, ok)
	require.Equal(t, "annotated-commit-sha", peeled)

	// Commits should be cached.
	_, ok = repo.cache.getCommit("commit-sha")
	require.True(t, ok)
	_, ok = repo.cache.getCommit("annotated-commit-sha")
	require.True(t, ok)
}

func TestFetchAllBranchesGraphQL_Pagination(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var resp graphQLResponse
		if callCount == 1 {
			resp = graphQLResponse{
				Data: json.RawMessage(`{
					"repository": {
						"refs": {
							"nodes": [
								{
									"name": "main",
									"target": {
										"__typename": "Commit",
										"oid": "sha1",
										"message": "commit 1",
										"committedDate": "2024-01-01T00:00:00Z",
										"parents": {"nodes": []}
									}
								}
							],
							"pageInfo": {"hasNextPage": true, "endCursor": "cursor1"}
						}
					}
				}`),
			}
		} else {
			resp = graphQLResponse{
				Data: json.RawMessage(`{
					"repository": {
						"refs": {
							"nodes": [
								{
									"name": "develop",
									"target": {
										"__typename": "Commit",
										"oid": "sha2",
										"message": "commit 2",
										"committedDate": "2024-02-01T00:00:00Z",
										"parents": {"nodes": []}
									}
								}
							],
							"pageInfo": {"hasNextPage": false, "endCursor": ""}
						}
					}
				}`),
			}
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := gh.NewClient(nil).WithEnterpriseURLs(server.URL+"/", server.URL+"/")
	require.NoError(t, err)

	repo := NewGitHubRepository(client, "owner", "repo")
	repo.ctx = context.Background()
	repo.baseURL = server.URL

	branches, err := repo.fetchAllBranchesGraphQL()
	require.NoError(t, err)
	require.Len(t, branches, 2)
	require.Equal(t, "main", branches[0].Name.Friendly)
	require.Equal(t, "develop", branches[1].Name.Friendly)
	require.Equal(t, 2, callCount)
}
