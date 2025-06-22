package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/AmrEsam0/license-manager/pkg/license"
	"github.com/joho/godotenv"
)

func main() {
	// Try loading from current directory
	err := godotenv.Load()
	if err != nil {
		// Load .env file from project root (since CLI is in cmd/license-manager/)
		err = godotenv.Load("../../.env")
		if err != nil {
			// Don't fail if .env file doesn't exist, just log a warning
			log.Printf("Warning: Could not load .env file: %v", err)
		}
	}

	if len(os.Args) < 2 {
		printUsage()
		return
	}

	manager, err := license.NewManager()
	if err != nil {
		fmt.Printf("Error initializing license manager: %v\n", err)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "pcid":
		handlePCID(manager)
	case "create":
		handleCreate(manager)
	case "check":
		handleCheck(manager)
	case "view":
		handleView(manager)
	case "revoke":
		handleRevoke(manager)
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("License Manager")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  license-manager create <product_name> <max_days|lifetime>")
	fmt.Println("  license-manager check <product_name>")
	fmt.Println("  license-manager view <product_name>")
	fmt.Println("  license-manager pcid")
	fmt.Println("  license-manager revoke <product_name>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  pcid             Show the current PC ID")
	fmt.Println("  create           Create a new license")
	fmt.Println("  check            Validate and check license status for specific product")
	fmt.Println("  view             View license details without updating usage for specific product")
	fmt.Println("  revoke           Revoke the license for specific product")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  license-manager create \"My Product\" 30")
	fmt.Println("  license-manager create \"My Product\" lifetime")
	fmt.Println("  license-manager check \"My Product\"")
	fmt.Println("  license-manager view \"My Product\"")
	fmt.Println("  license-manager revoke \"My Product\"")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  LICENSE_MASTER_KEY              Master encryption key (recommended)")
	fmt.Println("  LICENSE_DEFAULT_DAYS            Default license duration in days")
	fmt.Println("  LICENSE_LIFETIME_DAYS           Days representing lifetime license")
	fmt.Println("  LICENSE_DIR                     Directory to store license files (optional)")
	fmt.Println()
	fmt.Println("Notes:")
	fmt.Println("  - License files are created in the directory specified by LICENSE_DIR or current directory")
	fmt.Println("  - Filename format: <product_name>.license")
	fmt.Println("  - Commands like check/view/revoke work with .license files in the license directory")
}

func handlePCID(manager *license.Manager) {
	pcId := manager.GetPCID()
	fmt.Printf("PC ID: %s\n", pcId)
}

func handleCreate(manager *license.Manager) {
	if len(os.Args) < 4 {
		fmt.Println("Usage: license-manager create <product_name> <max_days|lifetime>")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  license-manager create \"My Product\" 30")
		fmt.Println("  license-manager create \"My Product\" lifetime")
		return
	}

	productName := os.Args[2]
	daysStr := os.Args[3]

	// Parse max days
	var maxDays int
	var isLifetime bool
	var err error

	if strings.ToLower(strings.TrimSpace(daysStr)) == "lifetime" {
		maxDays = 99999
		isLifetime = true
	} else {
		maxDays, err = strconv.Atoi(daysStr)
		if err != nil || maxDays <= 0 {
			fmt.Println("Invalid max days. Please provide a positive integer or 'lifetime'.")
			return
		}
		isLifetime = maxDays >= 99999
	}

	req := license.CreateLicenseRequest{
		ProductName: productName,
		MaxDays:     maxDays,
		IsLifetime:  isLifetime,
	}

	createdLicense, err := manager.Create(req)
	if err != nil {
		fmt.Printf("Error creating license: %v\n", err)
		return
	}

	// Generate filename to show user what was created
	sanitizedName := sanitizeProductName(productName)
	filename := sanitizedName + ".license"

	fmt.Printf("License created successfully!\n")
	fmt.Printf("File: %s\n", filename)
	fmt.Printf("Computer ID: %s\n", manager.GetPCID())
	fmt.Printf("Serial: %s\n", createdLicense.Serial)
	if isLifetime {
		fmt.Printf("Type: LIFETIME license\n")
	} else {
		fmt.Printf("Type: %d-day license\n", maxDays)
	}
	fmt.Printf("Product: %s\n", createdLicense.ProductName)
	fmt.Printf("Created: %s\n", createdLicense.CreatedAt.Format("2006-01-02 15:04:05"))
}

func handleCheck(manager *license.Manager) {
	if len(os.Args) < 3 {
		fmt.Println("Usage: license-manager check <product_name>")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  license-manager check \"My Product\"")
		fmt.Println("  license-manager check \"Another App\"")
		return
	}

	productName := os.Args[2]

	result, err := manager.Validate(productName)
	if err != nil {
		fmt.Printf("Error checking license: %v\n", err)
		return
	}

	if !result.IsValid {
		fmt.Printf("License validation failed: %s\n", result.ErrorMessage)
		return
	}

	lic := result.License

	if lic.IsLifetime {
		fmt.Printf("License is VALID (LIFETIME)\n")
		fmt.Printf("Product: %s\n", lic.ProductName)
		fmt.Printf("Used days: %d (unlimited)\n", len(lic.UsageHistory))
		fmt.Printf("Remaining days: UNLIMITED\n")
	} else {
		remainingDays := lic.MaxDays - len(lic.UsageHistory)
		if remainingDays < 0 {
			remainingDays = 0
		}
		fmt.Printf("License is VALID\n")
		fmt.Printf("Product: %s\n", lic.ProductName)
		fmt.Printf("Used days: %d/%d\n", len(lic.UsageHistory), lic.MaxDays)
		fmt.Printf("Remaining days: %d\n", remainingDays)
	}

	fmt.Printf("Total runs: %d\n", lic.RunCount)
	if lic.FirstRunDate != "" {
		fmt.Printf("First activated: %s\n", lic.FirstRunDate)
	}
	if lic.LastUsedDate != "" {
		fmt.Printf("Last used: %s\n", lic.LastUsedDate)
	}
	fmt.Printf("Usage history: %v\n", lic.UsageHistory)
}

func handleView(manager *license.Manager) {
	if len(os.Args) < 3 {
		fmt.Println("Usage: license-manager view <product_name>")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  license-manager view \"My Product\"")
		fmt.Println("  license-manager view \"Another App\"")
		return
	}

	productName := os.Args[2]

	licInfo, err := manager.View(productName)
	if err != nil {
		fmt.Printf("Error viewing license: %v\n", err)
		return
	}

	fmt.Printf("License Details:\n")
	fmt.Printf("Product: %s\n", licInfo.ProductName)
	fmt.Printf("Serial: %s\n", licInfo.Serial)
	fmt.Printf("PC ID: %s\n", licInfo.PCId)
	fmt.Printf("Created: %s\n", licInfo.CreatedAt.Format("2006-01-02 15:04:05"))

	if licInfo.IsLifetime {
		fmt.Printf("License Type: LIFETIME\n")
		fmt.Printf("Used days: %d (unlimited)\n", len(licInfo.UsageHistory))
	} else {
		remainingDays := licInfo.MaxDays - len(licInfo.UsageHistory)
		if remainingDays < 0 {
			remainingDays = 0
		}
		fmt.Printf("License Type: Time-limited\n")
		fmt.Printf("Max days: %d\n", licInfo.MaxDays)
		fmt.Printf("Used days: %d\n", len(licInfo.UsageHistory))
		fmt.Printf("Remaining days: %d\n", remainingDays)
	}

	fmt.Printf("Total runs: %d\n", licInfo.RunCount)
	fmt.Printf("Activated: %v\n", licInfo.IsActivated)
	if licInfo.FirstRunDate != "" {
		fmt.Printf("First run: %s\n", licInfo.FirstRunDate)
	}
	if licInfo.LastUsedDate != "" {
		fmt.Printf("Last used: %s\n", licInfo.LastUsedDate)
	}
	fmt.Printf("Usage history: %v\n", licInfo.UsageHistory)
}

func handleRevoke(manager *license.Manager) {
	if len(os.Args) < 3 {
		fmt.Println("Usage: license-manager revoke <product_name>")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  license-manager revoke \"My Product\"")
		fmt.Println("  license-manager revoke \"Another App\"")
		return
	}

	productName := os.Args[2]

	err := manager.Revoke(productName)
	if err != nil {
		fmt.Printf("Error revoking license: %v\n", err)
		return
	}

	fmt.Printf("License for \"%s\" has been revoked successfully.\n", productName)
}

// sanitizeProductName removes invalid characters from product name for filename use
func sanitizeProductName(name string) string {
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
