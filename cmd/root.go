package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/harrybrwn/go-canvas/canvas"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Execute will execute the root comand on the cli
func Execute() (err error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$XDG_CONFIG_HOME/canvas")
	viper.AddConfigPath("$HOME/.config/canvas")
	viper.AddConfigPath("$HOME/.canvas")
	viper.SetEnvPrefix("canvas")
	viper.BindEnv("token")

	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	root.AddCommand(newFilesCmd(), newConfigCmd(), coursesCmd)
	if err = root.Execute(); err != nil {
		return err
	}
	return nil
}

var root = &cobra.Command{
	Use: "canvas",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		host := viper.GetString("host")
		if host != "" {
			canvas.DefaultHost = host
		}
	},
}

func newFilesCmd() *cobra.Command {
	var (
		contentType string
		sortby      = []string{"created_at"}
	)
	c := &cobra.Command{
		Use:   "files",
		Short: "This is a garbage command lol.",
		RunE: func(cmd *cobra.Command, args []string) error {
			token := viper.GetString("token")
			c := canvas.FromToken(token)
			courses, err := c.ActiveCourses()
			if err != nil {
				return err
			}

			opts := []canvas.Option{canvas.SortOpt(sortby...)}
			if contentType != "" {
				opts = append(opts, canvas.ContentType(contentType))
			}
			for _, course := range courses {
				course.SetErrorHandler(func(e error, stop chan int) {
					if e != nil {
						stop <- 1
						fmt.Println("Error: " + e.Error())
						os.Exit(1)
					}
				})
				files := course.Files(opts...)
				for f := range files {
					fmt.Println(f.CreatedAt, f.Size, f.Filename)
				}
			}
			return nil
		},
	}
	c.Flags().StringVarP(&contentType, "content-type", "c", "", "filter out files by content type (ex. application/pdf)")
	c.Flags().StringArrayVarP(&sortby, "sortyby", "s", sortby, "how the files should be sorted")
	return c
}

var coursesCmd = &cobra.Command{
	Use:   "courses",
	Short: "Show info on courses",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

func newConfigCmd() *cobra.Command {
	var file, edit bool
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			f := viper.ConfigFileUsed()
			if file {
				cmd.Println(f)
				return nil
			}
			if edit {
				editor := os.Getenv("EDITOR")
				ex := exec.Command(editor, f)
				ex.Stdout = os.Stdout
				ex.Stdin = os.Stdin
				ex.Stderr = os.Stderr
				return ex.Run()
			}
			return cmd.Usage()
		},
	}
	cmd.Flags().BoolVarP(&edit, "edit", "e", false, "edit the config file")
	cmd.Flags().BoolVarP(&file, "file", "f", false, "print the config file path")
	return cmd
}
