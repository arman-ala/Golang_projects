package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"onlineClinic/config"
	"onlineClinic/utils"
)

const (
	MaxUploadSize = 10 << 20 // 10 MB
)

func setCORSHeaders(w http.ResponseWriter, origin string) {
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Origin, Accept")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
}

func UploadProfilePhoto(w http.ResponseWriter, r *http.Request) {
	// log.Println("Starting profile photo upload...")

	// Set CORS headers for all responses
	setCORSHeaders(w, "https://kashan-clininc.liara.run")

	claims, ok := utils.GetUserClaims(r.Context())
	if !ok {
		// log.Println("No claims found in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseMultipartForm(MaxUploadSize); err != nil {
		// log.Printf("Error parsing multipart form: %v", err)
		http.Error(w, "File too large or invalid form data", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("photo")
	if err != nil {
		// log.Printf("Error getting photo from form: %v", err)
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	filePath, err := utils.SaveFile(file, header, claims.UserID, "profile", claims.IsDoctor)
	if err != nil {
		// log.Printf("Error saving file: %v", err)
		http.Error(w, fmt.Sprintf("Error saving file: %v", err), http.StatusBadRequest)
		return
	}

	var updateQuery string
	if claims.IsDoctor {
		updateQuery = "UPDATE doctors SET profile_photo_path = ? WHERE id = ?"
	} else {
		updateQuery = "UPDATE patients SET profile_photo_path = ? WHERE id = ?"
	}

	_, err = config.DB.Exec(updateQuery, filePath, claims.UserID)
	if err != nil {
		// log.Printf("Error updating database: %v", err)
		http.Error(w, "Error updating profile photo", http.StatusInternalServerError)
		return
	}

	// rowsAffected, _ := result.RowsAffected()
	// log.Printf("Updated profile photo for user %d, rows affected: %d", claims.UserID, rowsAffected)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Profile photo uploaded successfully",
		"path":    filePath,
	}); err != nil {
		// log.Printf("Error encoding response: %v", err)
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}
}

func UploadChatFile(w http.ResponseWriter, r *http.Request) {
	// log.Println("Starting chat file upload...")

	// Set CORS headers for all responses
	// setCORSHeaders(w, "https://kashan-clininc.liara.run")

	claims, ok := utils.GetUserClaims(r.Context())
	if !ok {
		// log.Println("No claims found in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseMultipartForm(MaxUploadSize); err != nil {
		// log.Printf("Error parsing multipart form: %v", err)
		http.Error(w, "File too large or invalid form data", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		// log.Printf("Error getting file from form: %v", err)
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	filePath, err := utils.SaveChatFile(file, header, claims.UserID, claims.IsDoctor)
	if err != nil {
		// log.Printf("Error saving file: %v", err)
		http.Error(w, fmt.Sprintf("Error saving file: %v", err), http.StatusBadRequest)
		return
	}

	baseURL := "https://online-clinic.liara.run/uploads"
	fullURL := fmt.Sprintf("%s/%s", baseURL, filePath)

	response := map[string]string{
		"url":  fullURL,
		"name": header.Filename,
		"type": header.Header.Get("Content-Type"),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// log.Printf("Error encoding response: %v", err)
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}
}
