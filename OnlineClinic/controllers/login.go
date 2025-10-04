// controllers/login.go
package controllers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"onlineClinic/config"
	"onlineClinic/utils"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	PhoneNumber string `json:"phoneNumber"`
	Password    string `json:"password"`
}

type LoginResponse struct {
	ID           string `json:"id"`
	FirstName    string `json:"firstName"`
	LastName     string `json:"lastName"`
	NationalCode string `json:"nationalCode"`
	Gender       string `json:"gender"`
	PhoneNumber  string `json:"phoneNumber"`
	IsDoctor     bool   `json:"isDoctor"`
	Age          int    `json:"age"`
	Job          string `json:"job,omitempty"` // omitempty for doctors
	Education    string `json:"education"`
	Address      string `json:"address"`
	Image        string `json:"image"`
	Token        string `json:"token"`
}

// Add this type to store temporary password during login
type userCredentials struct {
	hashedPassword string
}

func LoginPatient(w http.ResponseWriter, r *http.Request) {
	// // log.Printf("Patient login attempt initiated from IP: %s", r.RemoteAddr)

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// log.Printf("Error decoding patient login request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Trim whitespace from the phone number
	req.PhoneNumber = strings.TrimSpace(req.PhoneNumber)
	// log.Printf("Attempting patient login for phone number: '%s'", req.PhoneNumber)

	var patientResponse LoginResponse
	var patientCreds userCredentials
	var patientID int
	var nullAge sql.NullInt64
	var nullJob, nullEducation, nullAddress, nullProfilePhotoPath sql.NullString

	// Query for patient
	// queryStart := time.Now()
	err := config.DB.QueryRow(`
        SELECT id, first_name, last_name, national_code, gender,
               phone_number, password, age, job, education, address, 
               profile_photo_path
        FROM patients 
        WHERE phone_number = ?`, req.PhoneNumber).Scan(
		&patientID,
		&patientResponse.FirstName,
		&patientResponse.LastName,
		&patientResponse.NationalCode,
		&patientResponse.Gender,
		&patientResponse.PhoneNumber,
		&patientCreds.hashedPassword,
		&nullAge,
		&nullJob,
		&nullEducation,
		&nullAddress,
		&nullProfilePhotoPath,
	)

	// log.Printf("Patient DB query took %v", time.Since(queryStart))

	if err != nil {
		if err == sql.ErrNoRows {
			// log.Printf("No patient found with phone number: '%s'", req.PhoneNumber)
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}
		// log.Printf("Database error during patient login: %v", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	// Convert nullable fields
	if nullAge.Valid {
		patientResponse.Age = int(nullAge.Int64)
	}
	if nullJob.Valid {
		patientResponse.Job = nullJob.String
	}
	if nullEducation.Valid {
		patientResponse.Education = nullEducation.String
	}
	if nullAddress.Valid {
		patientResponse.Address = nullAddress.String
	}
	if nullProfilePhotoPath.Valid {
		patientResponse.Image = nullProfilePhotoPath.String
	}

	// Verify password
	// log.Printf("Hashed password from DB: %s", patientCreds.hashedPassword)
	// log.Printf("Input password: %s", req.Password)

	if err := bcrypt.CompareHashAndPassword([]byte(patientCreds.hashedPassword), []byte(req.Password)); err != nil {
		// log.Printf("Password verification failed for patient ID %d: %v", patientID, err)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// log.Printf("Password verified successfully for patient ID: %d", patientID)

	// Generate token
	token, err := utils.GenerateToken(patientID, patientResponse.PhoneNumber, false, true)
	if err != nil {
		// log.Printf("Error generating token for patient ID %d: %v", patientID, err)
		http.Error(w, "Error generating token", http.StatusInternalServerError)
		return
	}

	// log.Printf("USER TOKEN: %s", token)

	patientResponse.ID = strconv.Itoa(patientID)
	patientResponse.IsDoctor = false
	patientResponse.Token = token

	// log.Printf("Login successful for patient ID: %d", patientID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(patientResponse)
}

func LoginDoctor(w http.ResponseWriter, r *http.Request) {
	// log.Printf("Doctor login attempt initiated from IP: %s", r.RemoteAddr)

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// log.Printf("Error decoding doctor login request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Trim whitespace and remove any extra quotes from the phone number
	req.PhoneNumber = strings.TrimSpace(req.PhoneNumber)
	req.PhoneNumber = strings.Trim(req.PhoneNumber, "'\"") // Remove single or double quotes
	// log.Printf("Attempting doctor login for phone number: '%s'", req.PhoneNumber)

	// Validate phone number format
	if len(req.PhoneNumber) != 11 || !strings.HasPrefix(req.PhoneNumber, "09") {
		// log.Printf("Invalid phone number format: '%s'", req.PhoneNumber)
		http.Error(w, "Invalid phone number format", http.StatusBadRequest)
		return
	}

	var doctorResponse LoginResponse
	var doctorCreds userCredentials
	var doctorID int
	var nullAge sql.NullInt64
	var nullEducation, nullAddress, nullProfilePhotoPath sql.NullString

	// Query for doctor
	// queryStart := time.Now()
	err := config.DB.QueryRow(`
        SELECT 
            id, 
            first_name, 
            last_name, 
            national_code, 
            gender,
            phone_number,  
            password, 
            age,
            education, 
            address, 
            profile_photo_path
        FROM doctors 
        WHERE phone_number = ?`, req.PhoneNumber).Scan(
		&doctorID,
		&doctorResponse.FirstName,
		&doctorResponse.LastName,
		&doctorResponse.NationalCode,
		&doctorResponse.Gender,
		&doctorResponse.PhoneNumber,
		&doctorCreds.hashedPassword,
		&nullAge,
		&nullEducation,
		&nullAddress,
		&nullProfilePhotoPath,
	)

	// log.Printf("Doctor DB query took %v", time.Since(queryStart))

	if err != nil {
		if err == sql.ErrNoRows {
			// log.Printf("No doctor found with phone number: '%s'", req.PhoneNumber)
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}
		// log.Printf("Database error during doctor login: %v", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	// Convert nullable fields
	if nullAge.Valid {
		doctorResponse.Age = int(nullAge.Int64)
	}
	if nullEducation.Valid {
		doctorResponse.Education = nullEducation.String
	}
	if nullAddress.Valid {
		doctorResponse.Address = nullAddress.String
	}
	if nullProfilePhotoPath.Valid {
		doctorResponse.Image = nullProfilePhotoPath.String
	}

	// log.Printf("Doctor found with ID: %d, verifying password", doctorID)

	// Verify password
	// log.Printf("Hashed password from DB: %s", doctorCreds.hashedPassword)
	// log.Printf("Input password: %s", req.Password)

	if err := bcrypt.CompareHashAndPassword([]byte(doctorCreds.hashedPassword), []byte(req.Password)); err != nil {
		// log.Printf("Password verification failed for doctor ID %d: %v", doctorID, err)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// log.Printf("Password verified successfully for doctor ID: %d", doctorID)

	// Generate token
	token, err := utils.GenerateToken(doctorID, doctorResponse.PhoneNumber, true, false)
	if err != nil {
		// log.Printf("Error generating token for doctor ID %d: %v", doctorID, err)
		http.Error(w, "Error generating token", http.StatusInternalServerError)
		return
	}

	doctorResponse.ID = strconv.Itoa(doctorID)
	doctorResponse.IsDoctor = true
	doctorResponse.Token = token

	// log.Printf("Login successful for doctor ID: %d", doctorID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doctorResponse)
}

// Helper function to log SQL queries (optional, for debugging)
func logQuery(query string, args ...interface{}) {
	// log.Printf("Executing SQL Query: %s with args: %v", query, args)
}
