package skill

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersion(t *testing.T) {
	assert.Equal(t, Version("1.2.3"), Version("1.2.3"), "stable for the same server version")
	assert.NotEqual(t, Version("1.2.3"), Version("1.2.4"), "changes with the server version")
	assert.True(t, strings.HasPrefix(Version("dev"), "dev+"))
}

func TestContent(t *testing.T) {
	c := Content("1.2.3")
	assert.True(t, strings.HasPrefix(c, "---\nname: "+Name+"\n"), "frontmatter must name the skill")
	assert.NotContains(t, c, "{{VERSION}}")
	assert.Equal(t, 1, strings.Count(template, "{{VERSION}}"), "exactly one version marker")
	assert.Equal(t, Version("1.2.3"), InstalledVersion(c))
	assert.Empty(t, InstalledVersion("no marker here"))
}

func TestInstall(t *testing.T) {
	dir := t.TempDir()

	path, prev, written, err := Install(dir, Content("1.0.0"))
	require.NoError(t, err)
	assert.Equal(t, Path(dir), path)
	assert.Empty(t, prev)
	assert.True(t, written)

	_, prev, written, err = Install(dir, Content("1.0.0"))
	require.NoError(t, err)
	assert.Equal(t, Version("1.0.0"), prev)
	assert.False(t, written, "same version is not rewritten")

	_, prev, written, err = Install(dir, Content("1.1.0"))
	require.NoError(t, err)
	assert.Equal(t, Version("1.0.0"), prev)
	assert.True(t, written)

	b, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, Version("1.1.0"), InstalledVersion(string(b)))
}

func TestCheck(t *testing.T) {
	missing := Check("1.0.0", "", false)
	assert.False(t, missing.UpToDate)
	assert.Equal(t, Content("1.0.0"), missing.Content)
	assert.Equal(t, InstallPath, missing.InstallPath)

	current := Check("1.0.0", Version("1.0.0"), false)
	assert.True(t, current.UpToDate)
	assert.Empty(t, current.Content)

	forced := Check("1.0.0", Version("1.0.0"), true)
	assert.NotEmpty(t, forced.Content)

	outdated := Check("1.1.0", Version("1.0.0"), false)
	assert.False(t, outdated.UpToDate)
	assert.Contains(t, outdated.Instructions, "updated")
	assert.NotEmpty(t, outdated.Content)
}
