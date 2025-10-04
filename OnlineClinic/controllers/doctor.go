// controllers/doctor.go
package controllers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"onlineClinic/config"
	"onlineClinic/models"
	"onlineClinic/utils"
	"sort"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

type SearchRequest struct {
	UserSearch string `json:"userSearch"`
}

func SearchDoctors(w http.ResponseWriter, r *http.Request) {
	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// log.Printf("Error decoding search request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate search term
	if req.UserSearch == "" {
		http.Error(w, "Search term cannot be empty", http.StatusBadRequest)
		return
	}

	results, err := models.SearchDoctors(config.DB, req.UserSearch)
	if err != nil {
		// log.Printf("Error searching doctors: %v", err)
		http.Error(w, "Error performing search", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// GetDoctorProfile handles GET requests for doctor profile
func GetDoctorProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid doctor ID", http.StatusBadRequest)
		return
	}

	doctor, err := models.GetDoctorById(config.DB, id)
	if err != nil {
		if err.Error() == "doctor not found" {
			http.Error(w, "Doctor not found", http.StatusNotFound)
			return
		}
		// log.Printf("Error retrieving doctor: %v", err)
		http.Error(w, "Error retrieving doctor profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doctor)
}

// UpdateDoctorProfile handles PUT requests to update doctor profile
func UpdateDoctorProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid doctor ID", http.StatusBadRequest)
		return
	}

	var doctor models.Doctor
	if err := json.NewDecoder(r.Body).Decode(&doctor); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input
	if doctor.FirstName == "" || doctor.LastName == "" || doctor.PhoneNumber == "" ||
		doctor.MedicalCouncilCode == nil || *doctor.MedicalCouncilCode == "" { // Added validation
		http.Error(w, "Required fields cannot be empty", http.StatusBadRequest)
		return
	}

	if !validatePhoneNumber(doctor.PhoneNumber) {
		http.Error(w, "Invalid phone number format", http.StatusBadRequest)
		return
	}

	if doctor.NationalCode != "" && !validateNationalCode(doctor.NationalCode) {
		http.Error(w, "Invalid national code format", http.StatusBadRequest)
		return
	}

	// Set the ID from the URL
	doctor.ID = id

	// Start a transaction
	tx, err := config.DB.Begin()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Fetch the existing profile photo path from the database
	var existingProfilePhotoPath sql.NullString
	err = tx.QueryRow("SELECT profile_photo_path FROM doctors WHERE id = ?", id).Scan(&existingProfilePhotoPath)
	if err != nil {
		http.Error(w, "Error fetching profile photo path", http.StatusInternalServerError)
		return
	}

	// Set the profile photo path to the existing value if valid, else nil
	if existingProfilePhotoPath.Valid {
		doctor.ProfilePhotoPath = &existingProfilePhotoPath.String
	} else {
		doctor.ProfilePhotoPath = nil
	}

	// Update the main profile information
	if err := models.UpdateDoctor(tx, &doctor); err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") {
			http.Error(w, "Phone number already exists", http.StatusConflict)
			return
		}
		http.Error(w, "Error updating doctor profile", http.StatusInternalServerError)
		return
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		http.Error(w, "Error updating profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Profile updated successfully"})
}

// GetAllDoctors handles GET requests to list all doctors
func GetAllDoctors(w http.ResponseWriter, r *http.Request) {
	// Call the GetAllDoctors function from the models package
	doctors, err := models.GetAllDoctors(config.DB)
	if err != nil {
		http.Error(w, "Error retrieving doctors list", http.StatusInternalServerError)
		return
	}

	// Define a struct for the response without the password field
	type doctorWithoutPass struct {
		ID                 int     `json:"id"`
		FirstName          string  `json:"firstName"`
		LastName           string  `json:"lastName"`
		NationalCode       string  `json:"nationalCode"`
		Gender             string  `json:"gender"`
		PhoneNumber        string  `json:"phoneNumber"`
		ProfilePhotoPath   string  `json:"profilePhotoPath,omitempty"`
		Address            string  `json:"address,omitempty"`
		MedicalCouncilCode *string `json:"medicalCouncilCode,omitempty"` // Added field
	}

	// Convert the list of doctors to the response struct
	var docs []doctorWithoutPass
	for _, d := range doctors {
		doc := doctorWithoutPass{
			ID:                 d.ID,
			FirstName:          d.FirstName,
			LastName:           d.LastName,
			NationalCode:       d.NationalCode,
			Gender:             d.Gender,
			PhoneNumber:        d.PhoneNumber,
			MedicalCouncilCode: d.MedicalCouncilCode, // Added field
		}

		// Add ProfilePhotoPath if it's not nil
		if d.ProfilePhotoPath != nil {
			doc.ProfilePhotoPath = *d.ProfilePhotoPath
		}

		// Add Address if it's not nil
		if d.Address != nil {
			doc.Address = *d.Address
		}

		docs = append(docs, doc)
	}

	// Set the response headers and encode the response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(docs)
}

func UpdateDoctorPassword(w http.ResponseWriter, r *http.Request) {
	// log.Println("Starting password update process")

	// Get doctor ID from context
	doctorID, ok := utils.GetUserID(r.Context())
	if !ok {
		// log.Printf("Failed to get doctorID from context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	// log.Printf("Processing password update for doctor ID: %d", doctorID)

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
	if err := models.UpdateDoctorPassword(tx, doctorID, passwordUpdate.UserNewPassword); err != nil {
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

// DeleteDoctorProfile handles DELETE requests to remove doctor profile
func DeleteDoctorProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid doctor ID", http.StatusBadRequest)
		return
	}

	if err := models.DeleteDoctor(config.DB, id); err != nil {
		// log.Printf("Error deleting doctor: %v", err)
		http.Error(w, "Error deleting doctor profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Profile deleted successfully"})
}

// GetDoctorPrescriptions handles GET requests for doctor prescriptions
func GetDoctorPrescriptions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid doctor ID", http.StatusBadRequest)
		return
	}

	prescriptions, err := models.GetDoctorPrescriptions(config.DB, id)
	if err != nil {
		// log.Printf("Error retrieving prescriptions: %v", err)
		http.Error(w, "Error retrieving prescriptions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prescriptions)
}

// GetDoctorAvailability handles GET requests for doctor availability slots
func GetDoctorAvailability(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	doctorID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid doctor ID", http.StatusBadRequest)
		return
	}

	visitType := r.URL.Query().Get("visitType")
	if visitType == "" {
		http.Error(w, "Visit type is required", http.StatusBadRequest)
		return
	}

	if visitType != "online" && visitType != "in-person" {
		http.Error(w, "Invalid visit type. Must be 'online' or 'in-person'", http.StatusBadRequest)
		return
	}

	slots, err := models.GetDoctorAvailability(config.DB, doctorID, visitType)
	if err != nil {
		// log.Printf("Error retrieving availability slots: %v", err)
		http.Error(w, "Error retrieving availability slots", http.StatusInternalServerError)
		return
	}

	// Define response structs to control JSON output
	type ResponseSlot struct {
		ID        int    `json:"id,string"` // Keep as int, but marshal as string in JSON
		DoctorID  int    `json:"doctorId"`
		StartTime string `json:"startTime"`
		EndTime   string `json:"endTime"`
		Time      string `json:"time"`
		Type      string `json:"type"`
	}

	type AvailabilityDay struct {
		Date  string         `json:"date"`
		Times []ResponseSlot `json:"times"`
	}

	// Group slots by date
	availabilityByDate := make(map[string][]ResponseSlot)
	for _, slot := range slots {
		// Convert slot.ID from string to int
		slotID := slot.ID
		// Convert Gregorian date to Solar (Hijri) date format
		solarDate := utils.GregorianToSolar(slot.StartTime)
		availabilityByDate[solarDate] = append(availabilityByDate[solarDate], ResponseSlot{
			ID:        slotID,
			DoctorID:  slot.DoctorID,
			StartTime: slot.StartTime.Format("15:04"), // Format as HH:mm
			EndTime:   slot.EndTime.Format("15:04"),   // Format as HH:mm
			Time:      slot.Time,
			Type:      slot.Type,
		})
	}

	// Convert map to slice for response
	var response []AvailabilityDay
	for date, times := range availabilityByDate {
		response = append(response, AvailabilityDay{
			Date:  date,
			Times: times,
		})
	}

	// Sort the response by date in descending order (latest first)
	sort.Slice(response, func(i, j int) bool {
		res, _ := utils.CompareSolarDates(response[i].Date, response[j].Date)
		return res > 0 // For descending order (latest first)
	})
	// responseJSON, _ := json.Marshal(response)
	// log.Printf("Response JSON: %s", string(responseJSON))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// SetDoctorAvailability handles POST requests to add new availability slots
func SetDoctorAvailability(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid doctor ID", http.StatusBadRequest)
		return
	}

	var availabilityReq models.AvailabilityRequest
	if err := json.NewDecoder(r.Body).Decode(&availabilityReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate time slots
	// for _, timeRange := range availabilityReq.TimesRange {
	// 	if timeRange.Start.After(timeRange.End) || timeRange.Start.Equal(timeRange.End) {
	// 		http.Error(w, "Invalid time slot: end time must be after start time", http.StatusBadRequest)
	// 		return
	// 	}
	// }

	for _, dateRange := range availabilityReq.DatesRange {
		if dateRange.Start.After(dateRange.End) {
			http.Error(w, "Invalid date range: end date must be after start date", http.StatusBadRequest)
			return
		}
	}

	if err := models.SetDoctorAvailability(config.DB, id, &availabilityReq); err != nil {
		// log.Printf("Error setting availability: %v", err)
		http.Error(w, "Error setting availability slots", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Availability slots added successfully"})
}

// DeleteDoctorAvailability handles DELETE requests to remove availability slots
func DeleteDoctorAvailability(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	doctorID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid doctor ID", http.StatusBadRequest)
		return
	}

	slotID, err := strconv.Atoi(vars["slotId"])
	if err != nil {
		http.Error(w, "Invalid slot ID", http.StatusBadRequest)
		return
	}

	// Get authenticated user claims
	claims, ok := r.Context().Value(utils.UserClaimsKey).(*utils.Claims)
	if !ok || !claims.IsDoctor || claims.UserID != doctorID {
		http.Error(w, "Unauthorized: You can only delete your own availability slots", http.StatusForbidden)
		return
	}

	// Delete the availability slot
	if err := models.DeleteDoctorAvailability(config.DB, slotID, doctorID); err != nil {
		if errors.Is(err, models.ErrAvailabilityNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		// log.Printf("Error deleting availability slot: %v", err)
		http.Error(w, "Error deleting availability slot", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Availability slot deleted successfully"})
}

func DeleteDoctorProfilePhoto(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid patient ID", http.StatusBadRequest)
		return
	}

	if err := models.DeleteDoctorPhoto(config.DB, id); err != nil {
		// log.Printf("Error deleting patient profile photo: %v", err)
		http.Error(w, "Error deleting patient profile photo", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Profile photo deleted successfully"})
}
