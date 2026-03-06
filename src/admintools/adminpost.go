package admintools

import (
	"context"
	"fmt"
	"strings"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/spf13/cobra"
)

func addPostCommands(adminCommand *cobra.Command) {
	postCommand := &cobra.Command{
		Use:   "post",
		Short: "Admin commands for managing posts",
	}
	adminCommand.AddCommand(postCommand)

	addRegeneratePreviewsCommand(postCommand)
}

func addRegeneratePreviewsCommand(postCommand *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "regeneratepreviews",
		Short: "Regenerate plain text and HTML previews for some or all posts",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			conn := db.NewConn()
			defer conn.Close(ctx)

			type row struct {
				Post           models.Post        `db:"post"`
				Thread         models.Thread      `db:"thread"`
				CurrentVersion models.PostVersion `db:"ver"`
			}

			var qb db.QueryBuilder
			qb.Add(`
				SELECT $columns
				FROM
					post
					JOIN thread ON post.thread_id = thread.id
					JOIN post_version AS ver ON ver.id = post.current_id
			`)
			if len(args) > 0 {
				qb.Add(`WHERE post.id = ANY ($?)`, args)
			}
			allPostsAndVersions, err := db.Query[row](ctx, conn, qb.String(), qb.Args()...)
			if err != nil {
				panic(oops.New(err, "failed to fetch all posts and their current versions"))
			}

			var errs []error
			for _, p := range allPostsAndVersions {
				inlinePreview := p.CurrentVersion.TextRaw[:min(60, len(p.CurrentVersion.TextRaw))]
				inlinePreview = strings.ReplaceAll(inlinePreview, "\r\n", " ")
				inlinePreview = strings.ReplaceAll(inlinePreview, "\n", " ")
				fmt.Printf("(%d) %s // %s...", p.Post.ID, p.Thread.Title, inlinePreview)

				plain, html := hmndata.GeneratePostPreviews(p.CurrentVersion.TextRaw)
				_, err = conn.Exec(ctx,
					`
					UPDATE post
					SET preview = $1, preview_html = $2
					WHERE id = $3
					`,
					plain,
					html,
					p.Post.ID,
				)
				if err != nil {
					fmt.Printf("FAIL\n")
					errs = append(errs, oops.New(err, "for post %d", p.Post.ID))
					continue
				}

				fmt.Printf("ok\n")
			}

			if len(errs) > 0 {
				fmt.Printf("!!!!!!!!!!!!!!!!\n")
				fmt.Printf("!!!  ERRORS  !!!\n")
				fmt.Printf("!!!!!!!!!!!!!!!!\n")
				for _, err := range errs {
					fmt.Println(err)
				}
			}
		},
	}
	postCommand.AddCommand(cmd)
}
