package imgbb

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"testing"

	"noodu-login-service/internal/customerprofile"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (roundTrip roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTrip(request)
}

func TestUploadSendsBase64AndReturnsImageMetadata(t *testing.T) {
	imageBytes := []byte("image-bytes-for-test")
	client := NewClient("test-api-key")
	client.httpClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodPost || request.URL.Path != "/1/upload" {
			t.Fatalf("request = %s %s, want POST /1/upload", request.Method, request.URL.Path)
		}
		if request.URL.Query().Get("key") != "test-api-key" {
			t.Fatal("request did not include configured API key")
		}
		if err := request.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("parse multipart request: %v", err)
		}
		encoded := request.FormValue("image")
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			t.Fatalf("decode uploaded image: %v", err)
		}
		if string(decoded) != string(imageBytes) {
			t.Fatalf("uploaded image = %q, want %q", decoded, imageBytes)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"success":true,"data":{"id":"img123","url":"https://i.ibb.co/img123/photo.jpg"}}`)),
		}, nil
	})}

	image, err := client.Upload(context.Background(), customerprofile.ImageUpload{
		Filename:    "photo.jpg",
		ContentType: "image/jpeg",
		Data:        imageBytes,
	})
	if err != nil {
		t.Fatal(err)
	}
	if image.ID != "img123" || image.URL != "https://i.ibb.co/img123/photo.jpg" {
		t.Fatalf("uploaded image = %+v, want returned ImgBB metadata", image)
	}
}

func TestUploadRejectsNonHTTPSOrNonImgBBURL(t *testing.T) {
	for _, imageURL := range []string{
		"http://i.ibb.co/image.jpg",
		"https://attacker.example/image.jpg",
	} {
		t.Run(imageURL, func(t *testing.T) {
			client := NewClient("test-api-key")
			client.httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				body := `{"success":true,"data":{"id":"img123","url":"` + imageURL + `"}}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(body)),
				}, nil
			})}
			_, err := client.Upload(context.Background(), customerprofile.ImageUpload{Data: []byte("image")})
			if err == nil {
				t.Fatal("expected invalid ImgBB image URL to be rejected")
			}
		})
	}
}
