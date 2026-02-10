package utils

import (
	"time"
)

// ConvertGMTtoIST parses a datetime string in GMT and returns it formatted in IST (UTC+5:30).
// The input must use the layout "2006-01-02 15:04:05" in the GMT timezone.
//
// Parameters:
//   - gmtDatetime: Datetime string in "2006-01-02 15:04:05" format, interpreted as GMT
//
// Returns:
//   - The same instant formatted as "2006-01-02 15:04:05" in IST
//   - An error if parsing fails (e.g. invalid layout or timezone)
func ConvertGMTtoIST(gmtDatetime string) (string, error) {
	// Define GMT timezone
	gmtLocation, err := time.LoadLocation("GMT")
	if err != nil {
		return "", err
	}

	// Parse the GMT datetime string
	gmtTime, err := time.ParseInLocation("2006-01-02 15:04:05", gmtDatetime, gmtLocation)
	if err != nil {
		return "", err
	}

	// Add 5 hours 30 minutes to convert GMT to IST (UTC+5:30)
	istTime := gmtTime.Add(5*time.Hour + 30*time.Minute)

	// Format the IST time as a string
	istDatetime := istTime.Format("2006-01-02 15:04:05")

	return istDatetime, nil
}

// ConvertUTCtoIST parses a datetime string in UTC (RFC 3339 style with Z suffix) and
// returns it formatted in IST (UTC+5:30). The input must use the layout "2006-01-02T15:04:05Z".
//
// Parameters:
//   - utcDatetime: Datetime string in "2006-01-02T15:04:05Z" format (UTC)
//
// Returns:
//   - The same instant formatted as "2006-01-02 15:04:05" in IST
//   - An error if parsing fails (e.g. invalid layout)
func ConvertUTCtoIST(utcDatetime string) (string, error) {
	// Parse the UTC datetime string
	utcTime, err := time.Parse("2006-01-02T15:04:05Z", utcDatetime)
	if err != nil {
		return "", err
	}

	// Add 5 hours 30 minutes to convert UTC to IST (UTC+5:30)
	istTime := utcTime.Add(5*time.Hour + 30*time.Minute)

	// Format the IST time as a string
	istDatetime := istTime.Format("2006-01-02 15:04:05")

	return istDatetime, nil
}
