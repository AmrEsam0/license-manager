# License Manager

A secure, cross-platform license management library for Go applications with hardware-based PC identification and encrypted license files.

or

compile it for a cross-platform CLI tool.


## Features

-   **Hardware-Based PC Identification**: Generates unique PC IDs using hardware characteristics (CPU, motherboard, MAC address, etc.)
-   **Cross-Platform Support**: Works on Windows, Linux, and macOS
-   **Encrypted License Files**: AES-GCM encryption with derived keys
-   **Time-Based Licensing**: Support for both time-limited and lifetime licenses
-   **Usage Tracking**: Tracks daily usage with time rollback detection
-   **Secure Key Management**: Environment variable support for production deployments


## Quick Start

### Installation

```bash
go get github.com/AmrEsam0/license-manager
```

### Basic Usage

#### Option 1: Get your Master Key from your secure source

```go
import (
    "fmt"
    "log"

    "github.com/AmrEsam0/license-manager/pkg/license"
)

func main() {
    // Set your master key directly
    masterKey := "YourUniqueApplicationKey2024"

    // Initialize the license manager with programmatic key
    manager, err := license.NewManagerWithKey(masterKey)
    if err != nil {
        log.Fatal(err)
    }

    // Validate the license for a specific product
    result, err := manager.Validate("My Product")
    if err != nil {
        log.Fatal(err)
    }

    if !result.IsValid {
        log.Fatalf("License validation failed: %s", result.ErrorMessage)
    }

    fmt.Printf("License is valid for product: %s\n", result.License.ProductName)
}
```

#### Option 2: Environment Variables

```go
import (
    "fmt"
    "log"

    "github.com/AmrEsam0/license-manager/pkg/license"
)

func main() {
    // Uses LICENSE_MASTER_KEY environment variable or default
    manager, err := license.NewManager()
    if err != nil {
        log.Fatal(err)
    }
    defer manager.Cleanup()

    // Validate the license for a specific product
    result, err := manager.ValidateProduct("My Product")
    if err != nil {
        log.Fatal(err)
    }

    if !result.IsValid {
        log.Fatalf("License validation failed: %s", result.ErrorMessage)
    }

    fmt.Printf("License is valid for product: %s\n", result.License.ProductName)
}
```

### CLI Usage

### Build the CLI tool

```bash
cd cmd/license-manager
go build -o license-manager
```

### Command Line Interface

The package includes a CLI tool for license operations:

```bash
# Show PC ID
license-manager pcid

# Create a 30-day license (creates "My_Product.license" in license directory)
license-manager create "My Product" 30

# Create a lifetime license (creates "My_Product.license" in license directory)
license-manager create "My Product" lifetime

# Check license status for a specific product
license-manager check "My Product"

# View license details for a specific product
license-manager view "My Product"

# Revoke license for a specific product
license-manager revoke "My Product"
```


### Best Practices

1. **Always set LICENSE_MASTER_KEY** in production
2. **Use strong, random keys** (32+ characters)
3. **Store keys securely** (use secret management systems)
4. **Rotate keys periodically** (requires regenerating all licenses)

### Security Features

-   **Key Derivation**: All cryptographic keys are derived from the master key using PBKDF2 with 10,000 iterations
-   **Unique Salts**: Each key type (serial, encryption) uses a unique salt derived from the master key
-   **Deterministic**: Same master key always produces the same derived keys (ensures compatibility)
-   **Isolation**: Different master keys produce completely different derived keys
-   **No Hardcoded Secrets**: All salts and keys are dynamically generated from your master key

## Behaviors

-   **Filename format**: `<product_name>.license` (spaces and special characters become underscores)
-   **Example**: Creating a license for "My Great App" produces `My_Great_App.license`

### Constructor Methods

| Method                      | Description                               | Use Case               |
| --------------------------- | ----------------------------------------- | ---------------------- |
| `NewManager()`              | Uses environment variables or default key | CLI tools, development |
| `NewManagerWithKey(string)` | Uses provided string as master key        | Production client apps |

### Environment Variables

| Variable                | Default           | Description                                         |
| ----------------------- | ----------------- | --------------------------------------------------- |
| `LICENSE_MASTER_KEY`    | _(optional)_      | Master encryption key (only used by `NewManager()`) |
| `LICENSE_DEFAULT_DAYS`  | `30`              | Default license duration when not specified         |
| `LICENSE_LIFETIME_DAYS` | `99999`           | Number of days that represents a lifetime license   |
| `LICENSE_DIR`           | Current directory | Directory to store and search for license files     |

### Master Key Recommendations for Client Applications

-   **No .env files**: Use `NewManagerWithKey()` to avoid needing .env files on client machines
-   **Secure storage**: Retrieve keys from Windows Registry, macOS Keychain, or encrypted configs
-   **Obfuscation**: Use obfuscation techniques as an additional layer of security
-   **Remote retrieval**: Fetch keys from your secure API during startup

## Architecture

```
license-manager/
├── cmd/license-manager/     # CLI application
└── pkg/
    ├── license/            # Core license management
    ├── crypto/             # Cryptographic operations
    ├── hardware/           # PC ID generation
    └── config/             # Configuration management
```

### API Reference

### License Manager

```go
// Create a new license manager
manager, err := license.NewManager()

// Create a new license
req := license.CreateLicenseRequest{
    ProductName: "My Product",
    MaxDays:     30,
    IsLifetime:  false,
}
license, err := manager.Create(req)

// Validate license for a specific product (updates usage)
result, err := manager.Validate("My Product")

// Get license info for a specific product (read-only)
info, err := manager.GetInfo("My Product")

// View raw license data for a specific product
license, err := manager.View("My Product")

// Revoke license for a specific product
err := manager.Revoke("My Product")



// Get PC ID
pcid := manager.GetPCID()
```

### Data Structures

```go
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
	RunCount      int    // Matches field name in License struct
	FirstRunDate  string // Matches field name in License struct
	LastUsedDate  string // Matches field name in License struct
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

```

## Platform Support

| Platform | PC ID Sources                            |
| -------- | ---------------------------------------- |
| Windows  | CPU ID, Motherboard Serial, Machine GUID |
| Linux    | Machine ID, CPU Info, MAC Address        |
| macOS    | Hardware UUID, Serial Number             |

## Security Features

### Encryption

-   **Algorithm**: AES-256-GCM
-   **Key Derivation**: SHA-256 based key derivation from master key
-   **Nonce**: Cryptographically secure random nonce per encryption

### Anti-Tampering

-   **Hardware Binding**: Licenses tied to specific hardware
-   **Serial Validation**: Cryptographic serial number verification
-   **Time Rollback Detection**: Prevents system clock manipulation
-   **Encrypted Storage**: License files are encrypted at rest

### License Validation Process

1. Decrypt license file
2. Verify PC ID matches current hardware
3. Validate cryptographic serial number
4. Check expiration and usage limits
5. Detect time rollback attempts
6. Update usage tracking

## Examples

Here are some common usage patterns:

-   Basic license validation for a specific product
-   Creating licenses programmatically

-   Getting license information
-   Advanced configuration with environment variables

## Building

```bash
# Build CLI tool
cd cmd/license-manager
go build

# Build with version info
go build -ldflags "-X main.version=1.0.0"

# Cross-compilation examples
GOOS=windows GOARCH=amd64 go build -o license-manager.exe
GOOS=linux GOARCH=amd64 go build -o license-manager-linux
GOOS=darwin GOARCH=amd64 go build -o license-manager-macos
```

## Testing

```bash
# Run all tests
go test ./pkg/license
```

## Contributing

1. Fork the repository
2. Create a branch
3. Run your tests
4. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
