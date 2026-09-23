package service

import (
	"fmt"
	"os"
	"path/filepath"
)

// InitService is responsible for initializing the quality.yml file with intelligent analysis.
type InitService struct {
	detector  *LanguageDetector
	generator *TemplateGenerator
}

// NewInitService creates a new InitService with intelligent project analysis.
func NewInitService() *InitService {
	projectPath, _ := os.Getwd() // Default to current directory
	return &InitService{
		detector:  NewLanguageDetector(projectPath),
		generator: NewTemplateGenerator(),
	}
}

// NewInitServiceWithPath creates a new InitService for a specific project path.
func NewInitServiceWithPath(projectPath string) *InitService {
	return &InitService{
		detector:  NewLanguageDetector(projectPath),
		generator: NewTemplateGenerator(),
	}
}

// Init creates the quality.yml file with intelligent project analysis.
func (s *InitService) Init() error {
	return s.InitWithOptions(InitOptions{
		OutputPath: "quality.yml",
		Verbose:    false,
	})
}

// InitOptions provides configuration options for initialization
type InitOptions struct {
	OutputPath string
	Verbose    bool
	Force      bool // Overwrite existing file
}

// InitWithOptions creates the quality.yml file with specified options.
func (s *InitService) InitWithOptions(opts InitOptions) error {
	// Check if file already exists
	if !opts.Force {
		if _, err := os.Stat(opts.OutputPath); err == nil {
			return fmt.Errorf("quality.yml already exists. Use --force to overwrite")
		}
	}

	// Detect project structure
	if opts.Verbose {
		fmt.Println("🔍 Analyzing project structure...")
	}

	structure, err := s.detector.DetectProjectStructure()
	if err != nil {
		return fmt.Errorf("failed to analyze project structure: %w", err)
	}

	if opts.Verbose {
		s.printDetectedStructure(structure)
	}

	// Generate template based on detected structure
	if opts.Verbose {
		fmt.Println("📝 Generating quality.yml template...")
	}

	template := s.generator.GenerateTemplate(structure)

	// Write to file
	err = os.WriteFile(opts.OutputPath, []byte(template), 0644)
	if err != nil {
		return fmt.Errorf("failed to write quality.yml: %w", err)
	}

	if opts.Verbose {
		fmt.Printf("✅ Successfully created %s\n", opts.OutputPath)
		s.printNextSteps()
	}

	return nil
}

// GetProjectAnalysis returns the detected project structure without creating a file
func (s *InitService) GetProjectAnalysis() (*ProjectStructure, error) {
	return s.detector.DetectProjectStructure()
}

// GeneratePreview returns the generated quality.yml content without writing to disk
func (s *InitService) GeneratePreview() (string, error) {
	structure, err := s.detector.DetectProjectStructure()
	if err != nil {
		return "", fmt.Errorf("failed to analyze project structure: %w", err)
	}

	template := s.generator.GenerateTemplate(structure)
	return template, nil
}

// printDetectedStructure prints the detected project structure for verbose output
func (s *InitService) printDetectedStructure(structure *ProjectStructure) {
	fmt.Println("🎯 Detected project components:")

	if len(structure.Languages) > 0 {
		fmt.Printf("   Languages: %v\n", structure.Languages)
	}

	if len(structure.Frameworks) > 0 {
		fmt.Printf("   Frameworks: %v\n", structure.Frameworks)
	}

	if len(structure.Tools) > 0 {
		fmt.Printf("   Existing Tools: %v\n", structure.Tools)
	}

	// Show detected files
	for lang, files := range structure.Structure {
		if len(files) > 0 {
			fmt.Printf("   %s files: %d detected\n", lang, len(files))
			if len(files) <= 3 {
				for _, file := range files {
					relPath, _ := filepath.Rel(s.detector.projectPath, file)
					fmt.Printf("     - %s\n", relPath)
				}
			} else {
				relPath, _ := filepath.Rel(s.detector.projectPath, files[0])
				fmt.Printf("     - %s (and %d more)\n", relPath, len(files)-1)
			}
		}
	}
}

// printNextSteps prints helpful next steps for the user
func (s *InitService) printNextSteps() {
	fmt.Println("\n🚀 Next steps:")
	fmt.Println("   1. Review and customize the generated quality.yml")
	fmt.Println("   2. Set up git hooks: ./quality-gate --install")
	fmt.Println("   3. Test the configuration: ./quality-gate pre-commit")
	fmt.Println("   4. Enforce it: add the quality-gate GitHub Action as a required check")
	fmt.Println("\n💡 Tip: Use './quality-gate --fix' to automatically fix formatting issues")
	fmt.Println("💡 Tip: Run 'quality-gate --install' from your project bootstrap (package.json \"prepare\", make setup) so every clone gets the hooks")
}
