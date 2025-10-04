// models/doctor.go
package models

import (
	"database/sql"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// SQLExecutor is an interface that both *sql.DB and *sql.Tx satisfy
type SQLExecutor interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}

type Doctor struct {
	ID                 int     `json:"id"`
	FirstName          string  `json:"firstName"`
	LastName           string  `json:"lastName"`
	NationalCode       string  `json:"nationalCode"`
	Gender             string  `json:"gender"`
	PhoneNumber        string  `json:"phoneNumber"`
	Password           string  `json:"password"`
	Age                *int    `json:"age,omitempty"`
	Education          *string `json:"education,omitempty"`
	Address            *string `json:"address,omitempty"`
	ProfilePhotoPath   *string `json:"image,omitempty"`
	MedicalCouncilCode *string `json:"medicalCouncilCode,omitempty"` // Added new field
}

type DoctorPrescription struct {
	ID            int          `json:"id"`
	PatientID     int          `json:"patientId"`
	DoctorID      int          `json:"doctorId"`
	AppointmentID int          `json:"appointmentId"`
	PatientName   string       `json:"patientName"`
	Instructions  string       `json:"instructions"`
	Medications   []Medication `json:"medications"`
	CreatedAt     time.Time    `json:"createdAt"`
}

type DoctorSearchResult struct {
	ID        int    `json:"id,string"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Image     string `json:"image,omitempty"`
	Address   string `json:"address,omitempty"`
}

func SearchDoctors(db *sql.DB, searchTerm string) ([]DoctorSearchResult, error) {
	query := `
        SELECT id, first_name, last_name, profile_photo_path, address
        FROM doctors
        WHERE CONCAT(first_name, ' ', last_name) LIKE ?
        OR first_name LIKE ?
        OR last_name LIKE ?
    `

	// Add wildcards for partial matching
	searchPattern := "%" + searchTerm + "%"

	rows, err := db.Query(query, searchPattern, searchPattern, searchPattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []DoctorSearchResult
	for rows.Next() {
		var result DoctorSearchResult
		var image, address sql.NullString

		err := rows.Scan(
			&result.ID,
			&result.FirstName,
			&result.LastName,
			&image,
			&address,
		)
		if err != nil {
			return nil, err
		}

		if image.Valid {
			result.Image = image.String
		}
		if address.Valid {
			result.Address = address.String
		}

		results = append(results, result)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// Add this function to the existing file
func GetDoctorPrescriptions(db *sql.DB, doctorID int) ([]DoctorPrescription, error) {
	query := `
        SELECT 
            p.id,
            p.patient_id,
            p.doctor_id,
            p.appointment_id,
            p.instructions,
            p.created_at,
            CONCAT(pt.first_name, ' ', pt.last_name) as patient_name
        FROM prescriptions p
        JOIN patients pt ON p.patient_id = pt.id
        WHERE p.doctor_id = ?
        ORDER BY p.created_at DESC`

	rows, err := db.Query(query, doctorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prescriptions []DoctorPrescription
	for rows.Next() {
		var prescription DoctorPrescription
		err := rows.Scan(
			&prescription.ID,
			&prescription.PatientID,
			&prescription.DoctorID,
			&prescription.AppointmentID,
			&prescription.Instructions,
			&prescription.CreatedAt,
			&prescription.PatientName,
		)
		if err != nil {
			return nil, err
		}

		// Get medications for this prescription
		medQuery := `
            SELECT medicine, frequency 
            FROM medications 
            WHERE prescription_id = ?`

		medRows, err := db.Query(medQuery, prescription.ID)
		if err != nil {
			return nil, err
		}
		defer medRows.Close()

		var medications []Medication
		for medRows.Next() {
			var med Medication
			err := medRows.Scan(&med.Medicine, &med.Frequency)
			if err != nil {
				return nil, err
			}
			medications = append(medications, med)
		}
		prescription.Medications = medications

		prescriptions = append(prescriptions, prescription)
	}

	return prescriptions, nil
}

// CreateDoctor creates a new doctor record in the database
func CreateDoctor(db *sql.DB, doctor *Doctor) error {
	_, err := db.Exec(`CALL AddDoctor(?, ?, ?, ?, ?, ?, ?)`,
		doctor.FirstName,
		doctor.LastName,
		doctor.NationalCode,
		doctor.Gender,
		doctor.PhoneNumber,
		doctor.Password,
		doctor.MedicalCouncilCode, // Added parameter
	)
	return err
}

// GetDoctorById retrieves a doctor by their ID
func GetDoctorById(db *sql.DB, id int) (*Doctor, error) {
	var doctor Doctor
	var age sql.NullInt64
	var education, address, profilePhotoPath, medicalCouncilCode sql.NullString

	query := `SELECT id, first_name, last_name, national_code, gender, 
        phone_number, password, age, education, address,
        profile_photo_path, medical_council_code 
        FROM doctors WHERE id = ?`

	err := db.QueryRow(query, id).Scan(
		&doctor.ID,
		&doctor.FirstName,
		&doctor.LastName,
		&doctor.NationalCode,
		&doctor.Gender,
		&doctor.PhoneNumber,
		&doctor.Password,
		&age,
		&education,
		&address,
		&profilePhotoPath,
		&medicalCouncilCode,
	)
	if err != nil {
		return nil, err
	}

	// Convert nullable fields
	if age.Valid {
		ageValue := int(age.Int64)
		doctor.Age = &ageValue
	}
	if education.Valid {
		doctor.Education = &education.String
	}
	if address.Valid {
		doctor.Address = &address.String
	}
	if profilePhotoPath.Valid {
		doctor.ProfilePhotoPath = &profilePhotoPath.String
	}
	if medicalCouncilCode.Valid {
		doctor.MedicalCouncilCode = &medicalCouncilCode.String
	}

	return &doctor, nil
}

func UpdateDoctor(executor SQLExecutor, doctor *Doctor) error {
	_, err := executor.Exec(`CALL UpdateDoctor(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		doctor.ID,
		doctor.FirstName,
		doctor.LastName,
		doctor.NationalCode,
		doctor.Gender,
		doctor.PhoneNumber,
		doctor.Age,
		doctor.Education,
		doctor.Address,
		doctor.ProfilePhotoPath,
		doctor.MedicalCouncilCode, // Added parameter
	)
	return err
}

func UpdateDoctorPassword(executor SQLExecutor, doctorID int, hashedPassword string) error {
	// Hash the new password before storing it
	hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(hashedPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %v", err)
	}

	// Update the password in the database
	_, err = executor.Exec(`CALL UpdateDoctorPassword(?, ?)`, doctorID, string(hashedPasswordBytes))
	if err != nil {
		return fmt.Errorf("failed to update password: %v", err)
	}

	return nil
}

// DeleteDoctor removes a doctor from the database
func DeleteDoctor(db *sql.DB, id int) error {
	query := "DELETE FROM doctors WHERE id = ?"
	_, err := db.Exec(query, id)
	return err
}

// GetAllDoctors retrieves all doctors from the database
func GetAllDoctors(db *sql.DB) ([]Doctor, error) {
	var doctors []Doctor

	// Query to retrieve all doctors, including ProfilePhotoPath and Address
	query := `
        SELECT id, first_name, last_name, national_code, gender, 
               phone_number, password, age, education, address,
               profile_photo_path 
        FROM doctors`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Iterate through the rows and populate the Doctor struct
	for rows.Next() {
		var doctor Doctor
		var age sql.NullInt64
		var education, address, profilePhotoPath sql.NullString

		err := rows.Scan(
			&doctor.ID,
			&doctor.FirstName,
			&doctor.LastName,
			&doctor.NationalCode,
			&doctor.Gender,
			&doctor.PhoneNumber,
			&doctor.Password,
			&age,
			&education,
			&address,
			&profilePhotoPath,
		)
		if err != nil {
			return nil, err
		}

		// Handle nullable fields
		if age.Valid {
			ageValue := int(age.Int64)
			doctor.Age = &ageValue
		}
		if education.Valid {
			doctor.Education = &education.String
		}
		if address.Valid {
			doctor.Address = &address.String
		}
		if profilePhotoPath.Valid {
			doctor.ProfilePhotoPath = &profilePhotoPath.String
		}

		doctors = append(doctors, doctor)
	}

	// Check for any errors during iteration
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return doctors, nil
}

// Remove GetDoctorsBySpecialization as specialization field is removed

// DeleteDoctorAvailability removes an availability slot for a specific doctor
func DeleteDoctorAvailability(db *sql.DB, slotID, doctorID int) error {
	result, err := db.Exec(
		"DELETE FROM doctor_availability WHERE id = ? AND doctor_id = ? AND start_time > NOW()",
		slotID,
		doctorID,
	)
	if err != nil {
		return fmt.Errorf("database error: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return ErrAvailabilityNotFound
	}

	return nil
}

func DeleteDoctorPhoto(db *sql.DB, id int) error {
	query := "UPDATE doctors SET profile_photo_path = NULL WHERE id = ?"
	_, err := db.Exec(query, id)
	return err
}
