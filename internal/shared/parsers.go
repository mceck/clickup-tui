package shared

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func ToInt(s string) int {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	if err != nil {
		return 0
	}
	return i
}

func ToUnix(s string) int {
	// s is a string in DD/MM/YYYY format
	// Convert it to int unix timestamp
	// Split the string by "/"
	parts := strings.Split(s, "/")
	// Convert each part to int
	day := ToInt(parts[0])
	month := ToInt(parts[1])
	year := ToInt(parts[2])
	// Create a time.Time object
	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	// Convert to unix timestamp
	unix := int(t.Unix())
	// Convert to milliseconds
	unixMillis := unix * 1000
	return unixMillis
}

func ToDate(s string) time.Time {
	// s is a string in unix timestamp format
	// Convert it to int
	i := ToInt(s)
	// Convert to time.Time object
	t := time.Unix(int64(i/1000), 0)
	return t
}

func ToDateString(s string) string {
	return ToDate(s).Format("2006-01-02")
}

func ToHours(s string) float64 {
	duration := ToInt(s)
	return float64(duration) / 3600000.0
}

func ToElapsedTime(s string) string {
	date := ToDate(s)
	now := time.Now()
	elapsed := now.Sub(date)
	days := int(elapsed.Hours() / 24)
	if days == 0 {
		hours := int(elapsed.Hours())
		if hours == 0 {
			minutes := int(elapsed.Minutes())
			return fmt.Sprintf("%dm", minutes)
		}
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dd", days)
}

func LightenColor(color string, percentage float64) string {
	// color is a hex color #RRGGBB
	// lighten it by 20%
	// return the new color
	r, g, b := hexToRGB(color)
	r = int(float64(r) + float64(255-r)*percentage)
	g = int(float64(g) + float64(255-g)*percentage)
	b = int(float64(b) + float64(255-b)*percentage)
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

func hexToRGB(s string) (int, int, int) {
	s = strings.TrimPrefix(s, "#")
	r, _ := strconv.ParseUint(s[:2], 16, 8)
	g, _ := strconv.ParseUint(s[2:4], 16, 8)
	b, _ := strconv.ParseUint(s[4:], 16, 8)
	return int(r), int(g), int(b)
}
