package cmd

import (
	"fmt"
	"os"

	"git.handmade.network/hmn/hmn/src/buildcss"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/website"
	"github.com/spf13/cobra"
)

func init() {
	buildCommand := &cobra.Command{
		Use:   "buildcss",
		Short: "Build the website CSS",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, err := buildcss.BuildContext()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			res := ctx.Rebuild()
			outputFilenames := make([]string, 0)
			for _, o := range res.OutputFiles {
				outputFilenames = append(outputFilenames, o.Path)
			}
			logging.Info().
				Interface("Errors", res.Errors).
				Interface("Warnings", res.Warnings).
				Msg("Ran esbuild")
			if len(outputFilenames) > 0 {
				logging.Info().Interface("Files", outputFilenames).Msg("Wrote files")
			}
		},
	}
	website.WebsiteCommand.AddCommand(buildCommand)
}
