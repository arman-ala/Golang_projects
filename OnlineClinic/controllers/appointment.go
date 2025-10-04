// controllers/appointment.go

package controllers

import (
	"bytes"         // For handling byte buffers
	"database/sql"  // For database operations
	"encoding/json" // For encoding/decoding JSON data
	"fmt"           // For formatted I/O operations
	"io"            // For input/output operations

	// For logging errors and informational messages
	"net/http"            // For handling HTTP requests and responses
	"onlineClinic/config" // Custom package for configuration (e.g., database connection)
	"onlineClinic/models" // Custom package for models (e.g., appointment data structures)
	"onlineClinic/utils"  // Custom package for utility functions (e.g., user claims)
	"strconv"             // For string-to-integer conversions
	"time"                // For time-related operations

	"github.com/gorilla/mux" // Gorilla Mux router for handling HTTP routes
)

// CreateAppointment handles the creation of a new appointment.
func CreateAppointment(w http.ResponseWriter, r *http.Request) {
	// log.Printf("Received appointment creation request")

	// Retrieve user claims from the context to ensure authentication.
	claims, ok := utils.GetUserClaims(r.Context())
	if !ok {
		// log.Print("No user claims found in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Decode the request body into an AppointmentRequest struct.
	var appointmentReq models.AppointmentRequest
	if err := json.NewDecoder(r.Body).Decode(&appointmentReq); err != nil {
		// log.Printf("Error decoding request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Convert the patient ID from the request to an integer for validation.
	reqPatientID, err := strconv.Atoi(appointmentReq.PatientID)
	if err != nil {
		// log.Printf("Invalid patient ID format: %v", err)
		http.Error(w, "Invalid patient ID format", http.StatusBadRequest)
		return
	}

	// Ensure the patient ID in the request matches the authenticated user's ID.
	if reqPatientID != claims.UserID {
		// log.Printf("Patient ID mismatch. Token: %d, Request: %d", claims.UserID, reqPatientID)
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Attempt to create the appointment in the database.
	if err := models.CreateAppointment(config.DB, &appointmentReq); err != nil {
		// log.Printf("Error creating appointment: %v", err)
		if err == models.ErrTimeNotAvailable {
			http.Error(w, "Selected time slot is not available", http.StatusConflict)
			return
		}
		http.Error(w, "Error creating appointment", http.StatusInternalServerError)
		return
	}

	// If successful, return a success message with a 201 Created status.
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Appointment created successfully"})
}

// GetPatientTwoNearestAppointments retrieves the two nearest upcoming appointments for a patient.
func GetPatientTwoNearestAppointments(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)                        // Extract URL parameters
	patientID, err := strconv.Atoi(vars["id"]) // Convert the patient ID to an integer
	if err != nil {
		// log.Printf("Invalid patient ID: %v", err)
		http.Error(w, "Invalid patient ID", http.StatusBadRequest)
		return
	}

	// Verify that the authenticated user is requesting their own appointments.
	claims, _ := utils.GetUserClaims(r.Context())
	if claims.UserID != patientID {
		// log.Printf("Unauthorized access attempt. Token UserID: %d, Requested PatientID: %d", claims.UserID, patientID)
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Retrieve the two nearest appointments for the patient from the database.
	appointments, err := models.GetPatientTwoNearestAppointments(config.DB, patientID)
	if err != nil {
		// log.Printf("Error retrieving appointments: %v", err)
		http.Error(w, "Error retrieving appointments", http.StatusInternalServerError)
		return
	}

	// Return the appointments as a JSON response.
	w.Header().Set("Content-Type", "application/json")
	fmt.Println(appointments)
	json.NewEncoder(w).Encode(appointments)
}

// GetDoctorTwoNearestAppointments retrieves the two nearest upcoming appointments for a doctor.
func GetDoctorTwoNearestAppointments(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)                       // Extract URL parameters
	doctorID, err := strconv.Atoi(vars["id"]) // Convert the doctor ID to an integer
	if err != nil {
		// log.Printf("Invalid doctor ID: %v", err)
		http.Error(w, "Invalid doctor ID", http.StatusBadRequest)
		return
	}

	// Verify that the authenticated user is requesting their own appointments.
	claims, _ := utils.GetUserClaims(r.Context())
	if claims.UserID != doctorID {
		// log.Printf("Unauthorized access attempt. Token UserID: %d, Requested DoctorID: %d", claims.UserID, doctorID)
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Retrieve the two nearest appointments for the doctor from the database.
	appointments, err := models.GetDoctorTwoNearestAppointments(config.DB, doctorID)
	if err != nil {
		// log.Printf("Error retrieving appointments: %v", err)
		http.Error(w, "Error retrieving appointments", http.StatusInternalServerError)
		return
	}

	// Return the appointments as a JSON response.
	w.Header().Set("Content-Type", "application/json")
	fmt.Println(appointments)
	json.NewEncoder(w).Encode(appointments)
}

// DeleteAppointment deletes an existing appointment.
func DeleteAppointment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)                            // Extract URL parameters
	appointmentID, err := strconv.Atoi(vars["id"]) // Convert the appointment ID to an integer
	if err != nil {
		// log.Printf("Invalid appointment ID: %v", err)
		http.Error(w, "Invalid appointment ID", http.StatusBadRequest)
		return
	}

	// Retrieve user claims from the context to ensure authentication.
	claims, _ := utils.GetUserClaims(r.Context())

	// Retrieve the appointment details from the database.
	appointment, err := models.GetAppointmentById(config.DB, appointmentID)
	if err != nil {
		// log.Printf("Error retrieving appointment: %v", err)
		http.Error(w, "Error retrieving appointment", http.StatusInternalServerError)
		return
	}

	// Ensure the authenticated user has permission to delete the appointment.
	if claims.IsPatient && appointment.PatientID != claims.UserID {
		// log.Printf("Unauthorized deletion attempt by patient. Token UserID: %d, Appointment PatientID: %d", claims.UserID, appointment.PatientID)
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}
	if claims.IsDoctor && appointment.DoctorID != claims.UserID {
		// log.Printf("Unauthorized deletion attempt by doctor. Token UserID: %d, Appointment DoctorID: %d", claims.UserID, appointment.DoctorID)
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Delete the appointment from the database.
	if err := models.DeleteAppointment(config.DB, appointmentID); err != nil {
		// log.Printf("Error deleting appointment: %v", err)
		http.Error(w, "Error deleting appointment", http.StatusInternalServerError)
		return
	}

	// If successful, return a success message with a 200 OK status.
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Appointment deleted successfully"})
}

// DeleteUnreservedAvailability removes unreserved availability slots for a doctor based on the visit type.
func DeleteUnreservedAvailability(w http.ResponseWriter, r *http.Request) {
	// log.Print("=== Starting DeleteUnreservedAvailability handler ===")

	// Retrieve user claims from the context to ensure authentication.
	claims, ok := utils.GetUserClaims(r.Context())
	if !ok {
		// log.Print("ERROR: No claims found in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	// log.Printf("INFO: Claims found - DoctorID: %d, IsDoctor: %v", claims.UserID, claims.IsDoctor)

	// Read and log the entire request body for debugging purposes.
	body, err := io.ReadAll(r.Body)
	if err != nil {
		// log.Printf("ERROR: Failed to read request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	// log.Printf("DEBUG: Raw request body: %s", string(body))

	// Reset the request body for further processing.
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	// Decode the request body into a struct.
	var req struct {
		Type string `json:"type"` // Key name must match the front-end
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// log.Printf("ERROR: Failed to decode JSON: %v", err)
		// log.Printf("DEBUG: Request body was: %s", string(body))
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}
	// log.Printf("INFO: Successfully decoded request - Type: %s", req.Type)

	// Validate the visit type field.
	if req.Type == "" {
		// log.Print("ERROR: Type field is empty")
		http.Error(w, "Type field is required", http.StatusBadRequest)
		return
	}

	// Normalize the visit type (e.g., convert Persian "آنلاین" to "online").
	visitType := req.Type
	switch req.Type {
	case "آنلاین":
		visitType = "online"
		// log.Printf("INFO: Converted 'آنلاین' to 'online'")
	case "online", "in-person":
		visitType = req.Type
		// log.Printf("INFO: Using provided type: %s", visitType)
	default:
		// log.Printf("ERROR: Invalid visit type: %s", req.Type)
		http.Error(w, "Visit type must be 'online', 'in-person', or 'آنلاین'", http.StatusBadRequest)
		return
	}

	// Delete the unreserved availability slots for the specified visit type.
	if err := models.DeleteUnreservedAvailability(config.DB, claims.UserID, visitType); err != nil {
		// log.Printf("ERROR: Failed to delete availability: %v", err)
		http.Error(w, fmt.Sprintf("Failed to delete availability: %v", err), http.StatusInternalServerError)
		return
	}

	// If successful, return a success message with a 200 OK status.
	// log.Printf("SUCCESS: Deleted unreserved %s slots for doctor %d", visitType, claims.UserID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("Successfully deleted unreserved %s slots", visitType),
	})
}

// GetPatientAppointments retrieves all upcoming appointments for a patient.
func GetPatientAppointments(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)                        // Extract URL parameters
	patientID, err := strconv.Atoi(vars["id"]) // Convert the patient ID to an integer
	if err != nil {
		// log.Printf("Invalid patient ID: %v", err)                  // Log the error
		http.Error(w, "Invalid patient ID", http.StatusBadRequest) // Return a 400 Bad Request response
		return
	}

	// Retrieve user claims from the context to ensure authentication.
	claims, ok := utils.GetUserClaims(r.Context())
	if !ok {
		// log.Printf("No user claims found in context")          // Log if no claims are found
		http.Error(w, "Unauthorized", http.StatusUnauthorized) // Return a 401 Unauthorized response
		return
	}

	// Ensure the authenticated user is accessing their own appointments.
	if claims.IsPatient && claims.UserID != patientID {
		// log.Printf("Unauthorized access attempt. Token UserID: %d, Requested PatientID: %d", claims.UserID, patientID)
		http.Error(w, "Unauthorized", http.StatusForbidden) // Return a 403 Forbidden response
		return
	}

	// If the user is a doctor, verify they are authorized to access the patient's appointments.
	if claims.IsDoctor {
		isAuthorized, err := isDoctorAuthorizedForPatient(claims.UserID, patientID)
		if err != nil {
			// log.Printf("Error checking doctor authorization: %v", err)             // Log any errors during authorization check
			http.Error(w, "Internal server error", http.StatusInternalServerError) // Return a 500 Internal Server Error response
			return
		}
		if !isAuthorized {
			// log.Printf("Doctor %d is not authorized to access appointments for patient %d", claims.UserID, patientID)
			http.Error(w, "Unauthorized", http.StatusForbidden) // Return a 403 Forbidden response
			return
		}
	}

	// Query the database for the patient's appointments.
	query := `
        SELECT 
            CONCAT(d.first_name, ' ', d.last_name) as name, 
            a.visit_type as type,
            a.start_time as date
        FROM appointments a
        JOIN doctors d ON a.doctor_id = d.id
        WHERE a.patient_id = ? AND a.start_time >= NOW()
        ORDER BY a.start_time ASC`

	rows, err := config.DB.Query(query, patientID) // Execute the query
	if err != nil {
		// log.Printf("Error querying appointments: %v", err)                             // Log any database query errors
		http.Error(w, "Error retrieving appointments", http.StatusInternalServerError) // Return a 500 Internal Server Error response
		return
	}
	defer rows.Close() // Ensure the database rows are closed after use

	// Define a struct to represent the appointment response.
	type AppointmentListResponse struct {
		Name string `json:"name"` // Doctor's name
		Type string `json:"type"` // Visit type (e.g., online, in-person)
		Date string `json:"date"` // Appointment date (in Hijri format)
	}

	var appointments []AppointmentListResponse // Slice to store the appointment data

	// Iterate through the query results and populate the appointments slice.
	for rows.Next() {
		var appt AppointmentListResponse
		var dateStr []uint8 // Use []uint8 to read the raw date string from the database
		err := rows.Scan(&appt.Name, &appt.Type, &dateStr)
		if err != nil {
			// log.Printf("Error scanning appointment: %v", err)                              // Log any errors during row scanning
			http.Error(w, "Error processing appointments", http.StatusInternalServerError) // Return a 500 Internal Server Error response
			return
		}

		// Convert the []uint8 date string to a time.Time object.
		startTime, err := time.Parse("2006-01-02 15:04:05", string(dateStr)) // Adjust the layout based on your date format
		if err != nil {
			// log.Printf("Error parsing date: %v", err)                              // Log any errors during date parsing
			http.Error(w, "Error processing date", http.StatusInternalServerError) // Return a 500 Internal Server Error response
			return
		}

		// Convert Gregorian date to Hijri date using a utility function.
		appt.Date = utils.GregorianToSolar(startTime)
		appointments = append(appointments, appt) // Add the appointment to the slice
	}

	// Check for any errors after iterating through the rows.
	if err = rows.Err(); err != nil {
		// log.Printf("Error after iterating rows: %v", err)                              // Log any errors that occurred during iteration
		http.Error(w, "Error processing appointments", http.StatusInternalServerError) // Return a 500 Internal Server Error response
		return
	}

	// Log the number of retrieved appointments.
	// log.Printf("Retrieved %d appointments for patient %d", len(appointments), patientID)

	// Set the response content type to JSON and encode the appointments slice.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(appointments)
}

// isDoctorAuthorizedForPatient checks if a doctor is authorized to access a patient's appointments.
func isDoctorAuthorizedForPatient(doctorID, patientID int) (bool, error) {
	// Query the database to check if the patient has an appointment with the doctor.
	query := `
        SELECT EXISTS (
            SELECT 1
            FROM appointments
            WHERE doctor_id = ? AND patient_id = ?
        )`
	var exists bool
	err := config.DB.QueryRow(query, doctorID, patientID).Scan(&exists)
	if err != nil {
		return false, err // Return false and the error if the query fails
	}
	return exists, nil // Return true if the doctor is authorized, otherwise false
}

// GetDoctorAppointments retrieves all future appointments for a doctor
func GetDoctorAppointments(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	doctorID, err := strconv.Atoi(vars["id"])
	if err != nil {
		// log.Printf("Invalid doctor ID: %v", err)
		http.Error(w, "Invalid doctor ID", http.StatusBadRequest)
		return
	}

	// Verify the authenticated user is requesting their own appointments
	claims, _ := utils.GetUserClaims(r.Context())
	if claims.UserID != doctorID {
		// log.Printf("Unauthorized access attempt. Token UserID: %d, Requested DoctorID: %d", claims.UserID, doctorID)
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Load Asia/Tehran timezone
	tehranLoc, err := time.LoadLocation("Asia/Tehran")
	if err != nil {
		// log.Printf("Error loading Asia/Tehran timezone: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Get current time in Asia/Tehran and format for SQL query
	now := time.Now().In(tehranLoc)
	nowFormatted := now.Format("2006-01-02 15:04:05")

	query := `
        SELECT 
            CONCAT(p.first_name, ' ', p.last_name) as name,
            a.visit_type as type,
            DATE_FORMAT(a.start_time, '%Y-%m-%d %H:%i:%s') as start_time,
            DATE_FORMAT(a.start_time, '%H:%i') as time
        FROM appointments a
        JOIN patients p ON a.patient_id = p.id
        WHERE a.doctor_id = ? AND a.start_time >= ?
        ORDER BY a.start_time ASC`

	rows, err := config.DB.Query(query, doctorID, nowFormatted)
	if err != nil {
		// log.Printf("Error querying appointments: %v", err)
		http.Error(w, "Error retrieving appointments", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type DoctorAppointmentResponse struct {
		PatientName string `json:"name"`
		Type        string `json:"type"`
		Date        string `json:"date"`
		Time        string `json:"time"`
	}

	var appointments []DoctorAppointmentResponse
	for rows.Next() {
		var appt DoctorAppointmentResponse
		var startTimeStr string
		err := rows.Scan(&appt.PatientName, &appt.Type, &startTimeStr, &appt.Time)
		if err != nil {
			// log.Printf("Error scanning appointment: %v", err)
			http.Error(w, "Error processing appointments", http.StatusInternalServerError)
			return
		}

		// Parse the start_time string
		startTime, err := time.Parse("2006-01-02 15:04:05", startTimeStr)
		if err != nil {
			// log.Printf("Error parsing date: %v", err)
			http.Error(w, "Error processing date", http.StatusInternalServerError)
			return
		}

		// Convert to Asia/Tehran timezone
		startTime = startTime.In(tehranLoc)

		// Convert Gregorian date to Hijri date
		appt.Date = utils.GregorianToSolar(startTime)
		// Time is already formatted by SQL as HH:MM, no further conversion needed

		appointments = append(appointments, appt)
	}

	if err = rows.Err(); err != nil {
		// log.Printf("Error after iterating rows: %v", err)
		http.Error(w, "Error processing appointments", http.StatusInternalServerError)
		return
	}

	// log.Printf("Retrieved %d appointments for doctor %d", len(appointments), doctorID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(appointments)
}

// GetDoctorAllAppointments retrieves all appointments (past and future) for a doctor.
func GetDoctorAllAppointments(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)                       // Extract URL parameters
	doctorID, err := strconv.Atoi(vars["id"]) // Convert the doctor ID to an integer
	if err != nil {
		// log.Printf("Invalid doctor ID: %v", err)                  // Log the error
		http.Error(w, "Invalid doctor ID", http.StatusBadRequest) // Return a 400 Bad Request response
		return
	}

	// Verify the authenticated user is requesting their own appointments.
	claims, _ := utils.GetUserClaims(r.Context())
	if claims.UserID != doctorID {
		// log.Printf("Unauthorized access attempt. Token UserID: %d, Requested DoctorID: %d", claims.UserID, doctorID)
		http.Error(w, "Unauthorized", http.StatusForbidden) // Return a 403 Forbidden response
		return
	}

	// Query the database for all the doctor's appointments.
	query := `
        SELECT 
            a.id as appointment_id,
            a.doctor_id,
            a.patient_id,
            a.visit_type as type,
            a.start_time as date,
            DATE_FORMAT(a.start_time, '%H:%i') as time,
            CONCAT(p.first_name, ' ', p.last_name) as name
        FROM appointments a
        JOIN patients p ON a.patient_id = p.id
        WHERE a.doctor_id = ?
        ORDER BY a.start_time ASC`

	rows, err := config.DB.Query(query, doctorID) // Execute the query
	if err != nil {
		// log.Printf("Error querying appointments: %v", err)                             // Log any database query errors
		http.Error(w, "Error retrieving appointments", http.StatusInternalServerError) // Return a 500 Internal Server Error response
		return
	}
	defer rows.Close() // Ensure the database rows are closed after use

	// Define a struct to represent the appointment response.
	type DoctorAppointmentResponse struct {
		ID          string `json:"id"`        // Appointment ID
		DoctorID    string `json:"doctorId"`  // Doctor ID
		PatientID   string `json:"patientId"` // Patient ID
		Type        string `json:"type"`      // Visit type (e.g., online, in-person)
		Date        string `json:"date"`      // Appointment date (in Hijri format)
		Time        string `json:"time"`      // Appointment time
		PatientName string `json:"name"`      // Patient's name
	}

	var appointments []DoctorAppointmentResponse // Slice to store the appointment data

	// Iterate through the query results and populate the appointments slice.
	for rows.Next() {
		var appt DoctorAppointmentResponse
		var appointmentID, doctorID, patientID int
		var dateStr []uint8 // Use []uint8 to read the raw date string from the database
		err := rows.Scan(&appointmentID, &doctorID, &patientID, &appt.Type, &dateStr, &appt.Time, &appt.PatientName)
		if err != nil {
			// log.Printf("Error scanning appointment: %v", err)                              // Log any errors during row scanning
			http.Error(w, "Error processing appointments", http.StatusInternalServerError) // Return a 500 Internal Server Error response
			return
		}

		// Convert the []uint8 date string to a time.Time object.
		startTime, err := time.Parse(time.RFC3339, string(dateStr)) // Use RFC3339 format
		if err != nil {
			// log.Printf("Error parsing date: %v", err)                              // Log any errors during date parsing
			http.Error(w, "Error processing date", http.StatusInternalServerError) // Return a 500 Internal Server Error response
			return
		}

		// Convert IDs to strings.
		appt.ID = strconv.Itoa(appointmentID)
		appt.DoctorID = strconv.Itoa(doctorID)
		appt.PatientID = strconv.Itoa(patientID)

		// Convert Gregorian date to Hijri date using a utility function.
		appt.Date = utils.GregorianToSolar(startTime)
		appointments = append(appointments, appt) // Add the appointment to the slice
	}

	// Check for any errors after iterating through the rows.
	if err = rows.Err(); err != nil {
		// log.Printf("Error after iterating rows: %v", err)                              // Log any errors that occurred during iteration
		http.Error(w, "Error processing appointments", http.StatusInternalServerError) // Return a 500 Internal Server Error response
		return
	}

	// Log the number of retrieved appointments.
	// log.Printf("Retrieved %d appointments for doctor %d", len(appointments), doctorID)

	// Set the response content type to JSON and encode the appointments slice.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(appointments)
}

// GetPatientAllAppointments retrieves all appointments (past and future) for a patient.
func GetPatientAllAppointments(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)                        // Extract URL parameters
	patientID, err := strconv.Atoi(vars["id"]) // Convert the patient ID to an integer
	if err != nil {
		// log.Printf("Invalid patient ID: %v", err)                  // Log the error
		http.Error(w, "Invalid patient ID", http.StatusBadRequest) // Return a 400 Bad Request response
		return
	}

	// Retrieve user claims from the context to ensure authentication.
	claims, ok := utils.GetUserClaims(r.Context())
	if !ok {
		// log.Printf("Failed to retrieve user claims")           // Log if no claims are found
		http.Error(w, "Unauthorized", http.StatusUnauthorized) // Return a 401 Unauthorized response
		return
	}

	// Ensure the authenticated user is accessing their own appointments or is authorized as a doctor.
	if claims.IsPatient && claims.UserID != patientID {
		// log.Printf("Unauthorized access attempt. Token UserID: %d, Requested PatientID: %d", claims.UserID, patientID)
		http.Error(w, "Unauthorized", http.StatusForbidden) // Return a 403 Forbidden response
		return
	} else if !claims.IsDoctor && !claims.IsPatient {
		// log.Printf("Unauthorized access attempt. User is neither a doctor nor a patient")
		http.Error(w, "Unauthorized", http.StatusForbidden) // Return a 403 Forbidden response
		return
	}

	// Build the query based on the user's role.
	query := `
        SELECT 
            a.id as appointment_id,
            a.doctor_id,
            a.patient_id,
            a.visit_type as type,
            a.start_time as date,
            DATE_FORMAT(a.start_time, '%H:%i') as time,
            CONCAT(d.first_name, ' ', d.last_name) as name
        FROM appointments a
        JOIN doctors d ON a.doctor_id = d.id
        WHERE a.patient_id = ?`

	// If the user is a doctor, ensure they are only accessing appointments of their patients.
	if claims.IsDoctor {
		query += " AND a.doctor_id = ?"
	}

	query += " ORDER BY a.start_time ASC"

	// Execute the query.
	var rows *sql.Rows
	if claims.IsDoctor {
		rows, err = config.DB.Query(query, patientID, claims.UserID)
	} else {
		rows, err = config.DB.Query(query, patientID)
	}
	if err != nil {
		// log.Printf("Error querying appointments: %v", err)                             // Log any database query errors
		http.Error(w, "Error retrieving appointments", http.StatusInternalServerError) // Return a 500 Internal Server Error response
		return
	}
	defer rows.Close() // Ensure the database rows are closed after use

	// Define a struct to represent the appointment response.
	type PatientAppointmentResponse struct {
		ID        string `json:"id"`        // Appointment ID
		DoctorID  string `json:"doctorId"`  // Doctor ID
		PatientID string `json:"patientId"` // Patient ID
		Type      string `json:"type"`      // Visit type (e.g., online, in-person)
		Date      string `json:"date"`      // Appointment date (in Hijri format)
		Time      string `json:"time"`      // Appointment time
		Name      string `json:"name"`      // Doctor's name
	}

	var appointments []PatientAppointmentResponse // Slice to store the appointment data

	// Iterate through the query results and populate the appointments slice.
	for rows.Next() {
		var appt PatientAppointmentResponse
		var appointmentID, doctorID, patientID int
		var dateStr []uint8 // Use []uint8 to read the raw date string from the database
		err := rows.Scan(&appointmentID, &doctorID, &patientID, &appt.Type, &dateStr, &appt.Time, &appt.Name)
		if err != nil {
			// log.Printf("Error scanning appointment: %v", err)                              // Log any errors during row scanning
			http.Error(w, "Error processing appointments", http.StatusInternalServerError) // Return a 500 Internal Server Error response
			return
		}

		// Convert the []uint8 date string to a time.Time object.
		startTime, err := time.Parse(time.RFC3339, string(dateStr)) // Use RFC3339 format
		if err != nil {
			// log.Printf("Error parsing date: %v", err)                              // Log any errors during date parsing
			http.Error(w, "Error processing date", http.StatusInternalServerError) // Return a 500 Internal Server Error response
			return
		}

		// Convert IDs to strings.
		appt.ID = strconv.Itoa(appointmentID)
		appt.DoctorID = strconv.Itoa(doctorID)
		appt.PatientID = strconv.Itoa(patientID)

		// Convert Gregorian date to Hijri date using a utility function.
		appt.Date = utils.GregorianToSolar(startTime)
		appointments = append(appointments, appt) // Add the appointment to the slice
	}

	// Check for any errors after iterating through the rows.
	if err = rows.Err(); err != nil {
		// log.Printf("Error after iterating rows: %v", err)                              // Log any errors that occurred during iteration
		http.Error(w, "Error processing appointments", http.StatusInternalServerError) // Return a 500 Internal Server Error response
		return
	}

	// Log the number of retrieved appointments.
	// log.Printf("Retrieved %d appointments for patient %d", len(appointments), patientID)

	// Set the response content type to JSON and encode the appointments slice.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(appointments)
}
