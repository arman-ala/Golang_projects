// controllers/debug.go (REMOVE IN PRODUCTION)
package controllers

import (
	"encoding/json"
	"net/http"
	"onlineClinic/config"
)

// VerifyStoredHash - Development only endpoint to verify stored hashes
func VerifyStoredHash(w http.ResponseWriter, r *http.Request) {
	phoneNumber := r.URL.Query().Get("phone")
	if phoneNumber == "" {
		http.Error(w, "Phone number required", http.StatusBadRequest)
		return
	}

	// log.Printf("Debugging hash for phone: %s", phoneNumber)

	// Check patients table
	var patientHash string
	err := config.DB.QueryRow("SELECT password FROM patients WHERE phone_number = ?", phoneNumber).Scan(&patientHash)
	if err == nil {
		// log.Printf("Found patient hash. Length: %d, Hash: %s", len(patientHash), patientHash)
		json.NewEncoder(w).Encode(map[string]string{
			"type": "patient",
			"hash": patientHash,
		})
		return
	}

	// Check doctors table
	var doctorHash string
	err = config.DB.QueryRow("SELECT password FROM doctors WHERE phone_number = ?", phoneNumber).Scan(&doctorHash)
	if err == nil {
		// log.Printf("Found doctor hash. Length: %d, Hash: %s", len(doctorHash), doctorHash)
		json.NewEncoder(w).Encode(map[string]string{
			"type": "doctor",
			"hash": doctorHash,
		})
		return
	}

	// log.Printf("No hash found for phone number: %s", phoneNumber)
	http.Error(w, "No hash found", http.StatusNotFound)
}
