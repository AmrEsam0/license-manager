package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config holds the configuration for the license manager
type Config struct {
	// Application settings
	DefaultMaxDays int
	LifetimeDays   int

	// Security settings
	MasterKey string
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		DefaultMaxDays: 30,
		LifetimeDays:   99999,
		MasterKey:      "", // Will be set by environment or default
	}
}

// LoadConfig loads configuration from environment variables or uses defaults
func LoadConfig() *Config {
	config := DefaultConfig()

	// Override with environment variables if they exist
	if defaultDays := os.Getenv("LICENSE_DEFAULT_DAYS"); defaultDays != "" {
		if days, err := strconv.Atoi(defaultDays); err == nil && days > 0 {
			config.DefaultMaxDays = days
		}
	}

	if lifetimeDays := os.Getenv("LICENSE_LIFETIME_DAYS"); lifetimeDays != "" {
		if days, err := strconv.Atoi(lifetimeDays); err == nil && days > 0 {
			config.LifetimeDays = days
		}
	}

	// Master key is handled in crypto package, but we store the env var name here
	config.MasterKey = os.Getenv("LICENSE_MASTER_KEY")

	return config
}

// GetLicenseFilePathForProduct returns the license file path with product name
func (c *Config) GetLicenseFilePathForProduct(productName string) (string, error) {
	licenseDir := os.Getenv("LICENSE_DIR")
	var dir string
	var err error
	if licenseDir != "" {
		dir = licenseDir
	} else {
		dir, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	// Sanitize product name for filename use
	filename := sanitizeFilename(productName) + ".license"
	f := filepath.Join(dir, filename)
	return f, nil
}

// FindLicenseFile finds the first .license file in the license directory or current directory
func (c *Config) FindLicenseFile() (string, error) {
	licenseDir := os.Getenv("LICENSE_DIR")
	var dir string
	var err error
	if licenseDir != "" {
		dir = licenseDir
	} else {
		dir, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".license") {
			return filepath.Join(dir, entry.Name()), nil
		}
	}

	return "", os.ErrNotExist
}

// sanitizeFilename removes invalid characters from filename
func sanitizeFilename(name string) string {
	// Replace spaces and invalid characters with underscores
	result := strings.ReplaceAll(name, " ", "_")
	result = strings.ReplaceAll(result, "/", "_")
	result = strings.ReplaceAll(result, "\\", "_")
	result = strings.ReplaceAll(result, ":", "_")
	result = strings.ReplaceAll(result, "*", "_")
	result = strings.ReplaceAll(result, "?", "_")
	result = strings.ReplaceAll(result, "\"", "_")
	result = strings.ReplaceAll(result, "<", "_")
	result = strings.ReplaceAll(result, ">", "_")
	result = strings.ReplaceAll(result, "|", "_")
	return result
}

// IsLifetimeRequest checks if the given days value represents a lifetime license
func (c *Config) IsLifetimeRequest(maxDays int) bool {
	return maxDays == -1 || maxDays >= c.LifetimeDays
}

// IsLifetimeString checks if the given string represents a lifetime license request
func (c *Config) IsLifetimeString(daysStr string) bool {
	return strings.ToLower(strings.TrimSpace(daysStr)) == "lifetime"
}

// ParseMaxDays parses a string into max days, handling "lifetime" keyword
func (c *Config) ParseMaxDays(daysStr string) (int, bool, error) {
	if c.IsLifetimeString(daysStr) {
		return c.LifetimeDays, true, nil
	}

	days, err := strconv.Atoi(daysStr)
	if err != nil {
		return 0, false, err
	}

	if days <= 0 {
		return 0, false, strconv.ErrRange
	}

	isLifetime := c.IsLifetimeRequest(days)
	if isLifetime {
		days = c.LifetimeDays
	}

	return days, isLifetime, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.DefaultMaxDays <= 0 {
		return &ConfigError{Field: "DefaultMaxDays", Message: "must be positive"}
	}

	if c.LifetimeDays <= 0 {
		return &ConfigError{Field: "LifetimeDays", Message: "must be positive"}
	}

	return nil
}

// ConfigError represents a configuration validation error
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return "config error in " + e.Field + ": " + e.Message
}
