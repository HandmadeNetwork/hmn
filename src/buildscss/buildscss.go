package buildscss

/*

var compressed bool

func init() {
	libsass.RegisterSassFunc("base64($filename)", func(ctx context.Context, in libsass.SassValue) (*libsass.SassValue, error) {
		var filename string
		err := libsass.Unmarshal(in, &filename)
		if err != nil {
			return nil, err
		}

		fileBytes, err := os.ReadFile(filename)
		if err != nil {
			return nil, err
		}

		encoded, _ := libsass.Marshal(base64.StdEncoding.EncodeToString(fileBytes))
		return &encoded, nil
	})

	buildCommand := &cobra.Command{
		Use:   "buildscss",
		Short: "Build the website CSS",
		Run: func(cmd *cobra.Command, args []string) {
			style := libsass.NESTED_STYLE
			if compressed {
				style = libsass.COMPRESSED_STYLE
			}

			err := compile("src/rawdata/scss/style.scss", "public/style.css", "light", style)
			if err != nil {
				fmt.Println(color.Bold + color.Red + "Failed to compile main SCSS." + color.Reset)
				fmt.Println(err)
				os.Exit(1)
			}

			for _, theme := range []string{"light", "dark"} {
				err := compile("src/rawdata/scss/theme.scss", fmt.Sprintf("public/themes/%s/theme.css", theme), theme, style)
				if err != nil {
					fmt.Println(color.Bold + color.Red + "Failed to compile theme SCSS." + color.Reset)
					fmt.Println(err)
					os.Exit(1)
				}
			}
		},
	}
	buildCommand.Flags().BoolVar(&compressed, "compressed", false, "Minify the output CSS")

	website.WebsiteCommand.AddCommand(buildCommand)
}

func compile(inpath, outpath string, theme string, style int) error {
	err := os.MkdirAll(filepath.Dir(outpath), 0755)
	if err != nil {
		panic(oops.New(err, "failed to create directory for CSS file"))
	}

	outfile, err := os.OpenFile(outpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		panic(oops.New(err, "failed to open CSS file for writing"))
	}
	defer outfile.Close()

	infile, err := os.Open(inpath)
	if err != nil {
		panic(oops.New(err, "failed to open SCSS file"))
	}
	compiler, err := libsass.New(outfile, infile,
		libsass.IncludePaths([]string{
			"src/rawdata/scss",
			fmt.Sprintf("src/rawdata/scss/themes/%s", theme),
		}),
		libsass.OutputStyle(style),
	)
	if err != nil {
		panic(oops.New(err, "failed to create SCSS compiler"))
	}

	return compiler.Run()
}

*/
