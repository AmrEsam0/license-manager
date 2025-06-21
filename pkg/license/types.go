package license

import "time"

// License represents the license structure
type License struct {
	Serial       string          `json:"serial"`
	PCId         string          `json:"pc_id"`
	ProductName  string          `json:"product_name"`
	CreatedAt    time.Time       `json:"created_at"`
	MaxDays      int             `json:"max_days"`
	IsLifetime   bool            `json:"is_lifetime"`
	LastUsedDate string          `json:"last_used_date"`
	FirstRunDate string          `json:"first_run_date"`
	RunCount     int             `json:"run_count"`
	IsActivated  bool            `json:"is_activated"`
	UsageHistory []string        `json:"usage_history"`
	UsageMap     map[string]bool `json:"usage_map,omitempty"`
}

// LicenseInfo provides read-only license information
type LicenseInfo struct {
	ProductName   string
	IsLifetime    bool
	MaxDays       int
	UsedDays      int
	RemainingDays int
	RunCount      int    // Changed from TotalRuns to match the License struct
	FirstRunDate  string // Changed from FirstActivated to match the License struct
	LastUsedDate  string // Changed from LastUsed to match the License struct
	UsageHistory  []string
	IsValid       bool
}

// CreateLicenseRequest represents the parameters for creating a new license
type CreateLicenseRequest struct {
	ProductName string
	MaxDays     int
	IsLifetime  bool
}

// ValidationResult contains the result of license validation
type ValidationResult struct {
	IsValid      bool
	License      *License
	ErrorMessage string
}
