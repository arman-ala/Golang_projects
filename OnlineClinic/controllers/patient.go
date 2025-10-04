// ./controllers/patient.go
package controllers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"onlineClinic/config"
	"onlineClinic/models"
	"onlineClinic/utils"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

func GetPatientProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid patient ID", http.StatusBadRequest)
		return
	}

	patient, err := models.GetPatientById(config.DB, id)
	if err != nil {
		// log.Printf("Error retrieving patient: %v", err)
		http.Error(w, "Error retrieving patient profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(patient)
}

func GetPatientInfo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid patient ID", http.StatusBadRequest)
		return
	}

	patient, err := models.GetPatientById(config.DB, id)
	if err != nil {
		// log.Printf("Error retrieving patient: %v", err)
		http.Error(w, "Error retrieving patient profile", http.StatusInternalServerError)
		return
	}

	// Define the struct as a type
	type PatientInfo struct {
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		Age       *int   `json:"age,omitempty"` // Use pointer to int to handle nil values
		Gender    string `json:"gender"`
	}

	// Use the type to create a variable
	p := PatientInfo{
		FirstName: patient.FirstName,
		LastName:  patient.LastName,
		Age:       patient.Age, // Directly assign the pointer (nil if not set)
		Gender:    patient.Gender,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

func UpdatePatientProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid patient ID", http.StatusBadRequest)
		return
	}

	var patient models.Patient
	if err := json.NewDecoder(r.Body).Decode(&patient); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input
	if patient.FirstName == "" || patient.LastName == "" || patient.PhoneNumber == "" {
		http.Error(w, "Required fields cannot be empty", http.StatusBadRequest)
		return
	}

	if !utils.ValidatePhoneNumber(patient.PhoneNumber) {
		http.Error(w, "Invalid phone number format", http.StatusBadRequest)
		return
	}

	if patient.NationalCode != "" && !utils.ValidateNationalCode(patient.NationalCode) {
		http.Error(w, "Invalid national code format", http.StatusBadRequest)
		return
	}

	patient.ID = id

	// Fetch the existing profile photo path from the database
	var existingProfilePhotoPath sql.NullString
	err = config.DB.QueryRow("SELECT profile_photo_path FROM patients WHERE id = ?", id).Scan(&existingProfilePhotoPath)
	if err != nil {
		// log.Printf("Error fetching existing profile photo path: %v", err)
		http.Error(w, "Error fetching profile photo path", http.StatusInternalServerError)
		return
	}

	// Set the profile photo path to the existing value if valid, else nil
	if existingProfilePhotoPath.Valid {
		patient.ProfilePhotoPath = &existingProfilePhotoPath.String
	} else {
		patient.ProfilePhotoPath = nil
	}

	if err := models.UpdatePatient(config.DB, &patient); err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") {
			http.Error(w, "Phone number already exists", http.StatusConflict)
			return
		}
		// log.Printf("Error updating patient: %v", err)
		http.Error(w, "Error updating patient profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Profile updated successfully"})
}

func UpdatePatientPassword(w http.ResponseWriter, r *http.Request) {
	// log.Println("Starting patient password update process")

	// Get patient ID from context
	patientID, ok := utils.GetUserID(r.Context())
	if !ok {
		// log.Printf("Failed to get patientID from context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	// log.Printf("Processing password update for patient ID: %d", patientID)

	// Parse request body
	var passwordUpdate struct {
		UserNewPassword string `json:"userNewPassword"`
	}

	if err := json.NewDecoder(r.Body).Decode(&passwordUpdate); err != nil {
		// log.Printf("Error decoding request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	// log.Println("Request body decoded successfully")

	// Validate new password
	if passwordUpdate.UserNewPassword == "" {
		// log.Println("Empty password received")
		http.Error(w, "New password cannot be empty", http.StatusBadRequest)
		return
	}
	// log.Println("Password validation passed")

	// Start a transaction
	tx, err := config.DB.Begin()
	if err != nil {
		// log.Printf("Error starting transaction: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()
	// log.Println("Transaction started")

	// Update the password
	if err := models.UpdatePatientPassword(tx, patientID, passwordUpdate.UserNewPassword); err != nil {
		// log.Printf("Error updating password in database: %v", err)
		http.Error(w, "Error updating password", http.StatusInternalServerError)
		return
	}
	// log.Println("Password updated in database")

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		// log.Printf("Error committing transaction: %v", err)
		http.Error(w, "Error updating password", http.StatusInternalServerError)
		return
	}
	// log.Println("Transaction committed successfully")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Password updated successfully"})
	// log.Println("Password update process completed successfully")
}

func DeletePatientProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid patient ID", http.StatusBadRequest)
		return
	}

	if err := models.DeletePatient(config.DB, id); err != nil {
		// log.Printf("Error deleting patient: %v", err)
		http.Error(w, "Error deleting patient profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Profile deleted successfully"})
}

func DeletePatientProfilePhoto(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid patient ID", http.StatusBadRequest)
		return
	}

	if err := models.DeletePatientPhoto(config.DB, id); err != nil {
		// log.Printf("Error deleting patient profile photo: %v", err)
		http.Error(w, "Error deleting patient profile photo", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Profile photo deleted successfully"})
}

func GetAllPatients(w http.ResponseWriter, r *http.Request) {
	patients, err := models.GetAllPatients(config.DB)
	if err != nil {
		// log.Printf("Error retrieving patients: %v", err)
		http.Error(w, "Error retrieving patients list", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(patients)
}
