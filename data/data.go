package data

import (
	"time"
)

type Activity struct {
	ActivityID           int       `json:"activityId"`
	ActivityParentID     int       `json:"activityParentId"`
	ActivityParentName   string    `json:"activityParentName"`
	Calories             int       `json:"calories"`
	Description          string    `json:"description"`
	Distance             float64   `json:"distance"`
	Duration             int64     `json:"duration"`
	HasActiveZoneMinutes bool      `json:"hasActiveZoneMinutes"`
	HasStartTime         bool      `json:"hasStartTime"`
	IsFavorite           bool      `json:"isFavorite"`
	LastModified         time.Time `json:"lastModified"`
	LogID                int64     `json:"logId"`
	Name                 string    `json:"name"`
	StartDate            string    `json:"startDate"`
	StartTime            string    `json:"startTime"`
	Steps                int       `json:"steps"`
}

type Activities struct {
	Activities []Activity `json:"activities"`
}

type Credentials struct {
	CId         string `json:"clientID"`
	CSecret     string `json:"clientSecret"`
	RedirectURL string `json:"redirectUrl"`
}
