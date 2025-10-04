// controllers/register.go
package controllers

import (
	"encoding/json"
	"net/http"
	"onlineClinic/config"
	"onlineClinic/models"
	"onlineClinic/utils"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func RegisterPatient(w http.ResponseWriter, r *http.Request) {
	var patient models.Patient
	if err := json.NewDecoder(r.Body).Decode(&patient); err != nil {
		// log.Printf("Error decoding request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if patient.FirstName == "" || patient.LastName == "" ||
		patient.NationalCode == "" || patient.PhoneNumber == "" ||
		patient.Password == "" || patient.Gender == "" {
		http.Error(w, "Required fields missing", http.StatusBadRequest)
		return
	}

	// Validate phone number format
	if !utils.ValidatePhoneNumber(patient.PhoneNumber) {
		http.Error(w, "Invalid phone number format", http.StatusBadRequest)
		return
	}

	if len(patient.PhoneNumber) != 11 {
		http.Error(w, "Phone number must be exactly 11 digits", http.StatusBadRequest)
		return
	}

	// Validate national code format
	if !utils.ValidateNationalCode(patient.NationalCode) {
		http.Error(w, "Invalid national code format", http.StatusBadRequest)
		return
	}

	// Validate gender
	if patient.Gender != "man" && patient.Gender != "woman" {
		http.Error(w, "Gender must be either 'man' or 'woman'", http.StatusBadRequest)
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(patient.Password), bcrypt.DefaultCost)
	if err != nil {
		// log.Printf("Error hashing password: %v", err)
		http.Error(w, "Error processing registration", http.StatusInternalServerError)
		return
	}
	patient.Password = string(hashedPassword)

	if err := models.CreatePatient(config.DB, &patient); err != nil {
		// log.Printf("Error creating patient: %v", err)
		if strings.Contains(err.Error(), "Duplicate entry") {
			http.Error(w, "Phone number or national code already registered", http.StatusConflict)
			return
		}
		http.Error(w, "Error registering patient", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func RegisterDoctor(w http.ResponseWriter, r *http.Request) {
	var doctor models.Doctor
	if err := json.NewDecoder(r.Body).Decode(&doctor); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if doctor.FirstName == "" || doctor.LastName == "" ||
		doctor.NationalCode == "" || doctor.PhoneNumber == "" ||
		doctor.Password == "" || doctor.Gender == "" ||
		doctor.MedicalCouncilCode == nil || *doctor.MedicalCouncilCode == "" { // Added validation
		http.Error(w, "Required fields missing", http.StatusBadRequest)
		return
	}

	// Validate phone number format
	if !utils.ValidatePhoneNumber(doctor.PhoneNumber) {
		http.Error(w, "Invalid phone number format", http.StatusBadRequest)
		return
	}

	if len(doctor.PhoneNumber) != 11 {
		http.Error(w, "Phone number must be exactly 11 digits", http.StatusBadRequest)
		return
	}

	// Validate national code format
	if !utils.ValidateNationalCode(doctor.NationalCode) {
		http.Error(w, "Invalid national code format", http.StatusBadRequest)
		return
	}

	// Validate gender
	if doctor.Gender != "man" && doctor.Gender != "woman" {
		http.Error(w, "Gender must be either 'man' or 'woman'", http.StatusBadRequest)
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(doctor.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error processing registration", http.StatusInternalServerError)
		return
	}
	doctor.Password = string(hashedPassword)

	// Create doctor using stored procedure
	if err := models.CreateDoctor(config.DB, &doctor); err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") {
			http.Error(w, "Phone number or national code already registered", http.StatusConflict)
			return
		}
		http.Error(w, "Error registering doctor", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
