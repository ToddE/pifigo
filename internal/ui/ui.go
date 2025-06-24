package ui

import (
	"html/template"
	"log"
	"os"     // For os.Stat
	"path/filepath"
	"fmt"    // For fmt.Errorf

    "github.com/ToddE/pifigo/internal/config" 
)

// TemplateData combines all data needed for the HTML template.
type TemplateData struct {
	Networks        []string
	UIConfig        config.UiConfig
	Lang            config.LanguageStrings
	SuccessMessage  string
	ErrorMessage    string
	DeviceID        string
	ClaimCode       string
	LocalNodeSetupLink string
	Message         string
}

// GetTemplate parses and returns the appropriate HTML template.
// It requires AppPaths to know where to find external custom templates/assets.
func GetTemplate(customTemplatePath string, embeddedTemplateContent string, appPaths config.AppPaths) (*template.Template, error) { // ADD appPaths argument
	var t *template.Template
	var err error

	// 1. Check for external custom template first
	if customTemplatePath != "" {
		fullPath := filepath.Join(appPaths.AssetsDirPath, customTemplatePath) // Use appPaths.AssetsDirPath
		if _, statErr := os.Stat(fullPath); statErr == nil {
			t, err = template.ParseFiles(fullPath) // Parse from disk
			if err == nil {
				log.Printf("UI: Using custom template from: %s", fullPath)
				return t, nil
			}
			log.Printf("UI: Error parsing custom template %s: %v. Falling back to embedded.", fullPath, err)
		} else {
			log.Printf("UI: Custom template %s not found at %s. Falling back to embedded.", customTemplatePath, fullPath)
		}
	}

	// 2. Fallback to embedded/default template
	t, err = template.New("index.html").Parse(embeddedTemplateContent)
	if err != nil {
		return nil, fmt.Errorf("UI: failed to parse embedded template content: %v", err)
	}
	log.Println("UI: Using default embedded template.")
	return t, nil
}