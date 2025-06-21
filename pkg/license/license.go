package license

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"slices"

	"github.com/AmrEsam0/license-manager/pkg/config"
	"github.com/AmrEsam0/license-manager/pkg/crypto"
	"github.com/AmrEsam0/license-manager/pkg/hardware"
)

// Manager handles all license operations
type Manager struct {
	config  *config.Config
	crypto  *crypto.CryptoManager
	pcidGen *hardware.PCIDGenerator
	PCID    string
}

// NewManager creates a new license manager
func NewManager() (*Manager, error) {
	cfg := config.LoadConfig()
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %v", err)
	}

	cryptoMgr, err := crypto.NewCryptoManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize crypto manager: %v", err)
	}

	pcidGen := hardware.NewPCIDGenerator()
	if !pcidGen.IsSupported() {
		return nil, fmt.Errorf("unsupported platform: %s", pcidGen.GetSupportedPlatforms())
	}

	PCID, err := pcidGen.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate PC ID: %v", err)
	}

	return &Manager{
		config:  cfg,
		crypto:  cryptoMgr,
		pcidGen: pcidGen,
		PCID:    PCID,
	}, nil
}

// NewManagerWithKey creates a new license manager with a provided master key
func NewManagerWithKey(masterKey string) (*Manager, error) {
	cfg := config.LoadConfig()
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %v", err)
	}

	cryptoMgr, err := crypto.NewCryptoManagerWithKey(masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize crypto manager: %v", err)
	}

	pcidGen := hardware.NewPCIDGenerator()
	if !pcidGen.IsSupported() {
		return nil, fmt.Errorf("unsupported platform: %s", pcidGen.GetSupportedPlatforms())
	}

	currentPC, err := pcidGen.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate PC ID: %v", err)
	}

	return &Manager{
		config:  cfg,
		crypto:  cryptoMgr,
		pcidGen: pcidGen,
		PCID:    currentPC,
	}, nil
}

// GetPCID returns the current PC ID
func (m *Manager) GetPCID() string {
	return m.PCID
}

// Create creates a new license with the given parameters
func (m *Manager) Create(req CreateLicenseRequest) (*License, error) {
	licenseFile, err := m.config.GetLicenseFilePathForProduct(req.ProductName)
	if err != nil {
		return nil, fmt.Errorf("failed to get license file path: %v", err)
	}

	// Check if license file already exists
	if _, err := os.Stat(licenseFile); err == nil {
		return nil, fmt.Errorf("license file already exists")
	}

	// Handle lifetime license
	maxDays := req.MaxDays
	isLifetime := req.IsLifetime || m.config.IsLifetimeRequest(maxDays)
	if isLifetime {
		maxDays = m.config.LifetimeDays
	}

	// Determine PCID to use
	pcid := m.PCID

	// generate a new serial number

	serial := m.crypto.GenerateSerial(pcid, req.ProductName, maxDays)

	license := &License{
		Serial:       serial,
		PCId:         pcid,
		ProductName:  req.ProductName,
		CreatedAt:    time.Now(),
		MaxDays:      maxDays,
		IsLifetime:   isLifetime,
		LastUsedDate: "",
		FirstRunDate: "",
		RunCount:     0,
		IsActivated:  false,
		UsageHistory: []string{},
		UsageMap:     make(map[string]bool),
	}

	if err := m.saveLicense(license, licenseFile); err != nil {
		return nil, fmt.Errorf("failed to save license: %v", err)
	}

	return license, nil
}

// ValidateProduct validates a specific product's license and updates usage tracking
func (m *Manager) Validate(productName string) (*ValidationResult, error) {
	licenseFile, err := m.config.GetLicenseFilePathForProduct(productName)
	if err != nil {
		return &ValidationResult{
			IsValid:      false,
			ErrorMessage: fmt.Sprintf("failed to get license file path for product %s: %v", productName, err),
		}, nil
	}

	license, err := m.readAndVerifyLicense(licenseFile, m.PCID)
	if err != nil {
		return &ValidationResult{
			IsValid:      false,
			ErrorMessage: fmt.Sprintf("license validation failed for product %s: %v", productName, err),
		}, nil
	}

	return &ValidationResult{
		IsValid: true,
		License: license,
	}, nil
}

// GetProductInfo returns read-only license information for a specific product
func (m *Manager) GetInfo(productName string) (*LicenseInfo, error) {
	result, err := m.Validate(productName)
	if err != nil {
		return nil, err
	}

	if !result.IsValid {
		return nil, fmt.Errorf("%s", result.ErrorMessage)
	}

	license := result.License
	remainingDays := 0
	if !license.IsLifetime {
		remainingDays = max(license.MaxDays-len(license.UsageHistory), 0)
	}

	info := &LicenseInfo{
		ProductName:   license.ProductName,
		IsLifetime:    license.IsLifetime,
		MaxDays:       license.MaxDays,
		UsedDays:      len(license.UsageHistory),
		RemainingDays: remainingDays,
		RunCount:      license.RunCount,
		FirstRunDate:  license.FirstRunDate,
		LastUsedDate:  license.LastUsedDate,
		UsageHistory:  license.UsageHistory,
		IsValid:       true,
	}

	return info, nil
}

// ViewProduct returns the raw license data for a specific product (without updating usage)

// ViewProduct retrieves the license for a specific product without updating usage
func (m *Manager) View(productName string) (*License, error) {
	licenseFile, err := m.config.GetLicenseFilePathForProduct(productName)
	if err != nil {
		return nil, fmt.Errorf("failed to get license file path for product %s: %v", productName, err)
	}

	encryptedData, err := os.ReadFile(licenseFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read license file for product %s: %v", productName, err)
	}

	data, err := m.crypto.Decrypt(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt license file for product %s: %v", productName, err)
	}

	var license License
	if err := json.Unmarshal(data, &license); err != nil {
		return nil, fmt.Errorf("failed to parse license file for product %s: %v", productName, err)
	}

	if license.PCId != m.PCID {
		return nil, fmt.Errorf("license for product %s is not valid for this PC %s - expected: %s", productName, m.PCID, license.PCId)
	}

	return &license, nil
}

// RevokeProduct invalidates a specific product's license
func (m *Manager) Revoke(productName string) error {
	licenseFile, err := m.config.GetLicenseFilePathForProduct(productName)
	if err != nil {
		return fmt.Errorf("failed to get license file path for product %s: %v", productName, err)
	}

	// Check if license file exists
	if _, err := os.Stat(licenseFile); os.IsNotExist(err) {
		return fmt.Errorf("no license file found for product %s", productName)
	}

	// Corrupt the license file by overwriting it with random data
	// This makes any copied versions of the license invalid
	corruptData, err := m.crypto.GenerateRandomBytes(1024) // 1KB of random data
	if err != nil {
		return fmt.Errorf("failed to generate corrupt data: %v", err)
	}

	// Overwrite the license file with corrupt data
	if err := os.WriteFile(licenseFile, corruptData, 0644); err != nil {
		return fmt.Errorf("failed to corrupt license file for product %s: %v", productName, err)
	}

	return nil
}

// saveLicense encrypts and saves the license to file
func (m *Manager) saveLicense(license *License, filename string) error {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	data, err := json.MarshalIndent(license, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal license: %v", err)
	}

	encryptedData, err := m.crypto.Encrypt(data)
	if err != nil {
		return fmt.Errorf("failed to encrypt license: %v", err)
	}

	if err := os.WriteFile(filename, encryptedData, 0644); err != nil {
		return fmt.Errorf("failed to write license file: %v", err)
	}

	return nil
}

// readAndVerifyLicense reads, decrypts, and verifies the license
func (m *Manager) readAndVerifyLicense(filename, currentPcId string) (*License, error) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, fmt.Errorf("license file not found")
	}

	encryptedData, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read license file: %v", err)
	}

	data, err := m.crypto.Decrypt(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt license file (file may be corrupted): %v", err)
	}

	var license License
	if err := json.Unmarshal(data, &license); err != nil {
		return nil, fmt.Errorf("failed to parse license file: %v", err)
	}

	if license.PCId != currentPcId {
		return nil, fmt.Errorf("license is not valid for this PC")
	}

	expectedSerial := m.crypto.GenerateSerial(currentPcId, license.ProductName, license.MaxDays)
	if license.Serial != expectedSerial {
		return nil, fmt.Errorf("license serial is invalid")
	}

	now := time.Now()
	nowRFC3339 := now.Format(time.RFC3339)
	today := now.Format("2006-01-02")

	if !license.IsActivated {
		license.IsActivated = true
		license.FirstRunDate = nowRFC3339
		license.LastUsedDate = nowRFC3339
		license.RunCount = 1
		license.UsageHistory = []string{today}
		// Initialize usage map for better performance
		license.UsageMap = map[string]bool{today: true}
	} else {
		license.RunCount++

		// Simple time rollback detection and same time check
		if license.LastUsedDate != "" {
			// Check if it's the same time (prevent multiple uses within same time)
			if license.LastUsedDate == nowRFC3339 {
				// Same exact time, just update and return
				if err := m.saveLicense(&license, filename); err != nil {
					return nil, fmt.Errorf("failed to update license usage: %v", err)
				}
				return &license, nil
			}

			// Basic time rollback check - if last used date is in the future compared to now
			if license.LastUsedDate > nowRFC3339 {
				return nil, fmt.Errorf("system date/time appears to have been rolled back - license validation failed")
			}
		}

		// Initialize usage map if it doesn't exist (for backward compatibility)
		if license.UsageMap == nil {
			license.UsageMap = make(map[string]bool)
			for _, date := range license.UsageHistory {
				license.UsageMap[date] = true
			}
		}

		// Use map for O(1) lookup, fallback to optimized array search
		usedToday := false
		if license.UsageMap != nil {
			usedToday = license.UsageMap[today]
		} else {
			// Performance optimization: check the last entry first since dates are added chronologically
			if len(license.UsageHistory) > 0 && license.UsageHistory[len(license.UsageHistory)-1] == today {
				usedToday = true
			} else {
				// Fallback: search through all entries (for backward compatibility or edge cases)
				if slices.Contains(license.UsageHistory, today) {
					usedToday = true
				}
			}
		}

		if !usedToday {
			license.LastUsedDate = nowRFC3339
			license.UsageHistory = append(license.UsageHistory, today)
			// Update usage map
			if license.UsageMap != nil {
				license.UsageMap[today] = true
			}
		} else {
			license.LastUsedDate = nowRFC3339
		}
	}

	if !license.IsLifetime && len(license.UsageHistory) > license.MaxDays {
		return nil, fmt.Errorf("license has expired - used %d days out of %d allowed", len(license.UsageHistory), license.MaxDays)
	}

	if err := m.saveLicense(&license, filename); err != nil {
		return nil, fmt.Errorf("failed to update license usage: %v", err)
	}

	return &license, nil
}
