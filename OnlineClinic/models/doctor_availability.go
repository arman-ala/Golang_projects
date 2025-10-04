package models

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"onlineClinic/utils"
	"time"
)

type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type AvailabilityRequest struct {
	Type       string      `json:"type"` // 'online' or 'in-person'
	TimesRange []TimeRange `json:"timesRange"`
	DatesRange []TimeRange `json:"datesRange"`
}

// AvailabilitySlot represents a doctor's available time slot
type AvailabilitySlot struct {
	ID        int
	DoctorID  int
	StartTime time.Time
	EndTime   time.Time
	Type      string
	Date      string // Hijri date (yyyy-MM-dd) for front-end
	Time      string // HH:mm format for front-end
}

type AvailabilityDay struct {
	Date  string             `json:"date"`
	Times []AvailabilitySlot `json:"times"`
}

// Custom time format for JSON marshaling
type CustomTime time.Time

func (ct CustomTime) MarshalJSON() ([]byte, error) {
	t := time.Time(ct)
	stamp := fmt.Sprintf("\"%s\"", t.Format("2006-01-02 15:04:05"))
	return []byte(stamp), nil
}

func (tr *TimeRange) UnmarshalJSON(data []byte) error {
	log.Printf("Attempting to unmarshal TimeRange data: %s", string(data))

	var rawTimeRange struct {
		Start string `json:"start"`
		End   string `json:"end"`
	}

	if err := json.Unmarshal(data, &rawTimeRange); err != nil {
		log.Printf("Error unmarshaling TimeRange: %v", err)
		return fmt.Errorf("invalid time range format: %v", err)
	}

	// Define the expected time and date formats
	timeFormat := "15:04"      // For times (e.g., "10:00")
	dateFormat := "2006-01-02" // For dates (e.g., "1413-10-01")

	// Try parsing as time first
	startTime, err := time.Parse(timeFormat, rawTimeRange.Start)
	if err == nil {
		// If parsing as time succeeds, set the date to today
		now := time.Now()
		startTime = time.Date(now.Year(), now.Month(), now.Day(), startTime.Hour(), startTime.Minute(), 0, 0, time.UTC)
	} else {
		// If parsing as time fails, try parsing as date
		startTime, err = time.Parse(dateFormat, rawTimeRange.Start)
		if err != nil {
			// If parsing as date fails, try parsing as Hijri date
			startTime, err = utils.SolarToGregorian(rawTimeRange.Start)
			if err != nil {
				log.Printf("Failed to parse start time/date: %v", err)
				return fmt.Errorf("invalid start time/date format: %v", err)
			}
		}
	}

	// Try parsing as time first
	endTime, err := time.Parse(timeFormat, rawTimeRange.End)
	if err == nil {
		// If parsing as time succeeds, set the date to today
		now := time.Now()
		endTime = time.Date(now.Year(), now.Month(), now.Day(), endTime.Hour(), endTime.Minute(), 0, 0, time.UTC)
	} else {
		// If parsing as time fails, try parsing as date
		endTime, err = time.Parse(dateFormat, rawTimeRange.End)
		if err != nil {
			// If parsing as date fails, try parsing as Hijri date
			endTime, err = utils.SolarToGregorian(rawTimeRange.End)
			if err != nil {
				log.Printf("Failed to parse end time/date: %v", err)
				return fmt.Errorf("invalid end time/date format: %v", err)
			}
		}
	}

	// Assign parsed times/dates to the struct
	tr.Start = startTime
	tr.End = endTime

	log.Printf("Successfully parsed TimeRange - Start: %v, End: %v", tr.Start, tr.End)
	return nil
}

// Modified DoctorAvailability struct to use CustomTime
type DoctorAvailability struct {
	ID        int        `json:"id"`
	DoctorID  int        `json:"doctorId"`
	StartTime CustomTime `json:"startTime"`
	EndTime   CustomTime `json:"endTime"`
	Type      string     `json:"type"`
}

var ErrAvailabilityNotFound = errors.New("availability slot not found")

// SetDoctorAvailability splits time ranges into 15-minute sessions and inserts them into the database
func SetDoctorAvailability(db *sql.DB, doctorID int, req *AvailabilityRequest) error {
	log.Printf("Starting SetDoctorAvailability for doctorID: %d", doctorID)

	// Validate request
	if err := req.Validate(); err != nil {
		log.Printf("Validation failed for doctorID %d: %v", doctorID, err)
		return fmt.Errorf("validation error: %v", err)
	}

	tx, err := db.Begin()
	if err != nil {
		log.Printf("Failed to begin transaction for doctorID %d: %v", doctorID, err)
		return err
	}
	defer func() {
		if err != nil {
			log.Printf("Rolling back transaction for doctorID %d due to error: %v", doctorID, err)
			tx.Rollback()
		}
	}()

	log.Printf("Processing availability slots for doctorID %d - Dates: %d, Times: %d", doctorID, len(req.DatesRange), len(req.TimesRange))

	stmt, err := tx.Prepare(`INSERT INTO doctor_availability (doctor_id, start_time, end_time, type) VALUES (?, ?, ?, ?)`)
	if err != nil {
		log.Printf("Failed to prepare statement for doctorID %d: %v", doctorID, err)
		return err
	}
	defer stmt.Close()

	// Iterate over each date range
	for _, dateRange := range req.DatesRange {
		// Convert Hijri start and end dates to Gregorian
		startDate, err := utils.SolarToGregorian(dateRange.Start.Format("2006-01-02"))
		if err != nil {
			log.Printf("Error converting Hijri start date to Gregorian: %v", err)
			return fmt.Errorf("invalid start date format: %v", err)
		}

		endDate, err := utils.SolarToGregorian(dateRange.End.Format("2006-01-02"))
		if err != nil {
			log.Printf("Error converting Hijri end date to Gregorian: %v", err)
			return fmt.Errorf("invalid end date format: %v", err)
		}

		log.Printf("Processing date range: %v to %v", startDate, endDate)

		// Iterate over each day in the date range
		for currentDate := startDate; !currentDate.After(endDate); currentDate = currentDate.AddDate(0, 0, 1) {
			log.Printf("Processing date: %v", currentDate)

			// Iterate over each time range
			for _, timeRange := range req.TimesRange {
				startTime := timeRange.Start
				endTime := timeRange.End

				log.Printf("Processing time range: %v to %v", startTime, endTime)

				// Split the time range into 15-minute sessions
				currentTime := startTime
				for currentTime.Before(endTime) {
					// Calculate the end of the 15-minute session
					sessionEnd := currentTime.Add(15 * time.Minute)

					// If the next session would exceed the end time, adjust the session end to match the end time
					if sessionEnd.After(endTime) {
						sessionEnd = endTime
					}

					// Combine date and time for the session
					sessionStart := time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(),
						currentTime.Hour(), currentTime.Minute(), 0, 0, time.UTC)
					sessionEndTime := time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(),
						sessionEnd.Hour(), sessionEnd.Minute(), 0, 0, time.UTC)

					log.Printf("Inserting slot: %v to %v", sessionStart, sessionEndTime)

					// Insert the availability slot (in Gregorian format)
					_, err = stmt.Exec(doctorID, sessionStart, sessionEndTime, req.Type)
					if err != nil {
						log.Printf("Failed to insert availability slot: %v", err)
						return fmt.Errorf("failed to insert availability slot: %v", err)
					}

					// Move to the next 15-minute session
					currentTime = sessionEnd

					// If the current session end matches the end time, break the loop
					if sessionEnd.Equal(endTime) {
						break
					}
				}
			}
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction for doctorID %d: %v", doctorID, err)
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	log.Printf("Successfully completed SetDoctorAvailability for doctorID %d", doctorID)
	return nil
}

func (ar *AvailabilityRequest) Validate() error {
	log.Printf("Validating AvailabilityRequest - Type: %s, TimesRange: %d items, DatesRange: %d items", ar.Type, len(ar.TimesRange), len(ar.DatesRange))

	// Validate type
	if ar.Type != "online" && ar.Type != "in-person" {
		log.Printf("Invalid type provided: %s", ar.Type)
		return errors.New("type must be either 'online' or 'in-person'")
	}

	// Validate time ranges
	if len(ar.TimesRange) == 0 {
		log.Print("TimesRange is empty")
		return errors.New("timesRange cannot be empty")
	}

	// Validate date ranges
	if len(ar.DatesRange) == 0 {
		log.Print("DatesRange is empty")
		return errors.New("datesRange cannot be empty")
	}

	// Validate individual time ranges
	for i, tr := range ar.TimesRange {
		log.Printf("Validating TimeRange[%d] - Start: %v, End: %v", i, tr.Start, tr.End)
		if tr.Start.After(tr.End) {
			log.Printf("Invalid time range at index %d: end time must be after start time", i)
			return fmt.Errorf("invalid time range at index %d: end time must be after start time", i)
		}
	}

	// Validate individual date ranges
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	for i, dr := range ar.DatesRange {
		log.Printf("Validating DateRange[%d] - Start: %v, End: %v", i, dr.Start, dr.End)

		// Check if end is before start (invalid)
		if dr.Start.After(dr.End) {
			log.Printf("Invalid date range at index %d: end date must be after or equal to start date", i)
			return fmt.Errorf("invalid date range at index %d: end date must be after or equal to start date", i)
		}

		// Check if date range is in the past (convert Persian dates to Gregorian first)
		if dr.Start.Format("2006-01-02") != "" {
			gregDate, err := utils.SolarToGregorian(dr.Start.Format("2006-01-02"))
			if err != nil {
				log.Printf("Error converting Persian date: %v", err)
				return fmt.Errorf("invalid date format at index %d", i)
			}
			if gregDate.Before(today) {
				log.Printf("Invalid date range at index %d: date range is in the past", i)
				return fmt.Errorf("invalid date range at index %d: date range is in the past", i)
			}
		} else if dr.End.Before(today) {
			log.Printf("Invalid date range at index %d: date range is in the past", i)
			return fmt.Errorf("invalid date range at index %d: date range is in the past", i)
		}

		// If date range includes today, validate time slots
		if (dr.Start.Before(today) || dr.Start.Equal(today)) && (dr.End.After(today) || dr.End.Equal(today)) {
			currentMinutes := now.Hour()*60 + now.Minute()
			for _, tr := range ar.TimesRange {
				if tr.Start.Hour()*60+tr.Start.Minute() <= currentMinutes {
					log.Printf("Invalid time range at index %d: time slot is in the past", i)
					return fmt.Errorf("invalid time range at index %d: time slot is in the past", i)
				}
			}
		}
	}

	return nil
}

func GetDoctorAvailability(db *sql.DB, doctorID int, visitType string) ([]AvailabilitySlot, error) {
	log.Printf("Getting availability slots for doctor ID: %d, visit type: %s", doctorID, visitType)

	// Use server time (Tehran/Iran) directly
	now := time.Now()
	log.Printf("Current server time (Tehran): %v", now.Format("2006-01-02 15:04:05"))

	// Filter slots starting after current time and up to 2 years
	endDate := now.AddDate(2, 0, 0) // Fetch slots for the next 2 years
	log.Printf("Querying slots after %v until %v", now.Format("2006-01-02 15:04:05"), endDate.Format("2006-01-02 15:04:05"))

	// Query to fetch availability slots - we'll filter by time in Go code
	query := `
		SELECT id, doctor_id, start_time, end_time, type
		FROM doctor_availability
		WHERE doctor_id = ? AND type = ? AND end_time <= ?
		ORDER BY start_time ASC
	`
	rows, err := db.Query(query, doctorID, visitType, endDate)
	if err != nil {
		log.Printf("Error executing query: %v", err)
		return nil, fmt.Errorf("database query error: %v", err)
	}
	defer rows.Close()

	log.Printf("Query: %s, Params: doctorID=%d, visitType=%s, endDate=%v", query, doctorID, visitType, endDate)

	var slots []AvailabilitySlot
	for rows.Next() {
		var slot AvailabilitySlot
		var startTimeStr, endTimeStr string

		if err := rows.Scan(
			&slot.ID,
			&slot.DoctorID,
			&startTimeStr,
			&endTimeStr,
			&slot.Type,
		); err != nil {
			return nil, fmt.Errorf("error scanning row: %v", err)
		}

		// Parse times in Tehran time
		startTime, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			return nil, fmt.Errorf("error parsing start_time: %v", err)
		}

		endTime, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			return nil, fmt.Errorf("error parsing end_time: %v", err)
		}

		// Only filter by date
		today := now
		slotDate := startTime

		if slotDate.Before(today) {
			continue
		} else {

			// Set parsed times
			slot.StartTime = startTime
			slot.EndTime = endTime
			slot.Time = startTime.Format("15:04") // Format time as HH:mm

			slots = append(slots, slot)
		}
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error after iterating rows: %v", err)
		return nil, fmt.Errorf("error after iterating rows: %v", err)
	}

	return slots, nil
}
