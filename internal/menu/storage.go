package menu

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/bayarin/backend/config"
)

// UploadMenuImage uploads image bytes to the menu-images public bucket
// Returns the full public URL of the uploaded image
func UploadMenuImage(cfg *config.Config, path string, data []byte, contentType string) (string, error) {
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", cfg.SupabaseURL, cfg.SupabaseMenuBucket, path)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("create req: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+cfg.SupabaseServiceRoleKey)
	req.Header.Set("Content-Type", contentType)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("do req: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("supabase upload error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	publicURL := fmt.Sprintf("%s/storage/v1/object/public/%s/%s", cfg.SupabaseURL, cfg.SupabaseMenuBucket, path)
	return publicURL, nil
}

// DeleteMenuImage deletes an image from the menu-images bucket by storage path
func DeleteMenuImage(cfg *config.Config, path string) error {
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", cfg.SupabaseURL, cfg.SupabaseMenuBucket, path)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("create req: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+cfg.SupabaseServiceRoleKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("do req: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode != http.StatusNotFound {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("supabase delete error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}
