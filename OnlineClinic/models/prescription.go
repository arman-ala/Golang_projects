// models/prescription.go
package models

import (
	"database/sql"
	"fmt"
	"onlineClinic/utils"
	"time"
)

type Medication struct {
	Medicine  string `json:"medicine"`
	Frequency string `json:"frequency"`
}

type Prescription struct {
	ID            int          `json:"id"`
	AppointmentID int          `json:"appointmentId"`
	Instructions  string       `json:"instructions"`
	CreatedAt     string       `json:"createdAt"`
	Medications   []Medication `json:"medications"`
}

type PrescriptionResponse struct {
	ID            int          `json:"id"`
	AppointmentID int          `json:"appointmentId"`
	DoctorID      int          `json:"doctorId"`
	PatientID     int          `json:"patientId"`
	Instructions  string       `json:"instructions"`
	Medications   []Medication `json:"medications"`
	CreatedAt     string       `json:"createdAt"`
	VisitType     string       `json:"visitType"`
	PatientName   string       `json:"patientName,omitempty"` // Patient's name (for doctors)
	DoctorName    string       `json:"doctorName,omitempty"`  // Doctor's name (for patients)
}

type PrescriptionRequest struct {
	AppointmentID int          `json:"appointmentId"`
	Instructions  string       `json:"instructions"`
	Medications   []Medication `json:"medications"`
}

type PrescriptionWithDetails struct {
	ID            int    `json:"id"`
	PatientID     int    `json:"patientId"`
	DoctorID      int    `json:"doctorId"`
	AppointmentID int    `json:"appointmentId"` // Added field
	Instructions  string `json:"instructions"`
	CreatedAt     string `json:"createdAt"`
	VisitType     string `json:"visitType"`   // Added field
	PatientName   string `json:"patientName"` // Added field
}

func CreatePrescriptionDB(tx *sql.Tx, prescription *Prescription) error {
	// Fetch the appointment's start_date
	var startDateStr string // Use a string to temporarily store the raw value
	err := tx.QueryRow(`
        SELECT start_time 
        FROM appointments 
        WHERE id = ?`, prescription.AppointmentID).Scan(&startDateStr)
	if err != nil {
		return err
	}

	// Parse the string into a time.Time object
	startDate, err := time.Parse(time.RFC3339, startDateStr) // Use RFC3339 format
	if err != nil {
		return fmt.Errorf("error parsing start_time: %v", err)
	}

	// Insert prescription with the appointment's start_date as created_at
	query := `INSERT INTO prescriptions (
        appointment_id, instructions, created_at
    ) VALUES (?, ?, ?)`

	result, err := tx.Exec(query,
		prescription.AppointmentID,
		prescription.Instructions,
		startDate.Format("2006-01-02"), // Format the start_date to YYYY-MM-DD
	)
	if err != nil {
		return err
	}

	prescriptionID, err := result.LastInsertId()
	if err != nil {
		return err
	}

	// Insert medications
	medicationQuery := `INSERT INTO medications (
        prescription_id, medicine, frequency
    ) VALUES (?, ?, ?)`

	for _, med := range prescription.Medications {
		_, err := tx.Exec(medicationQuery,
			prescriptionID,
			med.Medicine,
			med.Frequency,
		)
		if err != nil {
			return err
		}
	}

	// Set the ID in the prescription object so it can be used in the response
	prescription.ID = int(prescriptionID)

	// Set the created_at value in the prescription object
	prescription.CreatedAt = startDate.Format("2006-01-02")
	// log.Printf("appointment start date is: %v", startDate)
	return nil
}

func GetPrescriptionByAppointment(db *sql.DB, appointmentID, userID int, isDoctor bool) (*PrescriptionResponse, error) {
	// log.Printf("Fetching prescription for appointment ID: %d", appointmentID)

	// First verify access
	var authorized bool
	var query string
	if isDoctor {
		// Modified query to ensure doctor can only see their own prescriptions
		query = `SELECT EXISTS(
            SELECT 1 FROM appointments 
            WHERE id = ? 
            AND doctor_id = ?
            AND doctor_id IN (
                SELECT doctor_id 
                FROM appointments 
                WHERE id = ?
            ))`
		err := db.QueryRow(query, appointmentID, userID, appointmentID).Scan(&authorized)
		if err != nil {
			return nil, err
		}
	} else {
		// For patients, keep the existing logic
		query = `SELECT EXISTS(
            SELECT 1 FROM appointments 
            WHERE id = ? AND patient_id = ?)`
		err := db.QueryRow(query, appointmentID, userID).Scan(&authorized)
		if err != nil {
			return nil, err
		}
	}

	if !authorized {
		return nil, fmt.Errorf("unauthorized access")
	}

	// Use the stored procedure for secure access
	rows, err := db.Query("CALL GetPrescriptionByAppointmentSecure(?, ?, ?)",
		appointmentID, userID, isDoctor)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prescription PrescriptionResponse
	var medications []Medication
	var createdAtStr string // Use a string to temporarily store the created_at value

	for rows.Next() {
		var med Medication
		var medicine, frequency sql.NullString // Use sql.NullString to handle NULL values

		err := rows.Scan(
			&prescription.ID,
			&prescription.DoctorID,
			&prescription.PatientID,
			&prescription.AppointmentID,
			&prescription.VisitType,
			&createdAtStr, // Scan into a string
			&prescription.Instructions,
			&prescription.PatientName,
			&medicine,  // Use sql.NullString
			&frequency, // Use sql.NullString
		)
		if err != nil {
			return nil, err
		}

		// Parse the created_at string into a time.Time object
		createdAt, err := time.Parse("2006-01-02", createdAtStr) // Use "2006-01-02" for YYYY-MM-DD format
		if err != nil {
			return nil, fmt.Errorf("error parsing created_at: %v", err)
		}

		// Convert Gregorian date to Hijri date
		prescription.CreatedAt = utils.GregorianToSolar(createdAt)

		// Handle NULL values for medicine and frequency
		med.Medicine = medicine.String
		med.Frequency = frequency.String

		medications = append(medications, med)
	}

	prescription.Medications = medications

	if prescription.VisitType == "online" {
		prescription.VisitType = "آنلاین"
	}

	return &prescription, nil
}

func GetPatientPrescriptions(db *sql.DB, patientID, userID int, isDoctor bool) ([]PrescriptionResponse, error) {
	// log.Printf("Fetching prescriptions for patient ID: %d", patientID)

	// First verify access
	if !isDoctor && patientID != userID {
		return nil, fmt.Errorf("unauthorized access")
	}

	if isDoctor {
		// Verify doctor has treated this patient
		var authorized bool
		query := `SELECT EXISTS(
            SELECT 1 FROM appointments 
            WHERE doctor_id = ? AND patient_id = ?)`
		err := db.QueryRow(query, userID, patientID).Scan(&authorized)
		if err != nil {
			return nil, err
		}
		if !authorized {
			return nil, fmt.Errorf("unauthorized access")
		}
	}

	// Use the stored procedure for secure access
	rows, err := db.Query("CALL GetPatientPrescriptionsSecure(?, ?, ?)",
		patientID, userID, isDoctor)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prescriptions []PrescriptionResponse
	for rows.Next() {
		var prescription PrescriptionResponse
		var createdAtStr string // Use a string to temporarily store the created_at value

		err := rows.Scan(
			&prescription.ID,
			&prescription.DoctorID,
			&prescription.PatientID,
			&prescription.AppointmentID,
			&prescription.VisitType,
			&createdAtStr, // Scan into a string
			&prescription.Instructions,
			&prescription.DoctorName,
		)
		if err != nil {
			return nil, err
		}

		// Parse the created_at string into a time.Time object
		createdAt, err := time.Parse("2006-01-02", createdAtStr) // Use "2006-01-02" for YYYY-MM-DD format
		if err != nil {
			return nil, fmt.Errorf("error parsing created_at: %v", err)
		}

		// Convert Gregorian date to Hijri date
		prescription.CreatedAt = utils.GregorianToSolar(createdAt)

		// Get medications
		medQuery := `SELECT medicine, frequency FROM medications WHERE prescription_id = ?`
		medRows, err := db.Query(medQuery, prescription.ID)
		if err != nil {
			return nil, err
		}
		defer medRows.Close()

		var medications []Medication
		for medRows.Next() {
			var med Medication
			if err := medRows.Scan(&med.Medicine, &med.Frequency); err != nil {
				return nil, err
			}
			medications = append(medications, med)
		}
		prescription.Medications = medications

		if prescription.VisitType == "online" {
			prescription.VisitType = "آنلاین"
		}

		prescriptions = append(prescriptions, prescription)
	}

	return prescriptions, nil
}

func UpdatePrescriptionDB(db *sql.DB, prescription *Prescription) error {
	// log.Printf("Starting update for prescription ID: %d", prescription.ID)

	tx, err := db.Begin()
	if err != nil {
		// log.Printf("Error starting transaction: %v", err)
		return err
	}
	defer tx.Rollback()

	// Update prescription instructions
	_, err = tx.Exec(`
        UPDATE prescriptions 
        SET instructions = ?
        WHERE id = ?`,
		prescription.Instructions,
		prescription.ID)
	if err != nil {
		// log.Printf("Error updating prescription instructions: %v", err)
		return err
	}

	// Delete existing medications
	_, err = tx.Exec("DELETE FROM medications WHERE prescription_id = ?", prescription.ID)
	if err != nil {
		// log.Printf("Error deleting existing medications: %v", err)
		return err
	}

	// Insert new medications
	for _, med := range prescription.Medications {
		_, err := tx.Exec(`
            INSERT INTO medications (prescription_id, medicine, frequency)
            VALUES (?, ?, ?)`,
			prescription.ID,
			med.Medicine,
			med.Frequency)
		if err != nil {
			// log.Printf("Error inserting medication: %v", err)
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		// log.Printf("Error committing transaction: %v", err)
		return err
	}

	// log.Printf("Prescription updated successfully")
	return nil
}

func GetPrescriptionWithName(db *sql.DB, prescriptionID int, isDoctor bool) (*PrescriptionResponse, error) {
	prescription := &PrescriptionResponse{}

	// Join with appropriate table to get the name based on role
	var query string
	if isDoctor {
		query = `
            SELECT 
                p.id, 
                a.doctor_id,
                a.patient_id,
                p.appointment_id,
                a.visit_type,
                p.created_at,
                p.instructions,
                CONCAT(pt.first_name, ' ', pt.last_name) as patient_name
            FROM prescriptions p
            JOIN appointments a ON p.appointment_id = a.id
            JOIN patients pt ON a.patient_id = pt.id
            WHERE p.id = ?`
	} else {
		query = `
            SELECT 
                p.id, 
                a.doctor_id,
                a.patient_id,
                p.appointment_id,
                a.visit_type,
                p.created_at,
                p.instructions,
                CONCAT(d.first_name, ' ', d.last_name) as doctor_name
            FROM prescriptions p
            JOIN appointments a ON p.appointment_id = a.id
            JOIN doctors d ON a.doctor_id = d.id
            WHERE p.id = ?`
	}

	var createdAt time.Time
	err := db.QueryRow(query, prescriptionID).Scan(
		&prescription.ID,
		&prescription.DoctorID,
		&prescription.PatientID,
		&prescription.AppointmentID,
		&prescription.VisitType,
		&createdAt,
		&prescription.Instructions,
		&prescription.PatientName, // Use PatientName or DoctorName
	)
	if err != nil {
		return nil, err
	}

	// Convert Gregorian date to Hijri date
	prescription.CreatedAt = utils.GregorianToSolar(createdAt)

	// Get medications (same as before)
	medQuery := `SELECT medicine, frequency FROM medications WHERE prescription_id = ?`
	medRows, err := db.Query(medQuery, prescription.ID)
	if err != nil {
		return nil, err
	}
	defer medRows.Close()

	var medications []Medication
	for medRows.Next() {
		var med Medication
		if err := medRows.Scan(&med.Medicine, &med.Frequency); err != nil {
			return nil, err
		}
		medications = append(medications, med)
	}
	prescription.Medications = medications

	if prescription.VisitType == "online" {
		prescription.VisitType = "آنلاین"
	}

	return prescription, nil
}

func GetPrescriptionsByDoctor(db *sql.DB, doctorID int) ([]PrescriptionResponse, error) {
	// log.Printf("Fetching prescriptions for doctor ID: %d", doctorID)

	// Query to fetch prescriptions written by the doctor
	query := `
        SELECT 
            p.id,
            p.appointment_id,
            a.doctor_id,
            a.patient_id,
            p.instructions,
            DATE_FORMAT(p.created_at, '%Y-%m-%d') as created_at, -- Format as string
            a.visit_type,
            CONCAT(pt.first_name, ' ', pt.last_name) as patient_name
        FROM prescriptions p
        JOIN appointments a ON p.appointment_id = a.id
        JOIN patients pt ON a.patient_id = pt.id
        WHERE a.doctor_id = ?
        ORDER BY p.created_at DESC`

	rows, err := db.Query(query, doctorID)
	if err != nil {
		// log.Printf("Error querying prescriptions: %v", err)
		return nil, err
	}
	defer rows.Close()

	var prescriptions []PrescriptionResponse
	for rows.Next() {
		var p PrescriptionResponse
		var createdAtStr string // Use a string to temporarily store the created_at value

		err := rows.Scan(
			&p.ID,
			&p.AppointmentID,
			&p.DoctorID,
			&p.PatientID,
			&p.Instructions,
			&createdAtStr, // Scan into a string
			&p.VisitType,
			&p.PatientName,
		)
		if err != nil {
			// log.Printf("Error scanning row: %v", err)
			return nil, err
		}

		// Parse the created_at string into a time.Time object
		createdAt, err := time.Parse("2006-01-02", createdAtStr) // Use "2006-01-02" for YYYY-MM-DD format
		if err != nil {
			return nil, fmt.Errorf("error parsing created_at: %v", err)
		}

		// Convert Gregorian date to Hijri date
		p.CreatedAt = utils.GregorianToSolar(createdAt)

		// Get medications for this prescription
		medRows, err := db.Query(`
            SELECT medicine, frequency 
            FROM medications 
            WHERE prescription_id = ?`, p.ID)
		if err != nil {
			// log.Printf("Error fetching medications: %v", err)
			return nil, err
		}
		defer medRows.Close()

		var medications []Medication
		for medRows.Next() {
			var med Medication
			if err := medRows.Scan(&med.Medicine, &med.Frequency); err != nil {
				// log.Printf("Error scanning medication: %v", err)
				return nil, err
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
		return nil, err
	}

	return prescriptions, nil
}

// GetPrescriptionsByPatientNameAndDate retrieves prescriptions based on patient name and/or date
func GetPrescriptionsByPatientNameAndDate(db *sql.DB, patientName string, date time.Time) ([]PrescriptionWithDetails, error) {
	// Base query with full name search
	query := `
        SELECT 
            p.id, 
            a.patient_id, 
            a.doctor_id, 
            p.appointment_id, -- Added field
            p.instructions, 
            DATE_FORMAT(p.created_at, '%Y-%m-%d %H:%i:%s') as created_at,
            a.visit_type, -- Added field
            CONCAT(pt.first_name, ' ', pt.last_name) as patient_name -- Added field
        FROM prescriptions p
        JOIN appointments a ON p.appointment_id = a.id
        JOIN patients pt ON a.patient_id = pt.id
        WHERE CONCAT(pt.first_name, ' ', pt.last_name) LIKE ?
    `
	args := []interface{}{"%" + patientName + "%"}

	// Add date filter if a date is provided
	if !date.IsZero() {
		query += " AND DATE(p.created_at) = ?"
		args = append(args, date.Format("2006-01-02"))
	}

	// Execute the query
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// log.Printf("Query: %s, Args: %v", query, args)

	var prescriptions []PrescriptionWithDetails
	for rows.Next() {
		var p PrescriptionWithDetails
		var createdAtStr string // Temporary variable to hold the string representation of the timestamp

		err := rows.Scan(
			&p.ID,
			&p.PatientID,
			&p.DoctorID,
			&p.AppointmentID, // Added field
			&p.Instructions,
			&createdAtStr,  // Scan into a string first
			&p.VisitType,   // Added field
			&p.PatientName, // Added field
		)
		if err != nil {
			return nil, err
		}

		// Parse the string into a time.Time object
		createdAt, err := time.Parse("2006-01-02 15:04:05", createdAtStr)
		if err != nil {
			return nil, fmt.Errorf("error parsing created_at: %v", err)
		}

		// Store the Gregorian date in the response
		p.CreatedAt = createdAt.Format("2006-01-02")
		prescriptions = append(prescriptions, p)
	}

	return prescriptions, nil
}
