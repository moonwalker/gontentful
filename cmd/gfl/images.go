package main

import "github.com/spf13/cobra"

const (
	cflUrl = "https://imagedelivery.net/"
)

var (
	variantId,
	filename,
	apiKey,
	folder,
	method,
	accountId string
	uploadCmd = &cobra.Command{
		Use:   "upsert",
		Short: "upload images to cloudflare",
		PreRun: func(cmd *cobra.Command, args []string) {
			cmd.MarkFlagRequired("brand")
			cmd.MarkFlagRequired("accountId")
			cmd.MarkFlagRequired("apiKey")
		},
		Run: func(cmd *cobra.Command, args []string) {
			upsertImage()
		},
	}
)

func init() {
	// Required flags
	uploadCmd.Flags().StringVarP(&brand, "brand", "b", "", "brand to upload images to")
	uploadCmd.Flags().StringVarP(&accountId, "accountId", "a", "", "account Id")
	uploadCmd.Flags().StringVarP(&apiKey, "apiKey", "k", "", "cloudflare image upload api key")
	// Optional flags
	uploadCmd.Flags().StringVarP(&filename, "image", "i", "", "name of image file")
	uploadCmd.Flags().StringVarP(&folder, "folder", "f", "", "folder of images to be uploaded if different from the default 'input/images'")
	uploadCmd.Flags().StringVarP(&variantId, "variantId", "v", "", "variant image settings")
	uploadCmd.Flags().StringVarP(&method, "method", "m", "", "<f|u> f: physical file(s) from local, u: url of image(s)")
	rootCmd.AddCommand(uploadCmd)
}
