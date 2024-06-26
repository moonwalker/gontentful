package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/moonwalker/gontentful"
)

const (
	apiurl      = "https://api.cloudflare.com/client/v4/accounts/%s/images/v1" //cloudflare api url
	inputUrl    = "http://github.com/moonwalker/%s/tree/main/_images/%s"       // add the brand name to path
	inputFolder = "input/images"
	VideoKeyFmt = "%s/%s"
)

type UploadImageParams struct {
	File     *strings.Reader
	URL      string
	Name     string
	Path     string
	Metadata map[string]string
}

type errorResponse struct {
	Errors []struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	Messages []interface{} `json:"messages"`
	Result   interface{}   `json:"result"`
	Success  bool          `json:"success"`
}

type imageDetailsResponse struct {
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

func upsertImage() error {
	start := time.Now()
	c := 0
	v := 0
	d := 0
	f := 0
	dir := inputFolder
	if folder != "" {
		dir = folder
	}
	dc, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("error reading input directory: %w", err)
	}
	c = len(dc)

	repo := fmt.Sprintf("cms-%s", brand)
	if brand == "games" {
		repo = fmt.Sprintf("mw-%s", brand)
	}

	log.Printf("uploading %d images to cloudflare...\n", c)
	for i, e := range dc {
		if e.Name() != "" && e.Name() != ".DS_Store" {

			iName := gontentful.GetCloudflareImagesID(brand) + "/" + e.Name()
			faultyName := gontentful.GetCloudflareImagesID(brand) + e.Name()
			fmt.Printf("uploading images: %d/%d - %s", i, c, e.Name())
			// Faulty image name deletion (can be removed one the path problem is resolved)
			fExists, err := imageExists(faultyName)
			if err != nil {
				return fmt.Errorf("failed to get image details: %w", err)
			}
			if fExists {
				f++
				delImage(faultyName)
			}
			// Legit image name deletion
			exists, err := imageExists(iName)
			if err != nil {
				return fmt.Errorf("failed to get image details: %w", err)
			}
			if exists {
				d++
				delImage(iName)
			}

			if method == "f" {
				log.Print("We're not uploading by file system now.")
			} else {
				iUrl, err := gontentful.GetBlobURL(repo, "_images", e.Name())
				if gontentful.IsVideoFile(e.Name()) {
					v++
					// uploadURL := fmt.Sprintf(VideoKeyFmt, iName, filename)
					// _, err = r2.UploadURL(uploadURL, iUrl)
				} else {
					url := fmt.Sprintf(apiurl, accountId)

					form := map[string]string{"id": iName}
					if err == nil {
						p := UploadImageParams{
							Name:     iName,
							Metadata: form,
							URL:      iUrl,
						}

						ct, payload, err := createForm(p)
						if err != nil {
							fmt.Println(fmt.Sprintf("failed to read file: %s", err.Error()))
							continue
						}

						var resp interface{}
						err = req(http.MethodPost, url, payload, resp, ct)
						if err != nil {
							fmt.Println(fmt.Sprintf("failed to upload image: %s - %s", p.URL, err.Error()))
						}
					} else {
						fmt.Println(fmt.Sprintf("failed to get blob url(brand:%s): %s - %s", brand, e.Name(), err.Error()))

					}
				}
			}

			fmt.Printf("\033[2K")
			fmt.Println()
			fmt.Printf("\033[1A")
		}
	}
	fmt.Printf("Video files ignored: %d", v)
	fmt.Println()
	fmt.Printf("Image files deleted: %d", d)
	fmt.Println()
	fmt.Printf("Faulty image files deleted: %d", f)
	fmt.Println()
	fmt.Printf("%d images successfully uploaded in %.1fs\n", c, time.Since(start).Seconds())
	return nil
}

func imageExists(imageID string) (bool, error) {
	url := fmt.Sprintf(apiurl, accountId) + "/" + imageID
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Add("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("http request failed: %w", err)
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read http response body: %w", err)
	}
	resp := &imageDetailsResponse{}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return false, fmt.Errorf("failed to unmarshal error response: %w", err)
	}
	return resp.Success, nil
}

func delImage(id string) error {
	url := fmt.Sprintf(apiurl, accountId) + "/" + id
	err := req(http.MethodDelete, url, nil, nil, "")
	if err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}
	return nil
}

func req(method, url string, payload io.Reader, resp interface{}, contentType string) error {
	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", apiKey))
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

func createForm(p UploadImageParams) (string, io.Reader, error) {
	body := new(bytes.Buffer)
	mp := multipart.NewWriter(body)
	defer mp.Close()
	for key, val := range p.Metadata {
		mp.WriteField(key, val)
	}

	if len(p.Path) > 0 {
		file, err := os.Open(p.Path)
		if err != nil {
			return "", nil, err
		}
		defer file.Close()
		part, err := mp.CreateFormFile("file", p.Path)
		if err != nil {
			return "", nil, err
		}
		io.Copy(part, file)
	}
	if p.File != nil {
		part, err := mp.CreateFormFile("file", p.Name)
		if err != nil {
			return "", nil, err
		}
		io.Copy(part, p.File)
	}
	if len(p.URL) > 0 {
		mp.WriteField("url", p.URL)
	}
	return mp.FormDataContentType(), body, nil
}
