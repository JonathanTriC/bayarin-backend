package qris

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bayarin/backend/config"
)

// UploadToSupabase uploads image bytes to private bucket
func UploadToSupabase(cfg *config.Config, path string, data []byte, contentType string) error {
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", cfg.SupabaseURL, cfg.SupabaseStorageBucket, path)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.SupabaseServiceRoleKey)
	req.Header.Set("Content-Type", contentType)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("supabase storage error: %s", string(bodyBytes))
	}

	return nil
}

type signedURLRequest struct {
	ExpiresIn int `json:"expiresIn"`
}

type signedURLResponse struct {
	SignedURL string `json:"signedURL"`
}

// GetSignedURL returns a signed URL for a storage path
func GetSignedURL(cfg *config.Config, path string, expiresIn int) (string, error) {
	url := fmt.Sprintf("%s/storage/v1/object/sign/%s/%s", cfg.SupabaseURL, cfg.SupabaseStorageBucket, path)

	reqBody, _ := json.Marshal(signedURLRequest{ExpiresIn: expiresIn})
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.SupabaseServiceRoleKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("supabase signed url error: %s", string(bodyBytes))
	}

	var res signedURLResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	signedUrl := res.SignedURL
	if strings.HasPrefix(signedUrl, "/") {
		signedUrl = cfg.SupabaseURL + signedUrl
	}

	return signedUrl, nil
}
