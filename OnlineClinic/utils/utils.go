// utils/utils.go
package utils

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	ptime "github.com/yaa110/go-persian-calendar"
)

// GenerateRandomString generates a random string of specified length
func GenerateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// ValidateEmail checks if the email format is valid
func ValidateEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	match, _ := regexp.MatchString(pattern, email)
	return match
}

// SanitizeFileName removes potentially dangerous characters from filenames
func SanitizeFileName(filename string) string {
	// Remove any path components
	filename = filepath.Base(filename)

	// Replace potentially dangerous characters
	re := regexp.MustCompile(`[^a-zA-Z0-9.-]`)
	return re.ReplaceAllString(filename, "_")
}

func ValidatePhoneNumber(phone string) bool {
	matched, _ := regexp.MatchString(`^09\d{9}$`, phone)
	return matched
}

func ValidateNationalCode(code string) bool {
	matched, _ := regexp.MatchString(`^\d{10}$`, code)
	return matched
}

// Convert Gregorian date to Solar (Hijri) date
func GregorianToSolar(date time.Time) string {
	p := ptime.New(date)
	// Use the library's built-in formatting for consistent results
	return fmt.Sprintf("%04d-%02d-%02d", p.Year(), int(p.Month()), p.Day())
}

// Convert Solar (Hijri) date to Gregorian date
func SolarToGregorian(date string) (time.Time, error) {
	year, month, day, err := parseSolarDate(date)
	if err != nil {
		return time.Time{}, err
	}

	// Create Persian date with validation
	if month < 1 || month > 12 {
		return time.Time{}, fmt.Errorf("invalid month: %d", month)
	}
	if day < 1 || day > 31 {
		return time.Time{}, fmt.Errorf("invalid day: %d", day)
	}

	pt := ptime.Date(year, ptime.Month(month), day, 0, 0, 0, 0, ptime.Iran())
	return pt.Time(), nil
}

// parseSolarDate splits a Solar date string into year, month, and day with better validation
func parseSolarDate(date string) (int, int, int, error) {
	date = strings.TrimSpace(date)
	parts := strings.Split(date, "-")
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("invalid date format, expected yyyy-MM-dd")
	}

	year, err := strconv.Atoi(parts[0])
	if err != nil || year < 1 {
		return 0, 0, 0, fmt.Errorf("invalid year: %s", parts[0])
	}

	month, err := strconv.Atoi(parts[1])
	if err != nil || month < 1 || month > 12 {
		return 0, 0, 0, fmt.Errorf("invalid month: %s", parts[1])
	}

	day, err := strconv.Atoi(parts[2])
	if err != nil || day < 1 || day > 31 {
		return 0, 0, 0, fmt.Errorf("invalid day: %s", parts[2])
	}

	return year, month, day, nil
}

// CompareSolarDates compares two Solar dates (format: "yyyy-MM-dd")
// Returns:
//
//	-1 if date1 < date2
//	 0 if date1 == date2
//	 1 if date1 > date2
func CompareSolarDates(date1, date2 string) (int, error) {
	y1, m1, d1, err := parseSolarDate(date1)
	if err != nil {
		return 0, err
	}

	y2, m2, d2, err := parseSolarDate(date2)
	if err != nil {
		return 0, err
	}

	switch {
	case y1 < y2:
		return -1, nil
	case y1 > y2:
		return 1, nil
	default: // years equal
		switch {
		case m1 < m2:
			return -1, nil
		case m1 > m2:
			return 1, nil
		default: // months equal
			switch {
			case d1 < d2:
				return -1, nil
			case d1 > d2:
				return 1, nil
			default:
				return 0, nil
			}
		}
	}
}

// SortSolarDates sorts a slice of Solar dates in ascending order
func SortSolarDates(dates []string) error {
	for i := 0; i < len(dates); i++ {
		for j := i + 1; j < len(dates); j++ {
			res, err := CompareSolarDates(dates[i], dates[j])
			if err != nil {
				return err
			}
			if res > 0 {
				dates[i], dates[j] = dates[j], dates[i]
			}
		}
	}
	return nil
}

// IsSolarDateValid checks if a Solar date is valid
func IsSolarDateValid(date string) bool {
	_, _, _, err := parseSolarDate(date)
	return err == nil
}

func RespondWithError(w http.ResponseWriter, code int, message string) {
	RespondWithJSON(w, code, map[string]string{"error": message})
}

func RespondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
