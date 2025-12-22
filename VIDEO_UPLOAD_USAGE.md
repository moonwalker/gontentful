# Cloudflare Stream Video Upload Implementation

This document explains how to use the new video upload functionality for Cloudflare Stream.

## Overview

The implementation adds video upload capabilities to your existing gfl CLI tool, following the Cloudflare Stream API specifications for basic video uploads (files under 200MB).

## API Implementation Details

Based on the Cloudflare Stream API documentation:
- **Endpoint**: `https://api.cloudflare.com/client/v4/accounts/{account_id}/stream`
- **Method**: POST
- **Content-Type**: `multipart/form-data`
- **Authentication**: Bearer token (API key)

## New Functions Added

### 1. `uploadVideoFile(filePath string) error`
Main function that handles uploading a single video file to Cloudflare Stream:
- Extracts video ID from filename using existing brand naming convention
- Checks if video already exists and deletes it if needed
- Creates multipart form with video file and metadata
- Uploads to Cloudflare Stream API
- Returns detailed error messages for debugging

### 2. `videoExists(videoUID string) (bool, error)`
Checks if a video with the given UID exists in Cloudflare Stream:
- Makes GET request to Stream API to check video existence
- Returns false for 404 responses (video doesn't exist)
- Handles other HTTP errors appropriately

### 3. `deleteVideo(videoUID string) error`
Deletes a video from Cloudflare Stream:
- Makes DELETE request to Stream API
- Handles 404 responses gracefully (already deleted)
- Provides success confirmation

## Data Structures Added

### `VideoUploadResponse`
Represents the response from Cloudflare Stream video upload API, including:
- Video UID
- Upload status
- Metadata (dimensions, duration, etc.)
- Playback URLs (HLS, DASH)
- Thumbnail information

### `VideoDetailsResponse`
Used for checking video existence and getting video details.

## Usage

The video upload functionality is integrated into the existing `uploadVideo()` function and can be used via the CLI:

```bash
./gfl uploadvideo --accountId YOUR_ACCOUNT_ID --apiKey YOUR_API_KEY --brand YOUR_BRAND --folder path/to/videos
```

### Required Parameters:
- `--accountId` / `-a`: Your Cloudflare account ID
- `--apiKey` / `-k`: Your Cloudflare API key with Stream permissions
- `--brand` / `-b`: Brand identifier for organizing videos

### Optional Parameters:
- `--folder` / `-f`: Custom folder path (default: `input/images`)
- `--video` / `-i`: Specific video filename to upload

## Supported Video Formats

The implementation supports common video formats:
- `.mp4`
- `.mov`
- `.avi`
- `.wmv`

## File Size Limitations

This implementation is designed for basic uploads (files under 200MB). For larger files, consider implementing:
- Resumable uploads using tus protocol
- Chunked uploads
- Progress tracking

## Error Handling

The implementation includes comprehensive error handling for:
- File access errors
- Network connectivity issues
- Authentication failures
- API rate limiting
- Invalid file formats
- Upload failures

## Integration with Existing Codebase

The video upload functionality integrates seamlessly with your existing:
- Brand naming conventions (`gontentful.GetCloudflareImagesID(brand)`)
- Authentication system (same API key used for images)
- CLI structure and command patterns
- Error handling and logging patterns

## Next Steps

1. Test the implementation with various video file sizes and formats
2. Consider adding progress bars for large uploads
3. Implement batch video processing similar to images
4. Add video transcoding options
5. Integrate with your content management system

## Example Video Upload Flow

1. User runs: `./gfl uploadvideo --accountId abc123 --apiKey xyz789 --brand mygame`
2. System scans for video files in input directory
3. For each video file:
   - Generates video ID using brand prefix
   - Checks if video already exists
   - Deletes existing video if found
   - Uploads new video to Cloudflare Stream
   - Reports success/failure

The implementation follows the same patterns as your existing image upload functionality, ensuring consistency and maintainability.