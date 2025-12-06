package assets

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeFilename(t *testing.T) {
	assert.Equal(t, "cool_filename.txt.wow", SanitizeFilename("cool filename.txt.wow"))
	assert.Equal(t, "__hi_doggy__", SanitizeFilename("😎 hi doggy 🐶"))
	assert.Equal(t, "newlines_aretotallylegal", SanitizeFilename("newlines\naretotallylegal"))
}
