package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Fetch(t *testing.T) {
	t.Parallel()

	t.Run("maps profile and repos, excludes forks, ranks languages", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/users/octocat":
				_, _ = w.Write([]byte(`{"login":"octocat","public_repos":3,"followers":10,"following":2}`))
			case "/users/octocat/repos":
				_, _ = w.Write([]byte(`[
					{"stargazers_count":5,"language":"Go","fork":false},
					{"stargazers_count":2,"language":"Go","fork":false},
					{"stargazers_count":9,"language":"Rust","fork":true},
					{"stargazers_count":1,"language":"Shell","fork":false}
				]`))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer srv.Close()

		c := NewClient("octocat", "")
		c.baseURL = srv.URL

		stats, err := c.Fetch(context.Background())
		require.NoError(t, err)
		assert.True(t, stats.Available)
		assert.Equal(t, "octocat", stats.Login)
		assert.Equal(t, 3, stats.PublicRepos)
		assert.Equal(t, 10, stats.Followers)
		// forked Rust repo (9 stars) is excluded from the total.
		assert.Equal(t, 8, stats.TotalStars)
		require.Len(t, stats.TopLanguages, 2)
		assert.Equal(t, "Go", stats.TopLanguages[0].Name)
		assert.Equal(t, 2, stats.TopLanguages[0].Repos)
		assert.Equal(t, "Shell", stats.TopLanguages[1].Name)
	})

	t.Run("non-200 returns unavailable snapshot and error", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		defer srv.Close()

		c := NewClient("octocat", "")
		c.baseURL = srv.URL

		stats, err := c.Fetch(context.Background())
		require.Error(t, err)
		assert.False(t, stats.Available)
	})

	t.Run("empty user is an error", func(t *testing.T) {
		t.Parallel()

		stats, err := NewClient("", "").Fetch(context.Background())
		require.Error(t, err)
		assert.False(t, stats.Available)
	})
}
