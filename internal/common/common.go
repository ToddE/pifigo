package common // Package common for internal use

import (
	"bytes"
	"fmt"
	"log"
	"os"     // For os.ReadFile, os.WriteFile in EnsureHostname
	"os/exec"
	"strings"
	"time" // For GenerateClaimCode pseudo-randomness

	// uuid "github.com/google/uuid" // For GenerateDeviceId
)

// ExecCommand executes a shell command with sudo, logs it, and returns stdout or error.
func ExecCommand(name string, args ...string) (string, error) {
	cmdArgs := []string{"-n", name} // -n for no password prompt for sudo
	cmdArgs = append(cmdArgs, args...)
	
	cmd := exec.Command("sudo", cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	log.Printf("Executing command: sudo %s %s", name, strings.Join(args, " "))
	err := cmd.Run()

	stdoutStr := stdout.String()
	stderrStr := stderr.String()

	if err != nil {
		log.Printf("Command '%s %s' failed: %v", name, strings.Join(args, " "), err)
		if stderrStr != "" {
			log.Printf("Stderr: %s", stderrStr)
		}
		return "", fmt.Errorf("command '%s %s' failed: %v, Stderr: %s", name, strings.Join(args, " "), err, stderrStr)
	}
	if stderrStr != "" {
		log.Printf("Command '%s %s' produced stderr: %s", name, strings.Join(args, " "), stderrStr)
	}
	return stdoutStr, nil
}

// GenerateClaimCode creates a short, random alphanumeric string for a session.
// Note: This is simple pseudo-randomness for a claim code. For cryptographic security, use crypto/rand.
func GenerateClaimCode() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const codeLen = 6 // e.g., 6 characters for simplicity

	b := make([]byte, codeLen)
	for i := range b {
		// Using current time as a source of "randomness" for simplicity in a claim code.
		// For cryptographic-grade randomness, use crypto/rand.
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))] 
	}
	return string(b)
}

// EnsureHostname sets the system hostname.
// This function is included as a generic utility that might be needed in a common library,
// though its primary caller would be `netconfig` or directly `main` of the AP/CP service
// that manages the mDNS hostname.
func EnsureHostname(hostname string) error {
	currentHostname, err := ExecCommand("hostname")
	if err != nil || strings.TrimSpace(currentHostname) != hostname {
		log.Printf("Setting hostname to: %s", hostname)
		_, err := ExecCommand("hostnamectl", "set-hostname", hostname)
		if err != nil {
			return fmt.Errorf("failed to set hostname: %v", err)
		}
		// Also update /etc/hosts for localhost to resolve to new hostname (optional, but good practice)
		hostsContent, err := os.ReadFile("/etc/hosts")
		if err != nil {
			log.Printf("Warning: Could not read /etc/hosts: %v", err)
		} else {
			newHostsContent := strings.ReplaceAll(string(hostsContent), "127.0.1.1 "+strings.TrimSpace(currentHostname), "127.0.1.1 "+hostname)
			os.WriteFile("/etc/hosts", []byte(newHostsContent), 0644)
		}
		
	} else {
		log.Printf("Hostname already set to: %s", hostname)
	}
	return nil
}