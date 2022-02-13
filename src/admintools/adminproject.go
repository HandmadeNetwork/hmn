package admintools

import (
	"context"
	"fmt"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/parsing"
	"github.com/spf13/cobra"
)

func addProjectCommands(adminCommand *cobra.Command) {
	projectCommand := &cobra.Command{
		Use:   "project",
		Short: "Admin commands for managing projects",
	}
	adminCommand.AddCommand(projectCommand)

	addCreateProjectCommand(projectCommand)
	addProjectTagCommand(projectCommand)
}

func addCreateProjectCommand(projectCommand *cobra.Command) {
	createProjectCommand := &cobra.Command{
		Use:   "create",
		Short: "Create a new project",
		Run: func(cmd *cobra.Command, args []string) {
			name, _ := cmd.Flags().GetString("name")
			slug, _ := cmd.Flags().GetString("slug")
			blurb, _ := cmd.Flags().GetString("blurb")
			description, _ := cmd.Flags().GetString("description")
			personal, _ := cmd.Flags().GetBool("personal")
			userIDs, _ := cmd.Flags().GetIntSlice("userids")

			descParsed := parsing.ParseMarkdown(description, parsing.ForumRealMarkdown)

			ctx := context.Background()
			conn := db.NewConnPool(1, 1)
			defer conn.Close()

			tx, err := conn.Begin(ctx)
			if err != nil {
				panic(err)
			}
			defer tx.Rollback(ctx)

			p, err := hmndata.FetchProject(ctx, tx, nil, models.HMNProjectID, hmndata.ProjectsQuery{
				Lifecycles:    models.AllProjectLifecycles,
				IncludeHidden: true,
			})
			if err != nil {
				panic(err)
			}
			hmn := p.Project

			newProjectID, err := db.QueryInt(ctx, tx,
				`
				INSERT INTO handmade_project (
					slug,
					name,
					blurb,
					description,
					color_1,
					color_2,
					featured,
					hidden,
					descparsed,
					blog_enabled,
					forum_enabled,
					all_last_updated,
					annotation_last_updated,
					blog_last_updated,
					forum_last_updated,
					lifecycle,
					date_approved,
					date_created,
					bg_flags,
					library_enabled,
					personal
				) VALUES (
					$1,							-- slug
					$2,							-- name
					$3,							-- blurb
					$4,							-- description
					$5,							-- color_1
					$6,							-- color_2
					FALSE,						-- featured
					FALSE,						-- hidden
					$7,							-- descparsed
					FALSE,						-- blog_enabled
					FALSE,						-- forum_enabled
					NOW(),						-- all_last_updated
					'epoch',					-- annotation_last_updated
					'epoch',					-- blog_last_updated
					'epoch',					-- forum_last_updated
					$8,							-- lifecycle
					NOW(),						-- date_approved
					NOW(),						-- date_created
					0,							-- bg_flags
					FALSE,						-- library_enabled
					$9							-- personal
				)
				RETURNING id
				`,
				slug,
				name,
				blurb,
				description,
				hmn.Color1,
				hmn.Color2,
				descParsed,
				models.ProjectLifecycleActive,
				personal,
			)
			if err != nil {
				panic(err)
			}

			for _, userID := range userIDs {
				_, err := tx.Exec(ctx,
					`
					INSERT INTO handmade_user_projects (user_id, project_id)
					VALUES ($1, $2)
					`,
					userID,
					newProjectID,
				)
				if err != nil {
					panic(err)
				}
			}

			err = tx.Commit(ctx)
			if err != nil {
				panic(err)
			}

			fmt.Printf("Created new project with id: %d\n", newProjectID)
		},
	}
	createProjectCommand.Flags().String("name", "", "")
	createProjectCommand.Flags().String("slug", "", "")
	createProjectCommand.Flags().String("blurb", "", "")
	createProjectCommand.Flags().String("description", "", "")
	createProjectCommand.Flags().Bool("personal", true, "")
	createProjectCommand.Flags().IntSlice("userids", nil, "")
	createProjectCommand.MarkFlagRequired("name")
	createProjectCommand.MarkFlagRequired("userids")
	projectCommand.AddCommand(createProjectCommand)
}

func addProjectTagCommand(projectCommand *cobra.Command) {
	projectTagCommand := &cobra.Command{
		Use:   "tag",
		Short: "Create or update a project's tag",
		Run: func(cmd *cobra.Command, args []string) {
			projectID, _ := cmd.Flags().GetInt("projectid")
			tag, _ := cmd.Flags().GetString("tag")

			ctx := context.Background()
			conn := db.NewConnPool(1, 1)
			defer conn.Close()

			resultTag, err := hmndata.SetProjectTag(ctx, conn, nil, projectID, tag)
			if err != nil {
				panic(err)
			}

			if resultTag == nil {
				fmt.Printf("Project tag was deleted.\n")
			} else {
				fmt.Printf("Project now has tag: %s\n", tag)
			}
		},
	}
	projectTagCommand.Flags().Int("projectid", 0, "")
	projectTagCommand.Flags().String("tag", "", "")
	projectTagCommand.MarkFlagRequired("projectid")
	projectTagCommand.MarkFlagRequired("tag")
	projectCommand.AddCommand(projectTagCommand)
}
