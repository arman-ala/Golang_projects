// models/patient.go
package models

import (
	"database/sql"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type Patient struct {
	ID               int     `json:"id,string"`
	FirstName        string  `json:"firstName"`
	LastName         string  `json:"lastName"`
	NationalCode     string  `json:"nationalCode"`
	Gender           string  `json:"gender"`
	PhoneNumber      string  `json:"phoneNumber"`
	Password         string  `json:"password"`
	Age              *int    `json:"age,omitempty"`              // Use pointer for nullable age
	Job              *string `json:"job,omitempty"`              // Use pointer for nullable job
	Education        *string `json:"education,omitempty"`        // Use pointer for nullable education
	Address          *string `json:"address,omitempty"`          // Use pointer for nullable address
	ProfilePhotoPath *string `json:"profilePhotoPath,omitempty"` // Use pointer for nullable profile_photo_path
}

func CreatePatient(db *sql.DB, patient *Patient) error {
	_, err := db.Exec(`CALL AddPatient(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		patient.FirstName,
		patient.LastName,
		patient.NationalCode,
		patient.Gender,
		patient.PhoneNumber,
		patient.Password,
		patient.Age,
		patient.Job,
		patient.Education,
		patient.Address,
		patient.ProfilePhotoPath,
	)
	return err
}

func UpdatePatient(executor SQLExecutor, patient *Patient) error {
	_, err := executor.Exec(`CALL UpdatePatient(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		patient.ID,
		patient.FirstName,
		patient.LastName,
		patient.NationalCode,
		patient.Gender,
		patient.PhoneNumber,
		patient.Age,
		patient.Job,
		patient.Education,
		patient.Address,
		patient.ProfilePhotoPath,
	)
	return err
}

func UpdatePatientPassword(executor SQLExecutor, patientID int, hashedPassword string) error {
	// Hash the new password before storing it
	hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(hashedPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %v", err)
	}

	// Update the password in the database
	_, err = executor.Exec(`CALL UpdatePatientPassword(?, ?)`, patientID, string(hashedPasswordBytes))
	if err != nil {
		return fmt.Errorf("failed to update password: %v", err)
	}

	return nil
}

func GetAllPatients(db *sql.DB) ([]Patient, error) {
	var patients []Patient
	query := `SELECT id, first_name, last_name, national_code, gender, 
        phone_number, password, age, job, education, address,
        profile_photo_path FROM patients`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var patient Patient
		err := rows.Scan(
			&patient.ID,
			&patient.FirstName,
			&patient.LastName,
			&patient.NationalCode,
			&patient.Gender,
			&patient.PhoneNumber,
			&patient.Password,
			&patient.Age,
			&patient.Job,
			&patient.Education,
			&patient.Address,
			&patient.ProfilePhotoPath,
		)
		if err != nil {
			return nil, err
		}
		patients = append(patients, patient)
	}
	return patients, nil
}

func GetPatientByPhone(db *sql.DB, phoneNumber string) (*Patient, error) {
	var patient Patient
	query := `SELECT id, first_name, last_name, national_code, gender, 
        phone_number, password, age, job, education, address,
        profile_photo_path 
        FROM patients WHERE phone_number = ?`

	err := db.QueryRow(query, phoneNumber).Scan(
		&patient.ID,
		&patient.FirstName,
		&patient.LastName,
		&patient.NationalCode,
		&patient.Gender,
		&patient.PhoneNumber,
		&patient.Password,
		&patient.Age,
		&patient.Job,
		&patient.Education,
		&patient.Address,
		&patient.ProfilePhotoPath,
	)
	if err != nil {
		return nil, err
	}
	return &patient, nil
}

func DeletePatient(db *sql.DB, id int) error {
	query := "DELETE FROM patients WHERE id = ?"
	_, err := db.Exec(query, id)
	return err
}

func DeletePatientPhoto(db *sql.DB, id int) error {
	query := "UPDATE patients SET profile_photo_path = NULL WHERE id = ?"
	_, err := db.Exec(query, id)
	return err
}

func GetPatientById(db *sql.DB, id int) (*Patient, error) {
	var patient Patient
	var age sql.NullInt64
	var job, education, address, profilePhotoPath sql.NullString

	query := `SELECT id, first_name, last_name, national_code, gender, 
        phone_number, password, age, job, education, address,
        profile_photo_path 
        FROM patients WHERE id = ?`

	err := db.QueryRow(query, id).Scan(
		&patient.ID,
		&patient.FirstName,
		&patient.LastName,
		&patient.NationalCode,
		&patient.Gender,
		&patient.PhoneNumber,
		&patient.Password,
		&age,
		&job,
		&education,
		&address,
		&profilePhotoPath,
	)
	if err != nil {
		return nil, err
	}

	// Convert nullable fields to pointers
	if age.Valid {
		ageValue := int(age.Int64)
		patient.Age = &ageValue
	}
	if job.Valid {
		patient.Job = &job.String
	}
	if education.Valid {
		patient.Education = &education.String
	}
	if address.Valid {
		patient.Address = &address.String
	}
	if profilePhotoPath.Valid {
		patient.ProfilePhotoPath = &profilePhotoPath.String
	}

	return &patient, nil
}
