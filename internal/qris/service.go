package qris

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"path/filepath"

	"github.com/bayarin/backend/config"
	"github.com/bayarin/backend/internal/db/sqlcgen"
	"github.com/google/uuid"
	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	qrcodegen "github.com/skip2/go-qrcode"
)

type Service interface {
	UploadQRIS(ctx context.Context, branchID uuid.UUID, uploadedBy uuid.UUID, file multipart.File, header *multipart.FileHeader) (*sqlcgen.BranchQri, error)
	GenerateQRIS(ctx context.Context, branchID uuid.UUID, amount int64) (string, string, int64, error)
	GetHistory(ctx context.Context, branchID uuid.UUID) ([]sqlcgen.BranchQri, error)
}

type service struct {
	db  *sql.DB
	cfg *config.Config
}

func NewService(db *sql.DB, cfg *config.Config) Service {
	return &service{db: db, cfg: cfg}
}

func (s *service) UploadQRIS(ctx context.Context, branchID uuid.UUID, uploadedBy uuid.UUID, file multipart.File, header *multipart.FileHeader) (*sqlcgen.BranchQri, error) {
	// 1. Decode QR code from image bytes -> extract raw QRIS string
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read image file: %v", err)
	}

	img, _, err := image.Decode(bytes.NewReader(fileBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image format: %v", err)
	}

	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return nil, fmt.Errorf("failed to create binary bitmap from image: %v", err)
	}

	qrReader := qrcode.NewQRCodeReader()
	result, err := qrReader.Decode(bmp, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decode QR code from image: %v", err)
	}
	qrisString := result.GetText()

	// 2. Validate QRIS string
	if err := validateQRIS(qrisString); err != nil {
		return nil, fmt.Errorf("invalid QRIS string: %v", err)
	}

	// 3. Upload image to Supabase Storage
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".png"
	}
	storagePath := fmt.Sprintf("qris/%s/%s%s", branchID.String(), uuid.New().String(), ext)

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	if err := UploadToSupabase(s.cfg, storagePath, fileBytes, contentType); err != nil {
		return nil, fmt.Errorf("failed to upload image: %v", err)
	}

	// 4. Update DB in a transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	qtx := sqlcgen.New(tx)

	// Deactivate existing
	if err := qtx.DeactivateBranchQRIS(ctx, branchID); err != nil {
		return nil, err
	}

	// Insert new
	inserted, err := qtx.InsertBranchQRIS(ctx, sqlcgen.InsertBranchQRISParams{
		ID:         uuid.New(),
		BranchID:   branchID,
		QrisString: qrisString,
		ImagePath:  storagePath,
		UploadedBy: uuid.NullUUID{UUID: uploadedBy, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &inserted, nil
}

func (s *service) GenerateQRIS(ctx context.Context, branchID uuid.UUID, amount int64) (string, string, int64, error) {
	queries := sqlcgen.New(s.db)

	activeQRIS, err := queries.GetActiveBranchQRIS(ctx, branchID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", 0, fmt.Errorf("no active QRIS found for this branch")
		}
		return "", "", 0, err
	}

	dynQRIS, err := makeDynamic(activeQRIS.QrisString, amount)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to generate dynamic QRIS: %v", err)
	}

	pngBytes, err := qrcodegen.Encode(dynQRIS, qrcodegen.Medium, 256)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to encode QR image: %v", err)
	}

	base64Image := base64.StdEncoding.EncodeToString(pngBytes)

	return dynQRIS, base64Image, amount, nil
}

func (s *service) GetHistory(ctx context.Context, branchID uuid.UUID) ([]sqlcgen.BranchQri, error) {
	queries := sqlcgen.New(s.db)

	history, err := queries.ListBranchQRISHistory(ctx, branchID)
	if err != nil {
		return nil, err
	}

	// Replace image paths with signed URLs
	for i := range history {
		signedURL, err := GetSignedURL(s.cfg, history[i].ImagePath, 3600) // 1 hour expiry
		if err == nil {
			history[i].ImagePath = signedURL
		}
	}

	return history, nil
}
