package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/moonwalker/gontentful"
)

const (
	batchUrl    = "https://api.cloudflare.com/client/v4/accounts/%s/images/v1/batch_token"
	batchApiUrl = "https://batch.imagedelivery.net/images/v1" // Different endpoint for batch operations
	imagesDir   = "./images"                                  // Change to your images directory
)

type UploadResponse struct {
	Result struct {
		ID string `json:"id"`
	} `json:"result"`
	Success bool `json:"success"`
	Errors  []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type BatchTokenResponse struct {
	Result struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	} `json:"result"`
	Success bool `json:"success"`
	Errors  []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type ImageDetailsResponse struct {
	Errors   []interface{} `json:"errors"`
	Messages []interface{} `json:"messages"`
	Result   struct {
		Filename string `json:"filename"`
		ID       string `json:"id"`
		Meta     struct {
			Key string `json:"key"`
		} `json:"meta"`
		RequireSignedURLs bool      `json:"requiredSignedURLs"`
		Uploaded          time.Time `json:"uploaded"`
		Variants          []string  `json:"variants"`
	} `json:"result"`
	Success bool `json:"success"`
}

// ImageListResponse represents the response from Cloudflare Images list API
type ImageListResponse struct {
	Errors   []interface{} `json:"errors"`
	Messages []interface{} `json:"messages"`
	Result   struct {
		Images []struct {
			ID       string `json:"id"`
			Filename string `json:"filename"`
			Meta     struct {
				Key string `json:"key"`
			} `json:"meta"`
			RequireSignedURLs bool      `json:"requireSignedURLs"`
			Uploaded          time.Time `json:"uploaded"`
			Variants          []string  `json:"variants"`
		} `json:"images"`
		ContinuationToken string `json:"continuation_token,omitempty"`
	} `json:"result"`
	Success bool `json:"success"`
}

// VideoUploadResponse represents the response from Cloudflare Stream video upload API
type VideoUploadResponse struct {
	Errors   []interface{} `json:"errors"`
	Messages []interface{} `json:"messages"`
	Result   struct {
		UID                   string  `json:"uid"`
		Thumbnail             string  `json:"thumbnail"`
		ThumbnailTimestampPct float64 `json:"thumbnailTimestampPct"`
		ReadyToStream         bool    `json:"readyToStream"`
		Status                struct {
			State string `json:"state"`
		} `json:"status"`
		Meta struct {
			Name string `json:"name"`
		} `json:"meta"`
		Created            time.Time `json:"created"`
		Modified           time.Time `json:"modified"`
		Size               int64     `json:"size"`
		Preview            string    `json:"preview"`
		AllowedOrigins     []string  `json:"allowedOrigins"`
		RequireSignedURLs  bool      `json:"requireSignedURLs"`
		Uploaded           time.Time `json:"uploaded"`
		UploadExpiry       time.Time `json:"uploadExpiry"`
		MaxSizeBytes       int64     `json:"maxSizeBytes"`
		MaxDurationSeconds int       `json:"maxDurationSeconds"`
		Duration           float64   `json:"duration"`
		Input              struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"input"`
		Playback struct {
			HLS  string `json:"hls"`
			Dash string `json:"dash"`
		} `json:"playback"`
	} `json:"result"`
	Success bool `json:"success"`
}

// VideoDetailsResponse represents the response from Cloudflare Stream video details API
type VideoDetailsResponse struct {
	Errors   []interface{} `json:"errors"`
	Messages []interface{} `json:"messages"`
	Result   struct {
		UID                   string  `json:"uid"`
		Thumbnail             string  `json:"thumbnail"`
		ThumbnailTimestampPct float64 `json:"thumbnailTimestampPct"`
		ReadyToStream         bool    `json:"readyToStream"`
		Status                struct {
			State string `json:"state"`
		} `json:"status"`
		Meta struct {
			Name string `json:"name"`
		} `json:"meta"`
		Created            time.Time `json:"created"`
		Modified           time.Time `json:"modified"`
		Size               int64     `json:"size"`
		Preview            string    `json:"preview"`
		AllowedOrigins     []string  `json:"allowedOrigins"`
		RequireSignedURLs  bool      `json:"requireSignedURLs"`
		Uploaded           time.Time `json:"uploaded"`
		UploadExpiry       time.Time `json:"uploadExpiry"`
		MaxSizeBytes       int64     `json:"maxSizeBytes"`
		MaxDurationSeconds int       `json:"maxDurationSeconds"`
		Duration           float64   `json:"duration"`
		Input              struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"input"`
		Playbook struct {
			HLS  string `json:"hls"`
			Dash string `json:"dash"`
		} `json:"playbook"`
	} `json:"result"`
	Success bool `json:"success"`
}

// TokenManager manages batch tokens with automatic refresh
// Batch tokens typically expire after 60 minutes, but we use a 5-minute buffer for safety
type TokenManager struct {
	mu        sync.RWMutex
	token     string
	expiresAt time.Time
}

var tokenManager = &TokenManager{}
var batchToken string

// GetValidBatchToken returns a valid batch token, refreshing if necessary
func (tm *TokenManager) GetValidBatchToken() (string, error) {
	tm.mu.RLock()
	// Check if we have a valid token (with 5-minute buffer before expiration)
	if tm.token != "" && time.Now().Add(5*time.Minute).Before(tm.expiresAt) {
		token := tm.token
		tm.mu.RUnlock()
		return token, nil
	}
	tm.mu.RUnlock()

	// Need to refresh token
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Double-check in case another goroutine already refreshed
	if tm.token != "" && time.Now().Add(5*time.Minute).Before(tm.expiresAt) {
		return tm.token, nil
	}

	// Get new token
	newToken, expiresAt, err := getBatchToken()
	if err != nil {
		// If batch token fails, fall back to using API key directly
		fmt.Printf("⚠️  Batch token failed (%v), falling back to direct API key\n", err)
		tm.token = "FALLBACK_TO_APIKEY"
		tm.expiresAt = time.Now().Add(24 * time.Hour) // Pretend it expires in 24 hours
		return apiKey, nil
	}

	tm.token = newToken
	tm.expiresAt = expiresAt
	fmt.Printf("🔄 Refreshed batch token (expires at: %s)\n", expiresAt.Format(time.RFC3339))
	return newToken, nil
}

// IsTokenValid checks if the current token is still valid
func (tm *TokenManager) IsTokenValid() bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.token != "" && time.Now().Add(5*time.Minute).Before(tm.expiresAt)
}

// GetTokenExpiryTime returns when the current token expires
func (tm *TokenManager) GetTokenExpiryTime() time.Time {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.expiresAt
}

// InvalidateToken forces the token to be refreshed on the next request
func (tm *TokenManager) InvalidateToken() {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.token = ""
	tm.expiresAt = time.Time{}
	fmt.Println("🗑️  Token invalidated")
}

// getBatchToken requests a new batch token from Cloudflare API
func getBatchToken() (string, time.Time, error) {
	url := fmt.Sprintf(batchUrl, accountId)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", time.Time{}, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", time.Time{}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Add debugging for 404 errors specifically
	if resp.StatusCode == 404 {
		fmt.Printf("🔍 DEBUG - 404 Error Details:\n")
		fmt.Printf("   URL: %s\n", url)
		fmt.Printf("   Method: GET\n")
		fmt.Printf("   Headers: %v\n", req.Header)
		fmt.Printf("   Response: %s\n", string(body))
	}

	// Check for non-2xx status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", time.Time{}, fmt.Errorf("HTTP %d error getting batch token: %s", resp.StatusCode, string(body))
	}

	var tokenResp BatchTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to parse token response JSON: %v\nRaw response: %s", err, string(body))
	}
	if !tokenResp.Success {
		return "", time.Time{}, fmt.Errorf("failed to get batch token: %v", tokenResp.Errors)
	}

	return tokenResp.Result.Token, tokenResp.Result.ExpiresAt, nil
}

// listAllImages fetches all images from Cloudflare Images using v2 API with pagination
func listAllImages() ([]string, error) {
	var allImageIDs []string
	continuationToken := ""

	for {
		// Get current token
		token, err := tokenManager.GetValidBatchToken()
		if err != nil {
			return nil, fmt.Errorf("failed to get token for listing images: %w", err)
		}

		// Build URL with pagination
		var url string
		if token == apiKey {
			// Using regular API (v2)
			url = fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/images/v2", accountId)
		} else {
			// Batch API might not support listing, fallback to regular API
			url = fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/images/v2", accountId)
			token = apiKey // Force use of API key for listing
		}

		if continuationToken != "" {
			url += "?continuation_token=" + continuationToken
		}

		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("http request failed: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("HTTP %d error listing images: %s", resp.StatusCode, string(body))
		}

		var listResp ImageListResponse
		if err := json.Unmarshal(body, &listResp); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		if !listResp.Success {
			return nil, fmt.Errorf("API returned error: %v", listResp.Errors)
		}

		// Collect image IDs from this page
		for _, img := range listResp.Result.Images {
			allImageIDs = append(allImageIDs, img.ID)
		}

		// Check if there are more pages
		if listResp.Result.ContinuationToken == "" {
			break
		}
		continuationToken = listResp.Result.ContinuationToken

		fmt.Printf("📄 Fetched %d images, continuing with pagination...\n", len(listResp.Result.Images))
	}

	return allImageIDs, nil
}

// deleteAllImages deletes all images using batch token where possible
func deleteAllImages() error {
	fmt.Println("🔍 Fetching list of all images...")

	imageIDs, err := listAllImages()
	if err != nil {
		return fmt.Errorf("failed to list images: %w", err)
	}

	if len(imageIDs) == 0 {
		fmt.Println("📁 No images found to delete")
		return nil
	}

	fmt.Printf("🗑️  Found %d images to delete\n", len(imageIDs))
	fmt.Println("⚠️  WARNING: This will permanently delete ALL images from your Cloudflare Images account!")
	fmt.Println("Press Ctrl+C to cancel, or any other key to continue...")

	// Simple confirmation - wait for user input
	var input string
	fmt.Scanln(&input)

	fmt.Println("🚀 Starting batch deletion...")

	// Get token once at the beginning for efficiency
	token, err := tokenManager.GetValidBatchToken()
	if err != nil {
		return fmt.Errorf("failed to get initial token: %w", err)
	}

	// Determine endpoint based on token type
	var useRegularAPI bool
	if token == apiKey {
		useRegularAPI = true
		fmt.Println("🔑 Using regular API endpoint for deletions")
	} else {
		useRegularAPI = false
		fmt.Println("Expiry:", tokenManager.GetTokenExpiryTime())
		fmt.Printf("⚡ Using batch API endpoint for deletions (token expires: %s)\n",
			tokenManager.GetTokenExpiryTime().Format(time.RFC3339))
	}

	successCount := 0
	errorCount := 0
	tokenRefreshCount := 0
	startTime := time.Now()

	for i, imageID := range imageIDs {
		fmt.Printf("🗑️  [%d/%d] Deleting image %s...\n", i+1, len(imageIDs), imageID)

		// Check if we need to refresh token (every 50 operations or if expired)
		// if i > 0 && (i%50 == 0 || !tokenManager.IsTokenValid()) {
		// 	fmt.Printf("🔄 Checking token validity (operation %d)...\n", i+1)
		// 	newToken, err := tokenManager.GetValidBatchToken()
		// 	if err != nil {
		// 		fmt.Printf("⚠️  Token refresh failed: %v, continuing with existing token\n", err)
		// 	} else if newToken != token {
		// 		token = newToken
		// 		tokenRefreshCount++
		// 		if token == apiKey {
		// 			useRegularAPI = true
		// 			fmt.Println("🔄 Switched to regular API endpoint")
		// 		} else {
		// 			useRegularAPI = false
		// 			fmt.Printf("🔄 Refreshed batch token (expires: %s)\n",
		// 				tokenManager.GetTokenExpiryTime().Format(time.RFC3339))
		// 		}
		// 	}
		// }

		if err := deleteImageOptimized(imageID, token, useRegularAPI); err != nil {
			fmt.Printf("❌ Error deleting %s: %v\n", imageID, err)
			errorCount++
		} else {
			fmt.Printf("✅ Successfully deleted %s\n", imageID)
			successCount++
		}
	}

	elapsed := time.Since(startTime)
	fmt.Printf("\n✨ Batch deletion complete in %s!\n", elapsed.Round(time.Second))
	fmt.Printf("✅ Deleted: %d images\n", successCount)
	if errorCount > 0 {
		fmt.Printf("❌ Errors: %d images\n", errorCount)
	}
	if tokenRefreshCount > 0 {
		fmt.Printf("🔄 Token refreshes: %d\n", tokenRefreshCount)
	}

	return nil
}

// deleteImageOptimized deletes a single image using a pre-fetched token (optimized for bulk operations)
func deleteImageOptimized(imageID, token string, useRegularAPI bool) error {
	// Choose the appropriate endpoint
	var url string
	if useRegularAPI {
		// Using regular API
		url = fmt.Sprintf(apiurl, accountId) + "/" + imageID
	} else {
		// Using batch API
		url = fmt.Sprintf("https://batch.imagedelivery.net/images/v1/%s", imageID)
	}

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("delete request failed: %w", err)
	}
	defer resp.Body.Close()

	// If 404, image was already deleted or doesn't exist
	if resp.StatusCode == 404 {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read delete response: %w", err)
	}

	// Check for errors
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d error deleting image: %s", resp.StatusCode, string(body))
	}

	return nil
}

func uploadImage(filePath string) error {
	f := 0
	// Extract image ID from filename (without extension)
	fileName := gontentful.GetCloudflareImagesID(brand) + "/" + filepath.Base(filePath)
	faultyName := gontentful.GetCloudflareImagesID(brand) + filepath.Base(filePath)
	imageID := strings.TrimSuffix(fileName, filepath.Ext(fileName))

	// Check for faulty name (missing slash)
	fExists, err := imageExists(faultyName)
	if err != nil {
		fmt.Printf("⚠️  Warning: Failed to check for faulty name existence: %v\n", err)
	}
	if fExists {
		f++
		delImage(faultyName)
	}

	// Check if image already exists
	exists, err := imageExists(imageID)
	if err != nil {
		fmt.Printf("⚠️  Warning: Failed to check image existence: %v\n", err)
	}
	if !exists {
		file, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer file.Close()

		// Check if file is empty
		stat, err := file.Stat()
		if err != nil {
			return fmt.Errorf("failed to get file stats: %w", err)
		}
		if stat.Size() == 0 {
			return fmt.Errorf("file is empty: %s", filePath)
		}

		// Prepare upload parameters
		form := map[string]string{"id": imageID}
		p := UploadImageParams{
			Id:       imageID,
			Metadata: form,
			Path:     filePath,
		}

		fmt.Printf("🔧 Creating form for file: %s (size: ", filePath)
		if stat, err := file.Stat(); err == nil {
			fmt.Printf("%d bytes)\n", stat.Size())
		} else {
			fmt.Printf("unknown)\n")
		}

		ct, payload, err := createForm(p)
		if err != nil {
			return fmt.Errorf("failed to create form data: %w", err)
		}

		// Debug: Check if payload has data
		if buf, ok := payload.(*bytes.Buffer); ok {
			fmt.Printf("🔧 Form data size: %d bytes\n", buf.Len())
		}

		// Retry logic for token expiration
		maxRetries := 2
		for attempt := 0; attempt < maxRetries; attempt++ {
			if attempt > 0 {
				batchToken, err = tokenManager.GetValidBatchToken()
			}
			// Use the token manager to get a valid token
			var uploadResp UploadResponse
			err = batchReq(http.MethodPost, batchApiUrl, batchToken, payload, &uploadResp, ct)

			if err != nil {
				fmt.Printf("❌ Error uploading image (attempt %d): %v\n", attempt+1, err)
				if strings.Contains(err.Error(), "incomplete multipart stream") {
					fmt.Printf("🔍 Debug info - Content-Type: %s, File: %s\n", ct, filePath)
					break // Don't retry on multipart errors
				}
				continue
			}

			if !uploadResp.Success {
				return fmt.Errorf("failed to upload %s: %v", filePath, uploadResp.Errors)
			}

			fmt.Printf("✅ Uploaded %s as ID %s\n", filepath.Base(filePath), uploadResp.Result.ID)
			return nil
		}

		return fmt.Errorf("failed to upload %s after %d attempts", filePath, maxRetries)
	}
	return nil
}

func uploadVideo() error {
	fmt.Println("🚀 Starting video upload...")

	dir := inputFolder
	if folder != "" {
		dir = folder
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		fmt.Printf("❌ Error reading directory: %v\n", err)
		return err
	}
	vidFiles := make([]string, 0)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if gontentful.IsVideoFile(file.Name()) {
			vidFiles = append(vidFiles, file.Name())
		}
	}

	if len(vidFiles) == 0 {
		fmt.Println("📁 No video files found in directory")
		return nil
	}

	fmt.Printf("📹 Found %d video files to upload\n", len(vidFiles))

	successCount := 0
	errorCount := 0
	startTime := time.Now()

	for i, fileName := range vidFiles {
		filePath := filepath.Join(dir, fileName)
		fmt.Printf("📤 [%d/%d] Uploading %s...\n", i+1, len(vidFiles), fileName)

		if err := uploadVideoFile(filePath); err != nil {
			fmt.Printf("❌ Error uploading %s: %v\n", fileName, err)
			errorCount++
		} else {
			successCount++
		}
	}

	elapsed := time.Since(startTime)
	fmt.Printf("\n✨ Video upload complete in %s!\n", elapsed.Round(time.Second))
	fmt.Printf("✅ Success: %d videos\n", successCount)
	if errorCount > 0 {
		fmt.Printf("❌ Errors: %d videos\n", errorCount)
	}
	if successCount > 0 {
		avgTime := elapsed / time.Duration(successCount)
		fmt.Printf("📊 Average time per successful upload: %s\n", avgTime.Round(time.Millisecond))
	}

	return nil
}

func batchUpsertImages() {
	fmt.Println("🚀 Starting batch image upload...")
	fmt.Printf("� Using API key authentication (no batch tokens needed)\n")

	dir := imagesDir
	if folder != "" {
		dir = folder
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		fmt.Printf("❌ Error reading directory: %v\n", err)
		return
	}

	imageFiles := make([]string, 0)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		ext := filepath.Ext(file.Name())
		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" {
			imageFiles = append(imageFiles, file.Name())
		}
	}

	if len(imageFiles) == 0 {
		fmt.Println("📁 No image files found in directory")
		return
	}

	fmt.Printf("📷 Found %d image files to upload\n", len(imageFiles))
	fmt.Printf("⚡ Using batch API endpoint: %s\n", batchApiUrl)

	successCount := 0
	errorCount := 0
	startTime := time.Now()

	batchToken, err = tokenManager.GetValidBatchToken()
	if err != nil {
		fmt.Printf("❌ Error getting batch token: %v\n", err)
	}

	for i, fileName := range imageFiles {
		filePath := filepath.Join(dir, fileName)
		fmt.Printf("📤 [%d/%d] Uploading %s...\n", i+1, len(imageFiles), fileName)

		err := uploadImage(filePath)
		if err != nil {
			fmt.Printf("❌ Error uploading %s: %v\n", fileName, err)
			errorCount++
		} else {
			successCount++
		}
	}

	elapsed := time.Since(startTime)
	fmt.Printf("\n✨ Batch upload complete in %s!\n", elapsed.Round(time.Second))
	fmt.Printf("✅ Success: %d files\n", successCount)
	if errorCount > 0 {
		fmt.Printf("❌ Errors: %d files\n", errorCount)
	}
	if successCount > 0 {
		avgTime := elapsed / time.Duration(successCount)
		fmt.Printf("📊 Average time per successful upload: %s\n", avgTime.Round(time.Millisecond))
	}
}

func batchReq(method, url, batchToken string, payload io.Reader, resp interface{}, contentType string) error {
	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", batchToken))
	if len(contentType) == 0 {
		req.Header.Add("Content-Type", "application/json")
	} else {
		req.Header.Add("Content-Type", contentType)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read http response body: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		resp := &errorResponse{}
		err = json.Unmarshal(body, &resp)
		if err != nil {
			return fmt.Errorf("failed to unmarshal error response: %w", err)
		}
		err = fmt.Errorf("error: %d", res.StatusCode)
		if len(resp.Errors) > 0 {
			err = fmt.Errorf("%w: %s", err, resp.Errors[0].Message)
		}
		return err
	}

	if resp != nil {
		err = json.Unmarshal(body, &resp)
		if err != nil {
			return fmt.Errorf("failed to unmarshal response body: %w", err)
		}
	}

	return nil
}

func uploadVideoFile(filePath string) error {
	// Extract video ID from filename (without extension)
	fileName := gontentful.GetCloudflareImagesID(brand) + "/" + filepath.Base(filePath)
	videoID := strings.TrimSuffix(fileName, filepath.Ext(fileName))

	// Check if video already exists
	exists, err := videoExists(videoID)
	if err != nil {
		return fmt.Errorf("failed to check video existence: %w", err)
	}
	if exists {
		fmt.Printf("🗑️  Video '%s' already exists, deleting before upload...\n", videoID)
		if err := deleteVideo(videoID); err != nil {
			return fmt.Errorf("failed to delete existing video '%s': %w", videoID, err)
		}
	}

	// Upload new video file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open video file: %w", err)
	}
	defer file.Close()

	// Check if file is empty
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file stats: %w", err)
	}
	if stat.Size() == 0 {
		return fmt.Errorf("file is empty: %s", filePath)
	}

	// Create multipart form
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add the video file
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return fmt.Errorf("failed to copy file data: %w", err)
	}

	// Add video ID as metadata
	err = writer.WriteField("uid", videoID)
	if err != nil {
		return fmt.Errorf("failed to write video ID field: %w", err)
	}

	writer.Close()

	// Create HTTP request
	streamApiUrl := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/stream", accountId)
	req, err := http.NewRequest("POST", streamApiUrl, &body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Execute request
	client := &http.Client{Timeout: 10 * time.Minute} // Videos can take longer to upload
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("video upload request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d error uploading video: %s", resp.StatusCode, string(respBody))
	}

	var uploadResp VideoUploadResponse
	if err := json.Unmarshal(respBody, &uploadResp); err != nil {
		return fmt.Errorf("failed to parse video upload response: %w", err)
	}

	if !uploadResp.Success {
		return fmt.Errorf("failed to upload video %s: %v", filePath, uploadResp.Errors)
	}

	fmt.Printf("✅ Uploaded video %s as ID %s (UID: %s)\n", filepath.Base(filePath), uploadResp.Result.UID, videoID)
	return nil
}

// videoExists checks if a video with the given UID exists in Cloudflare Stream
func videoExists(videoUID string) (bool, error) {
	streamApiUrl := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/stream/%s", accountId, videoUID)

	req, err := http.NewRequest("GET", streamApiUrl, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	// If 404, video doesn't exist
	if resp.StatusCode == 404 {
		return false, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, fmt.Errorf("HTTP %d error checking video existence: %s", resp.StatusCode, string(body))
	}

	var videoResp VideoDetailsResponse
	if err := json.Unmarshal(body, &videoResp); err != nil {
		return false, fmt.Errorf("failed to parse video details response: %w", err)
	}

	return videoResp.Success, nil
}

// deleteVideo deletes a video from Cloudflare Stream
func deleteVideo(videoUID string) error {
	streamApiUrl := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/stream/%s", accountId, videoUID)

	req, err := http.NewRequest("DELETE", streamApiUrl, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("delete request failed: %w", err)
	}
	defer resp.Body.Close()

	// If 404, video was already deleted or doesn't exist
	if resp.StatusCode == 404 {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read delete response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d error deleting video: %s", resp.StatusCode, string(body))
	}

	fmt.Printf("✅ Successfully deleted video '%s'\n", videoUID)
	return nil
}
