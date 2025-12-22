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
	batchUploadCmd = &cobra.Command{
		Use:   "batchupsert",
		Short: "batch upload images to cloudflare",
		PreRun: func(cmd *cobra.Command, args []string) {
			cmd.MarkFlagRequired("accountId")
			cmd.MarkFlagRequired("apiKey")
			cmd.MarkFlagRequired("brand")
		},
		Run: func(cmd *cobra.Command, args []string) {
			batchUpsertImages()
		},
	}
	deleteAllCmd = &cobra.Command{
		Use:   "deleteall",
		Short: "delete all images from cloudflare",
		Long:  "Delete all images from your Cloudflare Images account. This operation is irreversible.",
		PreRun: func(cmd *cobra.Command, args []string) {
			cmd.MarkFlagRequired("accountId")
			cmd.MarkFlagRequired("apiKey")
		},
		Run: func(cmd *cobra.Command, args []string) {
			deleteAllImages()
		},
	}
	uploadVideoCmd = &cobra.Command{
		Use:   "uploadvideo",
		Short: "upload video to cloudflare",
		PreRun: func(cmd *cobra.Command, args []string) {
			cmd.MarkFlagRequired("accountId")
			cmd.MarkFlagRequired("apiKey")
			cmd.MarkFlagRequired("brand")
		},
		Run: func(cmd *cobra.Command, args []string) {
			uploadVideo()
		},
	}
)

func init() {
	initUploadCmd()
	initBatchUploadCmd()
	initDeleteAllCmd()
	initUploadVideoCmd()
}

func initUploadCmd() {
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

func initBatchUploadCmd() {
	// Required flags
	batchUploadCmd.Flags().StringVarP(&brand, "brand", "b", "", "brand to upload images to")
	batchUploadCmd.Flags().StringVarP(&accountId, "accountId", "a", "", "account Id")
	batchUploadCmd.Flags().StringVarP(&apiKey, "apiKey", "k", "", "cloudflare image upload api key")

	// Optional flags
	batchUploadCmd.Flags().StringVarP(&folder, "folder", "f", "", "folder of images to be uploaded if different from the default 'input/images'")
	rootCmd.AddCommand(batchUploadCmd)
}

func initDeleteAllCmd() {
	// Required flags
	deleteAllCmd.Flags().StringVarP(&accountId, "accountId", "a", "", "account Id")
	deleteAllCmd.Flags().StringVarP(&apiKey, "apiKey", "k", "", "cloudflare image upload api key")
	rootCmd.AddCommand(deleteAllCmd)
}

func initUploadVideoCmd() {
	// Required flags
	uploadVideoCmd.Flags().StringVarP(&brand, "brand", "b", "", "brand to upload video to")
	uploadVideoCmd.Flags().StringVarP(&accountId, "accountId", "a", "", "account Id")
	uploadVideoCmd.Flags().StringVarP(&apiKey, "apiKey", "k", "", "cloudflare image upload api key")

	// Optional flags
	uploadVideoCmd.Flags().StringVarP(&filename, "video", "i", "", "name of video file")
	uploadVideoCmd.Flags().StringVarP(&folder, "folder", "f", "", "folder of videos to be uploaded if different from the default 'input/images'")
	rootCmd.AddCommand(uploadVideoCmd)
}
