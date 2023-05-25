package main

import (
	"regexp"

	"github.com/spf13/cobra"
)

var (
	ctTransform = false
	direction,
	contentType,
	brand,
	repo string
	snake        = regexp.MustCompile(`([_ ]\w)`)
	transformCmd = &cobra.Command{
		Use:   "trans",
		Short: "transform content model / content",
		PreRun: func(cmd *cobra.Command, args []string) {
			if direction == "tocf" {
				cmd.MarkFlagRequired("repo")
			} else {
				rootCmd.MarkPersistentFlagRequired("space")
				rootCmd.MarkPersistentFlagRequired("token")
				rootCmd.MarkPersistentFlagRequired("cma")
				if !ctTransform {
					cmd.MarkFlagRequired("brand")
				}
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			switch direction {
			case "tocf":
				if ctTransform {
					formatContentType()
				} else {
					formatContent()
				}
			case "fromcf":
				if ctTransform {
					transformContentType()
				} else {
					transformContent()
				}
			}
		},
	}
)

func init() {
	// contentType to migrate
	transformCmd.Flags().BoolVarP(&ctTransform, "ct", "f", false, "content type transform")
	transformCmd.Flags().StringVarP(&contentType, "contentModel", "m", "", "type of the content to migrate")
	transformCmd.Flags().StringVarP(&repo, "repo", "r", "", "repo of the content to migrate")
	transformCmd.Flags().StringVarP(&brand, "brand", "b", "", "brand")
	transformCmd.PersistentFlags().StringVarP(&direction, "direction", "d", "", "directions: <fromcf|tocf>")
	transformCmd.MarkPersistentFlagRequired("direction")
	rootCmd.AddCommand(transformCmd)
}
