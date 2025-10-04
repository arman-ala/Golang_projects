package utils

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	MaxFileSize = 10 << 20 // 10 MB
	UploadDir   = "./uploads"
)

func init() {
	dirs := []string{
		filepath.Join(UploadDir, "profile", "doctors"),
		filepath.Join(UploadDir, "profile", "patients"),
		filepath.Join(UploadDir, "chat", "doctors"),
		filepath.Join(UploadDir, "chat", "patients"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			// log.Printf("Error creating directory %s: %v", dir, err)
		}
	}
}

func SaveFile(file multipart.File, header *multipart.FileHeader, userID int, fileType string, isDoctor bool) (string, error) {
	// log.Printf("Processing file upload: %s, type: %s, size: %d, isDoctor: %v", header.Filename, fileType, header.Size, isDoctor)

	if header.Size > MaxFileSize {
		return "", fmt.Errorf("file size exceeds maximum limit of %d bytes", MaxFileSize)
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if fileType == "profile" {
		allowedTypes := map[string]bool{
			".jpg":  true,
			".jpeg": true,
			".png":  true,
		}
		if !allowedTypes[ext] {
			return "", fmt.Errorf("unsupported file type for profile photo: %s. Allowed types: jpg, jpeg, png", ext)
		}
	} else {
		allowedTypes := map[string]bool{
			".jpg":  true,
			".jpeg": true,
			".png":  true,
			".pdf":  true,
			".doc":  true,
			".docx": true,
			".mp3":  true,
			".wav":  true,
			".mp4":  true,
			".avi":  true,
			".zip":  true,
		}
		if !allowedTypes[ext] {
			return "", fmt.Errorf("unsupported file type: %s", ext)
		}
	}

	userType := "patients"
	if isDoctor {
		userType = "doctors"
	}

	uploadPath := filepath.Join(UploadDir, fileType, userType, fmt.Sprintf("%d", userID))
	if err := os.MkdirAll(uploadPath, 0755); err != nil {
		// log.Printf("Error creating directory %s: %v", uploadPath, err)
		return "", fmt.Errorf("failed to create upload directory: %v", err)
	}

	if fileType == "profile" {
		files, err := os.ReadDir(uploadPath)
		if err == nil {
			for _, f := range files {
				if err := os.Remove(filepath.Join(uploadPath, f.Name())); err != nil {
					// log.Printf("Error removing old profile photo: %v", err)
				}
			}
		}
	}

	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	filePath := filepath.Join(uploadPath, filename)

	dst, err := os.Create(filePath)
	if err != nil {
		// log.Printf("Error creating file %s: %v", filePath, err)
		return "", fmt.Errorf("failed to create file: %v", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		// log.Printf("Error copying file: %v", err)
		return "", fmt.Errorf("failed to save file: %v", err)
	}

	relativePath := filepath.Join(fileType, userType, fmt.Sprintf("%d", userID), filename)
	relativePath = filepath.ToSlash(relativePath)

	// log.Printf("File saved successfully. Relative path: %s", relativePath)
	return relativePath, nil
}

func GetFilePath(relativePath string) string {
	return filepath.Join(UploadDir, relativePath)
}

func DeleteFile(relativePath string) error {
	fullPath := GetFilePath(relativePath)
	if err := os.Remove(fullPath); err != nil {
		// log.Printf("Error deleting file %s: %v", fullPath, err)
		return fmt.Errorf("failed to delete file: %v", err)
	}
	return nil
}

func SaveChatFile(file multipart.File, header *multipart.FileHeader, userID int, isDoctor bool) (string, error) {
	// log.Printf("Processing chat file upload: %s, size: %d, isDoctor: %v", header.Filename, header.Size, isDoctor)

	if header.Size > MaxFileSize {
		return "", fmt.Errorf("file size exceeds maximum limit of %d bytes", MaxFileSize)
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowedTypes := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".pdf":  true,
		".doc":  true,
		".docx": true,
		".mp3":  true,
		".wav":  true,
		".mp4":  true,
		".avi":  true,
		".zip":  true,
	}
	if !allowedTypes[ext] {
		return "", fmt.Errorf("unsupported file type: %s. Allowed types: jpg, jpeg, png, pdf, doc, docx, mp3, wav, mp4, avi", ext)
	}

	userType := "patients"
	if isDoctor {
		userType = "doctors"
	}

	uploadPath := filepath.Join(UploadDir, "chat", userType, fmt.Sprintf("%d", userID))
	if err := os.MkdirAll(uploadPath, 0755); err != nil {
		// log.Printf("Error creating directory %s: %v", uploadPath, err)
		return "", fmt.Errorf("failed to create upload directory: %v", err)
	}

	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	filePath := filepath.Join(uploadPath, filename)

	dst, err := os.Create(filePath)
	if err != nil {
		// log.Printf("Error creating file %s: %v", filePath, err)
		return "", fmt.Errorf("failed to create file: %v", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		// log.Printf("Error copying file: %v", err)
		return "", fmt.Errorf("failed to save file: %v", err)
	}

	relativePath := filepath.Join("chat", userType, fmt.Sprintf("%d", userID), filename)
	relativePath = filepath.ToSlash(relativePath)
	// log.Printf("Chat file saved successfully. Relative path: %s", relativePath)
	return relativePath, nil
}
