package discord

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCleanUpMarkdown(t *testing.T) {
	t.Skip("Skipping these tests because they are server-specific and make network requests. Feel free to re-enable, but don't commit :)")

	const userBen = "<@!132715550571888640>"
	const channelShowcaseTest = "<#759497527883202582>"
	const roleHmnMember = "<@&876685379770646538>"

	t.Run("normal behavior", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		assert.Equal(t, "@Frogbot some stuff", CleanUpMarkdown(ctx, "<@!745051593728196732> some stuff"))
		assert.Equal(t,
			"users: @Unknown User @bvisness @bvisness, channels: #Unknown Channel #showcase-test #showcase-test, roles: @Unknown Role @HMN Member @HMN Member, :shakefist: also normal text",
			CleanUpMarkdown(ctx, fmt.Sprintf("users: <@!000000> %s %s, channels: <#000000> %s %s, roles: <@&000000> %s %s, <a:shakefist:798333915973943307> also normal text", userBen, userBen, channelShowcaseTest, channelShowcaseTest, roleHmnMember, roleHmnMember)),
		)
	})
	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // immediately cancel

		assert.Equal(t,
			"@Unknown User #Unknown Channel @Unknown Role",
			CleanUpMarkdown(ctx, fmt.Sprintf("%s %s %s", userBen, channelShowcaseTest, roleHmnMember)),
		)
	})
}
