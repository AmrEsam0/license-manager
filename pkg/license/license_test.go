package license

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AmrEsam0/license-manager/pkg/config"
)

// TestProductName is used across tests
const TestProductName = "TestProduct"

// setupTestManager creates a test license manager
func setupTestManager(t *testing.T) (*Manager, string) {
	t.Helper()

	// Create a temp directory for test license files
	tempDir, err := os.MkdirTemp("", "license-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Set the LICENSE_DIR environment variable
	oldLicenseDir := os.Getenv("LICENSE_DIR")
	os.Setenv("LICENSE_DIR", tempDir)

	// Set a fixed master key for testing
	oldMasterKey := os.Getenv("LICENSE_MASTER_KEY")
	os.Setenv("LICENSE_MASTER_KEY", "TestMasterKeyForLicenseTests12345678901234")

	// Create the manager
	manager, err := NewManager()
	if err != nil {
		os.Setenv("LICENSE_DIR", oldLicenseDir)
		os.Setenv("LICENSE_MASTER_KEY", oldMasterKey)
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create license manager: %v", err)
	}

	// Return the manager and temp dir (for cleanup)
	return manager, tempDir
}

// cleanupTest restores environment and removes temp files
func cleanupTest(t *testing.T, tempDir string) {
	t.Helper()

	// Remove the temp directory
	os.RemoveAll(tempDir)

	// Restore environment variables
	os.Unsetenv("LICENSE_DIR")
	os.Unsetenv("LICENSE_MASTER_KEY")
}

// TestCreateLicense tests creating a new license
func TestCreateLicense(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTest(t, tempDir)

	// Create a new license
	req := CreateLicenseRequest{
		ProductName: TestProductName,
		MaxDays:     30,
		IsLifetime:  false,
	}

	license, err := manager.Create(req)
	if err != nil {
		t.Fatalf("Failed to create license: %v", err)
	}

	// Verify license properties
	if license.ProductName != TestProductName {
		t.Errorf("Expected product name %s, got %s", TestProductName, license.ProductName)
	}
	if license.MaxDays != 30 {
		t.Errorf("Expected max days 30, got %d", license.MaxDays)
	}
	if license.IsLifetime != false {
		t.Errorf("Expected is_lifetime false, got %t", license.IsLifetime)
	}
	if license.PCId != manager.GetPCID() {
		t.Errorf("Expected PC ID %s, got %s", manager.GetPCID(), license.PCId)
	}
	if license.RunCount != 0 {
		t.Errorf("Expected run count 0, got %d", license.RunCount)
	}
	if license.IsActivated != false {
		t.Errorf("Expected is_activated false, got %t", license.IsActivated)
	}

	// Verify license file was created
	sanitizedName := TestProductName
	sanitizedName = filepath.Join(tempDir, sanitizedName+".license")
	if _, err := os.Stat(sanitizedName); os.IsNotExist(err) {
		t.Errorf("License file was not created at %s", sanitizedName)
	}
}

// TestCreateLifetimeLicense tests creating a lifetime license
func TestCreateLifetimeLicense(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTest(t, tempDir)

	// Create a lifetime license
	req := CreateLicenseRequest{
		ProductName: TestProductName,
		MaxDays:     0,
		IsLifetime:  true,
	}

	license, err := manager.Create(req)
	if err != nil {
		t.Fatalf("Failed to create license: %v", err)
	}

	// Verify license is lifetime
	if !license.IsLifetime {
		t.Errorf("Expected lifetime license, but is_lifetime is false")
	}

	// Lifetime licenses should have MaxDays set to a very large number
	if license.MaxDays < 99000 {
		t.Errorf("Expected large max days for lifetime license, got %d", license.MaxDays)
	}
}

// TestValidateProduct tests validating a product license
func TestValidateProduct(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTest(t, tempDir)

	// First create a license
	req := CreateLicenseRequest{
		ProductName: TestProductName,
		MaxDays:     30,
		IsLifetime:  false,
	}

	_, err := manager.Create(req)
	if err != nil {
		t.Fatalf("Failed to create license: %v", err)
	}

	// Now validate it
	result, err := manager.Validate(TestProductName)
	if err != nil {
		t.Fatalf("Failed to validate license: %v", err)
	}

	// Check validation result
	if !result.IsValid {
		t.Errorf("Expected license to be valid, but it's not: %s", result.ErrorMessage)
	}

	// After validation, the license should be activated
	if result.License.IsActivated != true {
		t.Errorf("Expected license to be activated after validation")
	}

	// Run count should be incremented
	if result.License.RunCount != 1 {
		t.Errorf("Expected run count 1 after validation, got %d", result.License.RunCount)
	}

	// Should have today in usage history
	today := time.Now().Format("2006-01-02")
	found := false
	for _, date := range result.License.UsageHistory {
		if date == today {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected today's date in usage history after validation")
	}
}

// TestValidateProductNotFound tests validating a non-existent product
func TestValidateProductNotFound(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTest(t, tempDir)

	// Try to validate a non-existent product
	result, err := manager.Validate("NonExistentProduct")
	if err != nil {
		t.Fatalf("Unexpected error from Validate: %v", err)
	}

	// Should not be valid
	if result.IsValid {
		t.Errorf("Expected license to be invalid for non-existent product, but it's valid")
	}
}

// TestGetProductInfo tests getting license info for a product
func TestGetProductInfo(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTest(t, tempDir)

	// First create a license
	req := CreateLicenseRequest{
		ProductName: TestProductName,
		MaxDays:     30,
		IsLifetime:  false,
	}

	_, err := manager.Create(req)
	if err != nil {
		t.Fatalf("Failed to create license: %v", err)
	}

	// Validate to activate
	_, err = manager.Validate(TestProductName)
	if err != nil {
		t.Fatalf("Failed to validate license: %v", err)
	}

	// Now get info
	info, err := manager.GetInfo(TestProductName)
	if err != nil {
		t.Fatalf("Failed to get license info: %v", err)
	}

	// Check info properties
	if info.ProductName != TestProductName {
		t.Errorf("Expected product name %s, got %s", TestProductName, info.ProductName)
	}
	if info.MaxDays != 30 {
		t.Errorf("Expected max days 30, got %d", info.MaxDays)
	}
	if info.IsLifetime != false {
		t.Errorf("Expected is_lifetime false, got %t", info.IsLifetime)
	}
	if info.UsedDays != 1 {
		t.Errorf("Expected used days 1, got %d", info.UsedDays)
	}
	if info.RemainingDays != 29 {
		t.Errorf("Expected remaining days 29, got %d", info.RemainingDays)
	}
	if info.RunCount != 2 {
		t.Errorf("Expected run count 2, got %d", info.RunCount)
	}
	if !info.IsValid {
		t.Errorf("Expected license to be valid")
	}
}

// TestViewProduct tests viewing license data without updating usage
func TestViewProduct(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTest(t, tempDir)

	// First create a license
	req := CreateLicenseRequest{
		ProductName: TestProductName,
		MaxDays:     30,
		IsLifetime:  false,
	}

	createdLicense, err := manager.Create(req)
	if err != nil {
		t.Fatalf("Failed to create license: %v", err)
	}

	// Now view it
	viewedLicense, err := manager.View(TestProductName)
	if err != nil {
		t.Fatalf("Failed to view license: %v", err)
	}

	// Should have same properties as created license
	if viewedLicense.ProductName != createdLicense.ProductName {
		t.Errorf("Expected product name %s, got %s", createdLicense.ProductName, viewedLicense.ProductName)
	}
	if viewedLicense.Serial != createdLicense.Serial {
		t.Errorf("Expected serial %s, got %s", createdLicense.Serial, viewedLicense.Serial)
	}
	if viewedLicense.PCId != createdLicense.PCId {
		t.Errorf("Expected PC ID %s, got %s", createdLicense.PCId, viewedLicense.PCId)
	}

	// Viewing should not increment run count or mark as activated
	if viewedLicense.RunCount != 0 {
		t.Errorf("Viewing license should not update run count, got %d", viewedLicense.RunCount)
	}
	if viewedLicense.IsActivated {
		t.Errorf("Viewing license should not mark it as activated")
	}
}

// TestRevokeProduct tests revoking a license
func TestRevokeProduct(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTest(t, tempDir)

	// First create a license
	req := CreateLicenseRequest{
		ProductName: TestProductName,
		MaxDays:     30,
		IsLifetime:  false,
	}

	_, err := manager.Create(req)
	if err != nil {
		t.Fatalf("Failed to create license: %v", err)
	}

	// Now revoke it
	err = manager.Revoke(TestProductName)
	if err != nil {
		t.Fatalf("Failed to revoke license: %v", err)
	}

	// Try to validate the license - should fail
	result, err := manager.Validate(TestProductName)
	if err != nil {
		t.Fatalf("Unexpected error from Validate after revocation: %v", err)
	}

	// Should not be valid
	if result.IsValid {
		t.Errorf("Expected license to be invalid after revocation, but it's valid")
	}
}

// TestRevokeProductNotFound tests revoking a non-existent license
func TestRevokeProductNotFound(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTest(t, tempDir)

	// Try to revoke a non-existent license
	err := manager.Revoke("NonExistentProduct")
	if err == nil {
		t.Errorf("Expected error when revoking non-existent license, but got nil")
	}
}

// TestMultipleProducts tests managing multiple product licenses
func TestMultipleProducts(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTest(t, tempDir)

	// Create two different product licenses
	products := []string{"Product1", "Product2"}

	for _, product := range products {
		req := CreateLicenseRequest{
			ProductName: product,
			MaxDays:     30,
			IsLifetime:  false,
		}

		_, err := manager.Create(req)
		if err != nil {
			t.Fatalf("Failed to create license for %s: %v", product, err)
		}
	}

	// Validate both products
	for _, product := range products {
		result, err := manager.Validate(product)
		if err != nil {
			t.Fatalf("Failed to validate license for %s: %v", product, err)
		}
		if !result.IsValid {
			t.Errorf("License for %s should be valid", product)
		}
	}

	// Revoke one product
	err := manager.Revoke(products[0])
	if err != nil {
		t.Fatalf("Failed to revoke license for %s: %v", products[0], err)
	}

	// First product should now be invalid
	result, _ := manager.Validate(products[0])
	if result.IsValid {
		t.Errorf("License for %s should be invalid after revocation", products[0])
	}

	// Second product should still be valid
	result, _ = manager.Validate(products[1])
	if !result.IsValid {
		t.Errorf("License for %s should still be valid", products[1])
	}
}

// TestPCIDMismatch tests license validation with mismatched PC ID
func TestPCIDMismatch(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTest(t, tempDir)

	// Create a mocked license with a different PC ID
	cfg := config.LoadConfig()

	// We'll use a fake PC ID for creating the license
	fakePCID := "fake-pc-id-for-testing"

	// Save original PC ID
	originalPCID := manager.PCID

	// Temporarily set the manager's PC ID to the fake one
	manager.PCID = fakePCID

	req := CreateLicenseRequest{
		ProductName: TestProductName,
		MaxDays:     30,
		IsLifetime:  false,
	}

	// Create the license
	license, err := manager.Create(req)
	if err != nil {
		t.Fatalf("Failed to create license with fake PC ID: %v", err)
	}

	// License should be created with the fake PC ID
	if license.PCId != fakePCID {
		t.Errorf("Expected PC ID %s, got %s", fakePCID, license.PCId)
	}

	// Restore original PC ID
	manager.PCID = originalPCID

	// Validation should fail due to PC ID mismatch
	// We need to manually validate since we're using a fake PC ID
	licenseFile, _ := cfg.GetLicenseFilePathForProduct(TestProductName)
	_, err = manager.readAndVerifyLicense(licenseFile, manager.PCID)
	if err == nil {
		t.Errorf("Expected validation to fail due to PC ID mismatch, but it succeeded")
	}
}

// MockPCIDGenerator creates a mock PC ID generator for testing
type MockPCIDGenerator struct {
	pcid string
}

func (m *MockPCIDGenerator) Generate() (string, error) {
	return m.pcid, nil
}

func (m *MockPCIDGenerator) IsSupported() bool {
	return true
}

func (m *MockPCIDGenerator) GetSupportedPlatforms() string {
	return "mock"
}

// TestNewManager tests creating a new manager
func TestNewManager(t *testing.T) {
	// Set a test master key
	oldMasterKey := os.Getenv("LICENSE_MASTER_KEY")
	os.Setenv("LICENSE_MASTER_KEY", "TestNewManagerKey12345678901234567890")
	defer os.Setenv("LICENSE_MASTER_KEY", oldMasterKey)

	// Create a new manager
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create new manager: %v", err)
	}

	// Verify the manager was created successfully
	if manager == nil {
		t.Fatalf("NewManager returned nil manager")
	}

	// Manager should have a valid PC ID
	if manager.GetPCID() == "" {
		t.Errorf("New manager has empty PC ID")
	}
}

// TestNewManagerWithKey tests creating a new manager with a provided key
func TestNewManagerWithKey(t *testing.T) {
	// Create a new manager with a custom key
	manager, err := NewManagerWithKey("TestNewManagerWithKey12345678901234567890")
	if err != nil {
		t.Fatalf("Failed to create new manager with key: %v", err)
	}

	// Verify the manager was created successfully
	if manager == nil {
		t.Fatalf("NewManagerWithKey returned nil manager")
	}

	// Manager should have a valid PC ID
	if manager.GetPCID() == "" {
		t.Errorf("New manager has empty PC ID")
	}
}
