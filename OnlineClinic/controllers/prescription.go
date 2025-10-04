// controllers/prescription.go
package controllers

import (
	"encoding/json"
	"net/http"
	"onlineClinic/config"
	"onlineClinic/models"
	"onlineClinic/utils"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// func CreatePrescription(w http.ResponseWriter, r *http.Request) {
// 	// log.Printf("Attempting to create prescription")

// 	// Retrieve claims
// 	claims, ok := r.Context().Value(utils.UserClaimsKey).(*utils.Claims)
// 	if !ok {
// 		// log.Printf("Claims retrieval failed")
// 		http.Error(w, "Unauthorized", http.StatusUnauthorized)
// 		return
// 	}

// 	// Ensure the user is a doctor
// 	if !claims.IsDoctor {
// 		// log.Printf("User is not a doctor - UserID: %d", claims.UserID)
// 		http.Error(w, "Unauthorized: Only doctors can create prescriptions", http.StatusUnauthorized)
// 		return
// 	}

// 	// Decode the request body
// 	var req models.PrescriptionRequest
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		// log.Printf("Error decoding request: %v", err)
// 		http.Error(w, "Invalid request payload", http.StatusBadRequest)
// 		return
// 	}

// 	// Validate the request
// 	if req.AppointmentID == 0 || len(req.Instructions) == 0 || len(req.Medications) == 0 {
// 		// log.Printf("Invalid request: missing required fields")
// 		http.Error(w, "Appointment ID, instructions, and medications are required", http.StatusBadRequest)
// 		return
// 	}

// 	// Verify the appointment exists and belongs to the doctor
// 	appointment, err := models.GetAppointmentById(config.DB, req.AppointmentID)
// 	if err != nil {
// 		// log.Printf("Error getting appointment: %v", err)
// 		http.Error(w, "Invalid appointment ID", http.StatusBadRequest)
// 		return
// 	}

// 	if appointment.DoctorID != claims.UserID {
// 		// log.Printf("Unauthorized: Doctor %d does not own appointment %d", claims.UserID, req.AppointmentID)
// 		http.Error(w, "Unauthorized to create prescription for this appointment", http.StatusForbidden)
// 		return
// 	}

// 	// Check if a prescription already exists for this appointment
// 	existingPrescription, err := models.GetPrescriptionByAppointment(config.DB, req.AppointmentID, claims.UserID, claims.IsDoctor)
// 	if err == nil && existingPrescription != nil {
// 		// log.Printf("Prescription already exists for appointment ID: %d", req.AppointmentID)
// 		http.Error(w, "Only one prescription is allowed per appointment", http.StatusConflict)
// 		return
// 	}

// 	// Create the prescription
// 	prescription := &models.Prescription{
// 		AppointmentID: req.AppointmentID,
// 		Instructions:  req.Instructions,
// 		Medications:   req.Medications,
// 	}

// 	if err := models.CreatePrescriptionDB(config.DB, prescription); err != nil {
// 		// log.Printf("Error creating prescription: %v", err)
// 		http.Error(w, "Error creating prescription", http.StatusInternalServerError)
// 		return
// 	}

// 	// Prepare the response
// 	response := models.PrescriptionResponse{
// 		ID:            prescription.ID,
// 		AppointmentID: prescription.AppointmentID,
// 		DoctorID:      appointment.DoctorID,
// 		PatientID:     appointment.PatientID,
// 		Instructions:  prescription.Instructions,
// 		Medications:   prescription.Medications, // Corrected field name
// 		CreatedAt:     prescription.CreatedAt,
// 		VisitType:     appointment.VisitType,
// 	}

// 	w.WriteHeader(http.StatusCreated)
// 	json.NewEncoder(w).Encode(response)
// }

func GetPrescriptionsByDoctor(w http.ResponseWriter, r *http.Request) {
	// log.Printf("Fetching prescriptions for doctor")

	// Retrieve claims
	claims, ok := r.Context().Value(utils.UserClaimsKey).(*utils.Claims)
	if !ok || !claims.IsDoctor {
		// log.Printf("Unauthorized: User is not a doctor")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract doctor ID from URL
	vars := mux.Vars(r)
	doctorID, err := strconv.Atoi(vars["id"])
	if err != nil {
		// log.Printf("Invalid doctor ID: %v", err)
		http.Error(w, "Invalid doctor ID", http.StatusBadRequest)
		return
	}

	// Ensure the doctor is accessing their own prescriptions
	if claims.UserID != doctorID {
		// log.Printf("Unauthorized: Doctor %d cannot access prescriptions for doctor %d", claims.UserID, doctorID)
		http.Error(w, "Unauthorized to view these prescriptions", http.StatusForbidden)
		return
	}

	// Fetch prescriptions
	prescriptions, err := models.GetPrescriptionsByDoctor(config.DB, doctorID)
	if err != nil {
		// log.Printf("Error retrieving prescriptions: %v", err)
		http.Error(w, "Error retrieving prescriptions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prescriptions)
}

func GetPrescriptionsByPatient(w http.ResponseWriter, r *http.Request) {
	// log.Printf("Fetching prescriptions for patient")

	// Retrieve claims
	claims, ok := r.Context().Value(utils.UserClaimsKey).(*utils.Claims)
	if !ok {
		// log.Printf("Claims retrieval failed")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract patient ID from URL
	vars := mux.Vars(r)
	patientID, err := strconv.Atoi(vars["id"])
	if err != nil {
		// log.Printf("Invalid patient ID: %v", err)
		http.Error(w, "Invalid patient ID", http.StatusBadRequest)
		return
	}

	// Ensure the user is authorized to access these prescriptions
	if !claims.IsDoctor && claims.UserID != patientID {
		// log.Printf("Unauthorized: User %d cannot access prescriptions for patient %d", claims.UserID, patientID)
		http.Error(w, "Unauthorized to view these prescriptions", http.StatusUnauthorized)
		return
	}

	// Fetch prescriptions
	prescriptions, err := models.GetPatientPrescriptions(
		config.DB,
		patientID,
		claims.UserID,
		claims.IsDoctor,
	)
	if err != nil {
		if err.Error() == "unauthorized access" {
			// log.Printf("Unauthorized access to prescriptions for patient %d", patientID)
			http.Error(w, "Unauthorized access", http.StatusForbidden)
			return
		}
		// log.Printf("Error retrieving prescriptions: %v", err)
		http.Error(w, "Error retrieving prescriptions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prescriptions)
}

func UpdatePrescription(w http.ResponseWriter, r *http.Request) {
	// log.Printf("Attempting to update prescription")

	// Retrieve claims
	claims, ok := r.Context().Value(utils.UserClaimsKey).(*utils.Claims)
	if !ok || !claims.IsDoctor {
		// log.Printf("Unauthorized: User is not a doctor")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Decode the request body
	var req models.PrescriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// log.Printf("Error decoding request: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Validate the request - removed instructions length check
	if req.AppointmentID == 0 || len(req.Medications) == 0 {
		// log.Printf("Invalid request: missing required fields")
		http.Error(w, "Appointment ID and medications are required", http.StatusBadRequest)
		return
	}

	// Fetch the existing prescription
	existingPrescription, err := models.GetPrescriptionByAppointment(
		config.DB,
		req.AppointmentID,
		claims.UserID,
		claims.IsDoctor,
	)
	if err != nil {
		if err.Error() == "unauthorized access" {
			// log.Printf("Unauthorized access to update prescription for appointment %d", req.AppointmentID)
			http.Error(w, "Unauthorized access", http.StatusForbidden)
			return
		}
		// log.Printf("Error retrieving prescription: %v", err)
		http.Error(w, "Error retrieving prescription", http.StatusInternalServerError)
		return
	}

	// Ensure the doctor owns this prescription
	if existingPrescription.DoctorID != claims.UserID {
		// log.Printf("Unauthorized: Doctor %d does not own prescription for appointment %d", claims.UserID, req.AppointmentID)
		http.Error(w, "Unauthorized to modify this prescription", http.StatusForbidden)
		return
	}

	// Update the prescription
	updatedPrescription := &models.Prescription{
		ID:            existingPrescription.ID,
		AppointmentID: req.AppointmentID,
		Instructions:  req.Instructions,
		Medications:   req.Medications,
	}

	if err := models.UpdatePrescriptionDB(config.DB, updatedPrescription); err != nil {
		// log.Printf("Error updating prescription: %v", err)
		http.Error(w, "Error updating prescription", http.StatusInternalServerError)
		return
	}

	// log.Printf("Prescription updated successfully")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Prescription updated successfully"})
}

func GetPatientPrescriptions(w http.ResponseWriter, r *http.Request) {
	// log.Printf("Fetching prescriptions for patient")

	claims, ok := r.Context().Value(utils.UserClaimsKey).(*utils.Claims)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	patientID, err := strconv.Atoi(vars["id"])
	if err != nil {
		// log.Printf("Invalid patient ID: %v", err)
		http.Error(w, "Invalid patient ID", http.StatusBadRequest)
		return
	}

	// If not a doctor, can only view own prescriptions
	if !claims.IsDoctor && claims.UserID != patientID {
		http.Error(w, "Unauthorized to view these prescriptions", http.StatusUnauthorized)
		return
	}

	prescriptions, err := models.GetPatientPrescriptions(
		config.DB,
		patientID,
		claims.UserID,
		claims.IsDoctor,
	)
	if err != nil {
		if err.Error() == "unauthorized access" {
			http.Error(w, "Unauthorized access", http.StatusForbidden)
			return
		}
		// log.Printf("Error retrieving prescriptions: %v", err)
		http.Error(w, "Error retrieving prescriptions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prescriptions)
}

// GetAllPrescriptions handles retrieval of all prescriptions (admin only)
func GetAllPrescriptions(w http.ResponseWriter, r *http.Request) {
	// log.Printf("Fetching all prescriptions")

	claims, ok := r.Context().Value("claims").(*utils.Claims)
	if !ok || !claims.IsDoctor {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := config.DB.Query(`
        SELECT 
            p.id,
            p.appointment_id,
            p.doctor_id,
            p.patient_id,
            p.instructions,
            p.created_at,
            a.visit_type
        FROM prescriptions p
        JOIN appointments a ON p.appointment_id = a.id
        ORDER BY p.created_at DESC
    `)

	if err != nil {
		// log.Printf("Error querying prescriptions: %v", err)
		http.Error(w, "Error fetching prescriptions", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var prescriptions []models.PrescriptionResponse
	for rows.Next() {
		var p models.PrescriptionResponse
		var createdAt time.Time

		err := rows.Scan(
			&p.ID,
			&p.AppointmentID,
			&p.DoctorID,
			&p.PatientID,
			&p.Instructions,
			&createdAt,
			&p.VisitType,
		)
		if err != nil {
			// log.Printf("Error scanning prescription: %v", err)
			http.Error(w, "Error scanning prescriptions", http.StatusInternalServerError)
			return
		}

		// Convert Gregorian date to Hijri date
		p.CreatedAt = utils.GregorianToSolar(createdAt)

		// Get medications for this prescription
		medRows, err := config.DB.Query(`
            SELECT medicine, frequency 
            FROM medications 
            WHERE prescription_id = ?`, p.ID)
		if err != nil {
			// log.Printf("Error fetching medications: %v", err)
			http.Error(w, "Error reading medications", http.StatusInternalServerError)
			return
		}
		defer medRows.Close()

		var medications []models.Medication
		for medRows.Next() {
			var med models.Medication
			if err := medRows.Scan(&med.Medicine, &med.Frequency); err != nil {
				// log.Printf("Error scanning medication: %v", err)
				http.Error(w, "Error reading medication", http.StatusInternalServerError)
				return
			}
			medications = append(medications, med)
		}
		p.Medications = medications

		if p.VisitType == "online" {
			p.VisitType = "آنلاین"
		}

		prescriptions = append(prescriptions, p)
	}

	if err = rows.Err(); err != nil {
		// log.Printf("Error after scanning rows: %v", err)
		http.Error(w, "Error reading prescriptions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"prescriptions": prescriptions,
		"count":         len(prescriptions),
	})
}

func GetPrescriptionByAppointment(w http.ResponseWriter, r *http.Request) {
	// log.Printf("Fetching prescription by appointment")

	// Retrieve claims
	claims, ok := r.Context().Value(utils.UserClaimsKey).(*utils.Claims)
	if !ok {
		// log.Printf("Claims retrieval failed")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract appointment ID from URL
	vars := mux.Vars(r)
	appointmentID, err := strconv.Atoi(vars["appointmentId"])
	if err != nil {
		// log.Printf("Invalid appointment ID: %v", err)
		http.Error(w, "Invalid appointment ID", http.StatusBadRequest)
		return
	}

	// Fetch the prescription
	prescription, err := models.GetPrescriptionByAppointment(
		config.DB,
		appointmentID,
		claims.UserID,
		claims.IsDoctor,
	)
	if err != nil {
		if err.Error() == "unauthorized access" {
			// log.Printf("Unauthorized access to prescription for appointment %d", appointmentID)
			http.Error(w, "Unauthorized access", http.StatusForbidden)
			return
		}
		// log.Printf("Error retrieving prescription: %v", err)
		http.Error(w, "Error retrieving prescription", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prescription)
}

func GetPrescriptionWithName(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(utils.UserClaimsKey).(*utils.Claims)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	prescriptionID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid prescription ID", http.StatusBadRequest)
		return
	}

	prescription, err := models.GetPrescriptionWithName(config.DB, prescriptionID, claims.IsDoctor)
	if err != nil {
		if err.Error() == "unauthorized access" {
			http.Error(w, "Unauthorized access", http.StatusForbidden)
			return
		}
		http.Error(w, "Error retrieving prescription", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prescription)
}

func GetPrescriptionsByPatientNameAndDateHandler(w http.ResponseWriter, r *http.Request) {
	// log.Printf("Fetching prescriptions by patient name and/or date")

	// Retrieve query parameters
	patientName := r.URL.Query().Get("patientName")
	date := r.URL.Query().Get("date")

	// Validate that at least one parameter is provided
	if patientName == "" && date == "" {
		// log.Printf("No search parameters provided")
		http.Error(w, "At least one search parameter (patientName or date) is required", http.StatusBadRequest)
		return
	}

	var gregorianDate time.Time
	var err error

	// Handle date conversion if a date is provided
	if date != "" {
		// Check if the date has a prefix (1 for Hijri, 2 for Gregorian)
		if len(date) > 0 && (date[0] == '1' || date[0] == '2') {
			// Date already has a prefix, use it as is
		} else {
			// Assume the date is Hijri and prepend '1'
			date = "1" + date
		}

		// Check the first digit to determine the date type
		switch date[0] {
		case '1': // Hijri date
			gregorianDate, err = utils.SolarToGregorian(date[0:]) // Remove the first digit
			if err != nil {
				// log.Printf("Error converting Hijri date to Gregorian: %v", err)
				http.Error(w, "Invalid Hijri date format", http.StatusBadRequest)
				return
			}
		case '2': // Gregorian date
			gregorianDate, err = time.Parse("2006-01-02", date[0:]) // Remove the first digit
			if err != nil {
				// log.Printf("Error parsing Gregorian date: %v", err)
				http.Error(w, "Invalid Gregorian date format", http.StatusBadRequest)
				return
			}
		default:
			// log.Printf("Invalid date format: first digit must be 1 (Hijri) or 2 (Gregorian)")
			http.Error(w, "Invalid date format: first digit must be 1 (Hijri) or 2 (Gregorian)", http.StatusBadRequest)
			return
		}
	}

	// Fetch prescriptions from the database
	prescriptions, err := models.GetPrescriptionsByPatientNameAndDate(config.DB, patientName, gregorianDate)
	if err != nil {
		// log.Printf("Error retrieving prescriptions: %v", err)
		http.Error(w, "Error retrieving prescriptions", http.StatusInternalServerError)
		return
	}

	// Convert the response to include all fields and convert dates
	var response []models.PrescriptionResponse
	for _, p := range prescriptions {
		// Convert Gregorian date to Solar (Hijri) date
		createdAt, err := time.Parse("2006-01-02", p.CreatedAt)
		if err != nil {
			// log.Printf("Error parsing created_at: %v", err)
			http.Error(w, "Error processing dates", http.StatusInternalServerError)
			return
		}
		solarDate := utils.GregorianToSolar(createdAt)

		// Fetch medications for the prescription
		medRows, err := config.DB.Query("SELECT medicine, frequency FROM medications WHERE prescription_id = ?", p.ID)
		if err != nil {
			// log.Printf("Error fetching medications: %v", err)
			http.Error(w, "Error retrieving medications", http.StatusInternalServerError)
			return
		}
		defer medRows.Close()

		var medications []models.Medication
		for medRows.Next() {
			var med models.Medication
			if err := medRows.Scan(&med.Medicine, &med.Frequency); err != nil {
				// log.Printf("Error scanning medication: %v", err)
				http.Error(w, "Error reading medication", http.StatusInternalServerError)
				return
			}
			medications = append(medications, med)
		}

		// Build the response
		prescriptionResponse := models.PrescriptionResponse{
			ID:            p.ID,
			AppointmentID: p.AppointmentID,
			DoctorID:      p.DoctorID,
			PatientID:     p.PatientID,
			Instructions:  p.Instructions,
			Medications:   medications,
			CreatedAt:     solarDate,
			VisitType:     p.VisitType,
			PatientName:   p.PatientName,
		}

		response = append(response, prescriptionResponse)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
