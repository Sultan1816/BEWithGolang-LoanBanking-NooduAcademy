package imgbb

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"noodu-login-service/internal/customerprofile"
)

const uploadEndpoint = "https://api.imgbb.com/1/upload"

type Client struct {
	apiKey     string
	httpClient *http.Client
}

type uploadResponse struct {
	Success bool `json:"success"`
	Data    struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	} `json:"data"`
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: strings.TrimSpace(apiKey),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (client *Client) Upload(ctx context.Context, image customerprofile.ImageUpload) (customerprofile.UploadedImage, error) {
	if client.apiKey == "" {
		return customerprofile.UploadedImage{}, errors.New("IMGBB_API_KEY is not configured")
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("image", base64.StdEncoding.EncodeToString(image.Data)); err != nil {
		return customerprofile.UploadedImage{}, fmt.Errorf("write ImgBB upload form: %w", err)
	}
	if err := writer.Close(); err != nil {
		return customerprofile.UploadedImage{}, fmt.Errorf("close ImgBB upload form: %w", err)
	}

	endpoint, err := url.Parse(uploadEndpoint)
	if err != nil {
		return customerprofile.UploadedImage{}, fmt.Errorf("parse ImgBB endpoint: %w", err)
	}
	query := endpoint.Query()
	query.Set("key", client.apiKey)
	endpoint.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), &body)
	if err != nil {
		return customerprofile.UploadedImage{}, fmt.Errorf("create ImgBB request: %w", err)
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())

	response, err := client.httpClient.Do(request)
	if err != nil {
		return customerprofile.UploadedImage{}, errors.New("send ImgBB upload request failed")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return customerprofile.UploadedImage{}, fmt.Errorf("ImgBB returned HTTP status %d", response.StatusCode)
	}

	var result uploadResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&result); err != nil {
		return customerprofile.UploadedImage{}, fmt.Errorf("decode ImgBB response: %w", err)
	}
	if !result.Success || result.Data.ID == "" || !validImageURL(result.Data.URL) {
		return customerprofile.UploadedImage{}, errors.New("ImgBB returned an incomplete upload response")
	}
	return customerprofile.UploadedImage{ID: result.Data.ID, URL: result.Data.URL}, nil
}

func validImageURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme == "https" &&
		(parsed.Hostname() == "i.ibb.co" || strings.HasSuffix(parsed.Hostname(), ".ibb.co"))
}
