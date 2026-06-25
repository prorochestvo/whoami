package resume

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Parallel()

	t.Run("loads en content", func(t *testing.T) {
		t.Parallel()
		r, err := Load("en")
		require.NoError(t, err)

		assert.Equal(t, "Seilbek Skindirov", r.Person.Name)
		assert.Equal(t, "prorochestvo", r.Person.GitHubUser)
		assert.NotEmpty(t, r.Person.Tagline)
		assert.NotEmpty(t, r.Experience)
		assert.NotEmpty(t, r.Skills)
		assert.NotEmpty(t, r.Education)
		assert.NotEmpty(t, r.Contacts)

		current := 0
		for _, e := range r.Experience {
			assert.NotEmpty(t, e.Company, "every job has a company")
			assert.NotEmpty(t, e.Role, "every job has a role")
			if e.Current {
				current++
			}
		}
		assert.Equal(t, 1, current, "exactly one role is marked current")

		// spot-check contact with empty URL round-trips correctly
		var locationContact bool
		for _, c := range r.Contacts {
			if c.Label == "Location" {
				locationContact = true
				assert.Empty(t, c.URL, "location contact has no URL")
			}
		}
		assert.True(t, locationContact, "location contact must exist")
	})

	t.Run("errors on unknown locale", func(t *testing.T) {
		t.Parallel()
		_, err := Load("does-not-exist")
		require.Error(t, err)
	})

	t.Run("errors on invalid content (no current role)", func(t *testing.T) {
		t.Parallel()
		// write a synthetic resumeJSON with no current entry and run validate directly
		r := resumeJSON{
			Person: personJSON{Name: "Test", GitHubUser: "test"},
			Experience: []experienceJSON{
				{Company: "Acme", Role: "Dev", Current: false},
			},
		}
		err := validate(r)
		require.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "current:true"), "error must mention current flag")
	})

	t.Run("errors on invalid content (multiple current roles)", func(t *testing.T) {
		t.Parallel()
		r := resumeJSON{
			Person: personJSON{Name: "Test", GitHubUser: "test"},
			Experience: []experienceJSON{
				{Company: "Acme", Role: "Dev", Current: true},
				{Company: "Beta", Role: "Eng", Current: true},
			},
		}
		err := validate(r)
		require.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "current:true"), "error must mention current flag")
	})

	t.Run("errors on invalid content (empty person.name)", func(t *testing.T) {
		t.Parallel()
		r := resumeJSON{
			Person:     personJSON{Name: "", GitHubUser: "test"},
			Experience: []experienceJSON{{Company: "Acme", Role: "Dev", Current: true}},
		}
		err := validate(r)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "person.name")
	})
}

func TestAvailable(t *testing.T) {
	t.Parallel()

	t.Run("returns sorted locale list", func(t *testing.T) {
		t.Parallel()
		locales := Available()
		require.NotEmpty(t, locales)
		// must contain en
		assert.Contains(t, locales, "en")
		// must be sorted
		for i := 1; i < len(locales); i++ {
			assert.True(t, locales[i-1] <= locales[i], "locales must be sorted")
		}
	})
}

func TestRaw(t *testing.T) {
	t.Parallel()

	t.Run("returns bytes for existing locale", func(t *testing.T) {
		t.Parallel()
		b, err := Raw("en")
		require.NoError(t, err)
		assert.NotEmpty(t, b)
		// bytes must be valid JSON containing the locale key
		assert.Contains(t, string(b), `"locale"`)
		assert.Contains(t, string(b), `"en"`)
	})

	t.Run("errors for missing locale", func(t *testing.T) {
		t.Parallel()
		_, err := Raw("does-not-exist")
		require.Error(t, err)
	})
}
