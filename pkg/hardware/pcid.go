package hardware

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os/exec"
	"runtime"
	"slices"
	"strings"
)

// PCIDGenerator handles hardware identification
type PCIDGenerator struct{}

// NewPCIDGenerator creates a new PC ID generator
func NewPCIDGenerator() *PCIDGenerator {
	return &PCIDGenerator{}
}

// Generate creates a unique PC ID based on hardware characteristics
// This uses the exact same logic as the original GeneratePCId function
func (p *PCIDGenerator) Generate() (string, error) {
	var components []string

	switch runtime.GOOS {
	case "windows":
		if cpu := p.runCmd("wmic", "cpu", "get", "ProcessorId", "/format:list"); cpu != "" {
			if val := p.extractValue(cpu, "ProcessorId"); val != "" {
				components = append(components, "cpu:"+val)
			}
		}
		if mb := p.runCmd("wmic", "baseboard", "get", "SerialNumber", "/format:list"); mb != "" {
			if val := p.extractValue(mb, "SerialNumber"); val != "" {
				components = append(components, "mb:"+val)
			}
		}
		if guid := p.runCmd("reg", "query", "HKLM\\SOFTWARE\\Microsoft\\Cryptography", "/v", "MachineGuid"); guid != "" {
			lines := strings.SplitSeq(guid, "\n")
			for line := range lines {
				if strings.Contains(line, "MachineGuid") {
					parts := strings.Fields(line)
					if len(parts) >= 3 {
						components = append(components, "guid:"+parts[2])
						break
					}
				}
			}
		}
	case "linux":
		if machineId := p.runCmd("cat", "/etc/machine-id"); machineId != "" {
			components = append(components, "machine:"+strings.TrimSpace(machineId))
		}
		if cpuInfo := p.runCmd("cat", "/proc/cpuinfo"); cpuInfo != "" {
			lines := strings.SplitSeq(cpuInfo, "\n")
			for line := range lines {
				if strings.Contains(line, "processor") && strings.Contains(line, "0") {
					components = append(components, "cpu:"+strings.TrimSpace(line))
					break
				}
			}
		}
		if mac := p.runCmd("cat", "/sys/class/net/*/address"); mac != "" {
			macs := strings.Split(strings.TrimSpace(mac), "\n")
			if len(macs) > 0 && macs[0] != "00:00:00:00:00:00" {
				components = append(components, "mac:"+macs[0])
			}
		}
	case "darwin":
		if uuid := p.runCmd("system_profiler", "SPHardwareDataType"); uuid != "" {
			lines := strings.SplitSeq(uuid, "\n")
			for line := range lines {
				if strings.Contains(line, "Hardware UUID:") {
					parts := strings.Split(line, ":")
					if len(parts) >= 2 {
						components = append(components, "uuid:"+strings.TrimSpace(strings.Join(parts[1:], ":")))
						break
					}
				}
			}
		}
		if serial := p.runCmd("system_profiler", "SPHardwareDataType"); serial != "" {
			lines := strings.SplitSeq(serial, "\n")
			for line := range lines {
				if strings.Contains(line, "Serial Number") {
					parts := strings.Split(line, ":")
					if len(parts) >= 2 {
						components = append(components, "serial:"+strings.TrimSpace(parts[1]))
						break
					}
				}
			}
		}
	}

	if len(components) == 0 {
		return "", fmt.Errorf("could not generate PC ID - no hardware identifiers found")
	}

	combined := strings.Join(components, "|")
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:16]), nil
}

// runCmd executes a system command and returns its output
func (p *PCIDGenerator) runCmd(name string, args ...string) string {
	cmd := exec.Command(name, args...)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(output)
}

// extractValue extracts a value from command output in key=value format
func (p *PCIDGenerator) extractValue(output, key string) string {
	lines := strings.SplitSeq(output, "\n")
	for line := range lines {
		if strings.Contains(line, key+"=") {
			parts := strings.Split(line, "=")
			if len(parts) >= 2 {
				value := strings.TrimSpace(parts[1])
				if value != "" && value != "To be filled by O.E.M." {
					return value
				}
			}
		}
	}
	return ""
}

// GetSupportedPlatforms returns the list of supported operating systems
func (p *PCIDGenerator) GetSupportedPlatforms() []string {
	return []string{"windows", "linux", "darwin"}
}

// IsSupported checks if the current platform is supported
func (p *PCIDGenerator) IsSupported() bool {
	supported := p.GetSupportedPlatforms()
	return slices.Contains(supported, runtime.GOOS)
}
