package models

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"onlineClinic/utils"
	"sort"
	"strconv"
	"time"
)

type Appointment struct {
	ID         int       `json:"id"`
	PatientID  int       `json:"patientId"`
	DoctorID   int       `json:"doctorId"`
	StartTime  time.Time `json:"startTime"`
	EndTime    time.Time `json:"endTime"`
	VisitType  string    `json:"visitType"` // 'online' or 'in-person'
	CreatedAt  time.Time `json:"createdAt"`
	DoctorName string    `json:"name,omitempty"` // For GET responses
	Date       string    `json:"date,omitempty"` // For GET responses
	Time       string    `json:"time,omitempty"` // For GET responses
}

type AppointmentResponse struct {
	ID        int    `json:"id"`
	DoctorID  int    `json:"doctorId"`
	PatientID int    `json:"patientId"`
	Type      string `json:"type"`
	Date      string `json:"date"`
	Time      string `json:"time"`
	Name      string `json:"name"`
}

type AppointmentRequest struct {
	DoctorID  string `json:"doctorId"`
	PatientID string `json:"patientId"`
	Type      string `json:"type"` // آنلاین or in-person
	Date      string `json:"date"` // Format: "1403-08-23"
	Time      string `json:"time"` // Format: "12:00"
}

var (
	ErrTimeNotAvailable = errors.New("selected time slot is not available")
	ErrInvalidTimeSlot  = errors.New("invalid time slot")
	// ErrTimeInPast is returned when the requested appointment time is in the past.
	ErrTimeInPast = fmt.Errorf("cannot book appointment for a time in the past")
)

func (ar *AppointmentRequest) ToAppointment() (*Appointment, error) {
	log.Printf("Converting appointment request to appointment: %+v", ar)

	// Convert IDs from string to int
	doctorID, err := strconv.Atoi(ar.DoctorID)
	if err != nil {
		log.Printf("Invalid doctor ID format: %v", err)
		return nil, errors.New("invalid doctor ID format")
	}

	patientID, err := strconv.Atoi(ar.PatientID)
	if err != nil {
		log.Printf("Invalid patient ID format: %v", err)
		return nil, errors.New("invalid patient ID format")
	}

	// Convert Hijri date to Gregorian
	gregorianDate, err := utils.SolarToGregorian(ar.Date) // Convert Hijri to Gregorian
	if err != nil {
		log.Printf("Error converting Hijri date to Gregorian: %v", err)
		return nil, fmt.Errorf("invalid date format: %v", err)
	}

	log.Printf("Converted Hijri date %s to Gregorian: %v", ar.Date, gregorianDate)

	// Parse the combined date and time (server is in Tehran/Iran, no location needed)
	startTime, err := time.Parse("2006-01-02 15:04", gregorianDate.Format("2006-01-02")+" "+ar.Time)
	if err != nil {
		log.Printf("Error parsing date and time: %v", err)
		return nil, fmt.Errorf("invalid date or time format: %v", err)
	}

	log.Printf("Parsed start time: %v", startTime)

	// Calculate end time (15 minutes after start time)
	endTime := startTime.Add(15 * time.Minute)

	// Convert type (آنلاین to online if needed)
	visitType := ar.Type
	if visitType == "آنلاین" {
		visitType = "online"
	}

	appointment := &Appointment{
		DoctorID:  doctorID,
		PatientID: patientID,
		StartTime: startTime,
		EndTime:   endTime,
		VisitType: visitType,
	}

	log.Printf("Converted to appointment: %+v", appointment)
	return appointment, nil
}

func CreateAppointment(db *sql.DB, req *AppointmentRequest) error {
	log.Printf("Processing appointment request: %+v", req)

	appointment, err := req.ToAppointment()
	if err != nil {
		log.Printf("Error converting request to appointment: %v", err)
		return err
	}

	// Check if the time slot is available (match exact timestamps)
	var available bool
	query := `
        SELECT EXISTS(
            SELECT 1 FROM doctor_availability 
            WHERE doctor_id = ? 
            AND start_time = ?
            AND end_time = ?
            AND type = ?
        )`
	err = db.QueryRow(query, appointment.DoctorID, appointment.StartTime, appointment.EndTime, appointment.VisitType).Scan(&available)
	if err != nil {
		log.Printf("Error executing availability query: %v", err)
		log.Printf("Query: %s, Params: doctorID=%d, startTime=%v, endTime=%v, visitType=%s", query, appointment.DoctorID, appointment.StartTime, appointment.EndTime, appointment.VisitType)
		return err
	}

	log.Printf("Availability check result: slot available=%v", available)

	if !available {
		log.Printf("Time slot not available for doctor %d at %v", appointment.DoctorID, appointment.StartTime)
		return ErrTimeNotAvailable
	}

	if !appointment.StartTime.After(time.Now()) {
		return ErrTimeInPast
	}

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		return err
	}
	defer tx.Rollback()

	// Insert appointment
	result, err := tx.Exec(`
        INSERT INTO appointments (
            patient_id, doctor_id, start_time, end_time, visit_type
        ) VALUES (?, ?, ?, ?, ?)`,
		appointment.PatientID,
		appointment.DoctorID,
		appointment.StartTime,
		appointment.EndTime,
		appointment.VisitType,
	)
	if err != nil {
		log.Printf("Error inserting appointment: %v", err)
		return err
	}

	// Get the appointment ID
	appointmentID, err := result.LastInsertId()
	if err != nil {
		log.Printf("Error getting last insert ID: %v", err)
		return err
	}

	// Create a new prescription
	prescription := &Prescription{
		AppointmentID: int(appointmentID),
		Instructions:  "",
		Medications:   []Medication{},
	}

	if err := CreatePrescriptionDB(tx, prescription); err != nil {
		log.Printf("Error creating prescription: %v", err)
		return err
	}

	// Delete the availability slot
	_, err = tx.Exec(`
        DELETE FROM doctor_availability 
        WHERE doctor_id = ? 
        AND start_time = ?
        AND end_time = ?
        AND type = ?`,
		appointment.DoctorID,
		appointment.StartTime,
		appointment.EndTime,
		appointment.VisitType,
	)
	if err != nil {
		log.Printf("Error deleting availability slot: %v", err)
		return err
	}

	if err = tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		return err
	}

	log.Printf("Successfully created appointment %d and associated prescription", appointmentID)
	return nil
}

func GetPatientTwoNearestAppointments(db *sql.DB, patientID int) ([]AppointmentResponse, error) {
	log.Printf("Getting two nearest appointments for patient %d", patientID)

	now := time.Now()
	log.Printf("Current server time: %v", now.Format("2006-01-02 15:04:05"))

	// Step 1: Fetch all recent and future appointments
	query := `
        SELECT 
            a.id,
            a.doctor_id,
            a.patient_id,
            a.visit_type,
            a.start_time,
            a.end_time,
            CONCAT(d.first_name, ' ', d.last_name) AS doctor_name
        FROM appointments a
        JOIN doctors d ON a.doctor_id = d.id
        WHERE a.patient_id = ?
          AND a.start_time >= ?`

	rows, err := db.Query(query, patientID, now.Add(-15*time.Minute))
	if err != nil {
		log.Printf("Error querying appointments: %v", err)
		return nil, err
	}
	defer rows.Close()

	var appointments []Appointment
	for rows.Next() {
		var appt Appointment
		err := rows.Scan(
			&appt.ID,
			&appt.DoctorID,
			&appt.PatientID,
			&appt.VisitType,
			&appt.StartTime,
			&appt.EndTime,
			&appt.DoctorName,
		)
		if err != nil {
			log.Printf("Error scanning appointment: %v", err)
			return nil, err
		}
		appointments = append(appointments, appt)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error iterating rows: %v", err)
		return nil, err
	}

	var ongoing *Appointment
	var upcoming []Appointment
	nowMinute := now.Hour()*60 + now.Minute()

	for _, appt := range appointments {
		start := appt.StartTime
		end := appt.EndTime
		apptMinute := start.Hour()*60 + start.Minute()

		if start.After(now.Add(-15*time.Minute)) && start.Before(now) && end.After(now) {
			ongoing = &appt
		} else if apptMinute > nowMinute {
			upcoming = append(upcoming, appt)
		}
	}

	// Sort upcoming by minute value (hour*60 + minute)
	sort.Slice(upcoming, func(i, j int) bool {
		iMinute := upcoming[i].StartTime.Hour()*60 + upcoming[i].StartTime.Minute()
		jMinute := upcoming[j].StartTime.Hour()*60 + upcoming[j].StartTime.Minute()
		return iMinute < jMinute
	})

	// Sort upcoming by start time
	sort.Slice(upcoming, func(i, j int) bool {
		return upcoming[i].StartTime.Before(upcoming[j].StartTime)
	})

	var result []AppointmentResponse

	// Add ongoing session if exists
	if ongoing != nil {
		result = append(result, AppointmentResponse{
			ID:        ongoing.ID,
			DoctorID:  ongoing.DoctorID,
			PatientID: ongoing.PatientID,
			Type:      ongoing.VisitType,
			Date:      utils.GregorianToSolar(ongoing.StartTime),
			Time:      ongoing.StartTime.Format("15:04"),
			Name:      ongoing.DoctorName,
		})
	}

	// Add up to 2 upcoming sessions
	for i := 0; i < len(upcoming) && len(result) < 2; i++ {
		appt := upcoming[i]
		result = append(result, AppointmentResponse{
			ID:        appt.ID,
			DoctorID:  appt.DoctorID,
			PatientID: appt.PatientID,
			Type:      appt.VisitType,
			Date:      utils.GregorianToSolar(appt.StartTime),
			Time:      appt.StartTime.Format("15:04"),
			Name:      appt.DoctorName,
		})
	}

	log.Printf("Returning %d nearest appointments for patient %d", len(result), patientID)
	return result, nil
}

func GetDoctorTwoNearestAppointments(db *sql.DB, doctorID int) ([]AppointmentResponse, error) {
	log.Printf("Getting two nearest appointments for doctor %d", doctorID)

	now := time.Now()
	log.Printf("Current server time: %v", now.Format("2006-01-02 15:04:05"))

	// Step 1: Fetch all recent and future appointments
	query := `
        SELECT 
            a.id,
            a.doctor_id,
            a.patient_id,
            a.visit_type,
            a.start_time,
            a.end_time,
            CONCAT(p.first_name, ' ', p.last_name) AS patient_name
        FROM appointments a
        JOIN patients p ON a.patient_id = p.id
        WHERE a.doctor_id = ?
          AND a.start_time >= ?`

	rows, err := db.Query(query, doctorID, now.Add(-15*time.Minute))
	if err != nil {
		log.Printf("Error querying appointments: %v", err)
		return nil, err
	}
	defer rows.Close()

	var appointments []Appointment
	for rows.Next() {
		var appt Appointment
		err := rows.Scan(
			&appt.ID,
			&appt.DoctorID,
			&appt.PatientID,
			&appt.VisitType,
			&appt.StartTime,
			&appt.EndTime,
			&appt.DoctorName,
		)
		if err != nil {
			log.Printf("Error scanning appointment: %v", err)
			return nil, err
		}
		appointments = append(appointments, appt)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error iterating rows: %v", err)
		return nil, err
	}

	var ongoing *Appointment
	var upcoming []Appointment
	nowMinute := now.Hour()*60 + now.Minute()

	for _, appt := range appointments {
		start := appt.StartTime
		end := appt.EndTime
		apptMinute := start.Hour()*60 + start.Minute()

		if start.After(now.Add(-15*time.Minute)) && start.Before(now) && end.After(now) {
			ongoing = &appt
		} else if apptMinute > nowMinute {
			upcoming = append(upcoming, appt)
		}
	}

	// Sort upcoming by minute value (hour*60 + minute)
	sort.Slice(upcoming, func(i, j int) bool {
		iMinute := upcoming[i].StartTime.Hour()*60 + upcoming[i].StartTime.Minute()
		jMinute := upcoming[j].StartTime.Hour()*60 + upcoming[j].StartTime.Minute()
		return iMinute < jMinute
	})

	// Sort upcoming by start time
	sort.Slice(upcoming, func(i, j int) bool {
		return upcoming[i].StartTime.Before(upcoming[j].StartTime)
	})

	var result []AppointmentResponse

	// Add ongoing session if exists
	if ongoing != nil {
		result = append(result, AppointmentResponse{
			ID:        ongoing.ID,
			DoctorID:  ongoing.DoctorID,
			PatientID: ongoing.PatientID,
			Type:      ongoing.VisitType,
			Date:      utils.GregorianToSolar(ongoing.StartTime),
			Time:      ongoing.StartTime.Format("15:04"),
			Name:      ongoing.DoctorName,
		})
	}

	// Add up to 2 upcoming sessions
	for i := 0; i < len(upcoming) && len(result) < 2; i++ {
		appt := upcoming[i]
		result = append(result, AppointmentResponse{
			ID:        appt.ID,
			DoctorID:  appt.DoctorID,
			PatientID: appt.PatientID,
			Type:      appt.VisitType,
			Date:      utils.GregorianToSolar(appt.StartTime),
			Time:      appt.StartTime.Format("15:04"),
			Name:      appt.DoctorName,
		})
	}

	log.Printf("Returning %d nearest appointments for doctor %d", len(result), doctorID)
	return result, nil
}

func GetAppointmentById(db *sql.DB, id int) (*Appointment, error) {
	// log.Printf("Getting appointment with ID: %d", id)

	// Use NullTime to handle potential NULL values from database
	var (
		startTimeStr sql.NullString
		endTimeStr   sql.NullString
		createdAtStr sql.NullString
		appointment  Appointment
	)

	query := `
        SELECT id, patient_id, doctor_id, 
               DATE_FORMAT(start_time, '%Y-%m-%d %H:%i:%s') as start_time,
               DATE_FORMAT(end_time, '%Y-%m-%d %H:%i:%s') as end_time,
               visit_type,
               DATE_FORMAT(created_at, '%Y-%m-%d %H:%i:%s') as created_at
        FROM appointments 
        WHERE id = ?`

	err := db.QueryRow(query, id).Scan(
		&appointment.ID,
		&appointment.PatientID,
		&appointment.DoctorID,
		&startTimeStr,
		&endTimeStr,
		&appointment.VisitType,
		&createdAtStr,
	)

	if err == sql.ErrNoRows {
		// log.Printf("No appointment found with ID: %d", id)
		return nil, errors.New("appointment not found")
	}
	if err != nil {
		// log.Printf("Error querying appointment: %v", err)
		return nil, err
	}

	// Parse the datetime strings
	if startTimeStr.Valid {
		startTime, err := time.Parse("2006-01-02 15:04:05", startTimeStr.String)
		if err != nil {
			// log.Printf("Error parsing start time: %v", err)
			return nil, fmt.Errorf("error parsing start time: %v", err)
		}
		appointment.StartTime = startTime
	}

	if endTimeStr.Valid {
		endTime, err := time.Parse("2006-01-02 15:04:05", endTimeStr.String)
		if err != nil {
			// log.Printf("Error parsing end time: %v", err)
			return nil, fmt.Errorf("error parsing end time: %v", err)
		}
		appointment.EndTime = endTime
	}

	if createdAtStr.Valid {
		createdAt, err := time.Parse("2006-01-02 15:04:05", createdAtStr.String)
		if err != nil {
			// log.Printf("Error parsing created at time: %v", err)
			return nil, fmt.Errorf("error parsing created at time: %v", err)
		}
		appointment.CreatedAt = createdAt
	}

	// log.Printf("Successfully retrieved appointment: %+v", appointment)
	return &appointment, nil
}

func DeleteUnreservedAvailability(db *sql.DB, doctorID int, visitType string) error {
	// log.Printf("Starting DeleteUnreservedAvailability for doctor %d and type %s", doctorID, visitType)

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		// log.Printf("Error starting transaction: %v", err)
		return fmt.Errorf("failed to start transaction: %v", err)
	}
	defer func() {
		if err != nil {
			// log.Printf("Rolling back transaction due to error: %v", err)
			tx.Rollback()
		}
	}()

	// Set up variable to receive deleted count
	var deletedCount int
	// log.Printf("Calling delete_unreserved_availability procedure for doctor %d and type %s", doctorID, visitType)

	// Call the stored procedure
	_, err = tx.Exec("CALL delete_unreserved_availability(?, ?, @deleted_count)",
		doctorID, visitType)
	if err != nil {
		// log.Printf("Error calling delete procedure: %v", err)
		return fmt.Errorf("failed to delete availability slots: %v", err)
	}

	// Get the output parameter value
	err = tx.QueryRow("SELECT @deleted_count").Scan(&deletedCount)
	if err != nil {
		// log.Printf("Error getting deleted count: %v", err)
		return fmt.Errorf("failed to get deleted count: %v", err)
	}

	// log.Printf("Procedure reports %d slots deleted", deletedCount)

	// Commit transaction
	if err = tx.Commit(); err != nil {
		// log.Printf("Error committing transaction: %v", err)
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	// log.Printf("Successfully completed DeleteUnreservedAvailability for doctor %d. Deleted %d slots", doctorID, deletedCount)
	return nil
}

func DeleteAppointment(db *sql.DB, id int) error {
	// log.Printf("Attempting to delete appointment with ID: %d", id)

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		// log.Printf("Error starting transaction: %v", err)
		return err
	}
	defer tx.Rollback()

	// Get appointment details first using proper scanning
	var (
		doctorID  int
		startTime sql.NullString
		endTime   sql.NullString
		visitType string
	)

	err = tx.QueryRow(`
        SELECT doctor_id, 
               DATE_FORMAT(start_time, '%Y-%m-%d %H:%i:%s') as start_time,
               DATE_FORMAT(end_time, '%Y-%m-%d %H:%i:%s') as end_time,
               visit_type 
        FROM appointments 
        WHERE id = ?`, id).Scan(&doctorID, &startTime, &endTime, &visitType)
	if err != nil {
		// log.Printf("Error getting appointment details: %v", err)
		return err
	}

	// Parse the time strings
	start, err := time.Parse("2006-01-02 15:04:05", startTime.String)
	if err != nil {
		// log.Printf("Error parsing start time: %v", err)
		return err
	}

	end, err := time.Parse("2006-01-02 15:04:05", endTime.String)
	if err != nil {
		// log.Printf("Error parsing end time: %v", err)
		return err
	}

	// Delete associated prescription first (due to foreign key constraint)
	_, err = tx.Exec("DELETE FROM prescriptions WHERE appointment_id = ?", id)
	if err != nil {
		// log.Printf("Error deleting associated prescription: %v", err)
		return err
	}

	// Delete the appointment
	result, err := tx.Exec("DELETE FROM appointments WHERE id = ?", id)
	if err != nil {
		// log.Printf("Error deleting appointment: %v", err)
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		// log.Printf("Error getting rows affected: %v", err)
		return err
	}
	if rowsAffected == 0 {
		// log.Printf("No appointment found with ID: %d", id)
		return errors.New("appointment not found")
	}

	// Restore the availability slot
	_, err = tx.Exec(`
        INSERT INTO doctor_availability (doctor_id, start_time, end_time, type)
        VALUES (?, ?, ?, ?)`,
		doctorID, start, end, visitType)
	if err != nil {
		// log.Printf("Error restoring availability slot: %v", err)
		return err
	}

	if err = tx.Commit(); err != nil {
		// log.Printf("Error committing transaction: %v", err)
		return err
	}

	// log.Printf("Successfully deleted appointment %d and restored availability slot", id)
	return nil
}
