package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
	"time"
)

// Report holds the data for the final report.
type Report struct {
	Metadata ReportMetadata `json:"metadata"`
	Chart    ChartReport    `json:"chart"`
	Images   []ImageReport  `json:"images"`
	Summary  ReportSummary  `json:"summary"`
}

// ReportMetadata holds metadata about the report generation
type ReportMetadata struct {
	GeneratedAt time.Time `json:"generated_at"`
	Version     string    `json:"version"`
	DryRun      bool      `json:"dry_run,omitempty"`
}

// ChartReport holds the data for the chart section of the report.
type ChartReport struct {
	Name   string `json:"name"`
	Pushed bool   `json:"pushed"`
}

// ImageReport holds the data for each image in the report.
type ImageReport struct {
	Name                 string `json:"name"`
	Pushed               bool   `json:"pushed"`
	VulnerabilitiesFound int    `json:"vulnerabilities_found,omitempty"`
	Patched              bool   `json:"patched,omitempty"`
	Signed               bool   `json:"signed,omitempty"`
}

// ReportSummary holds summary statistics
type ReportSummary struct {
	TotalImages          int  `json:"total_images"`
	ImagesPushed         int  `json:"images_pushed"`
	ImagesSkipped        int  `json:"images_skipped"`
	TotalVulnerabilities int  `json:"total_vulnerabilities"`
	ChartPushed          bool `json:"chart_pushed"`
}

// GenerateReport generates the report in the specified format.
func (r *Report) GenerateReport(format, file string, config *Config) error {
	// Initialize metadata
	r.Metadata = ReportMetadata{
		GeneratedAt: time.Now(),
		Version:     Version,
		DryRun:      config.DryRun,
	}

	// Calculate summary statistics
	r.calculateSummary()

	switch format {
	case "table":
		return r.generateTableReport(file)
	case "json":
		return r.generateJSONReport(file)
	default:
		return fmt.Errorf("unsupported report format: %s", format)
	}
}

// calculateSummary calculates and populates the summary statistics
func (r *Report) calculateSummary() {
	r.Summary.TotalImages = len(r.Images)
	r.Summary.ChartPushed = r.Chart.Pushed

	for _, img := range r.Images {
		if img.Pushed {
			r.Summary.ImagesPushed++
		} else {
			r.Summary.ImagesSkipped++
		}
		r.Summary.TotalVulnerabilities += img.VulnerabilitiesFound
	}
}

func (r *Report) generateTableReport(file string) error {
	var writer io.Writer
	var f *os.File
	var err error

	if file != "" {
		f, err = os.Create(file)
		if err != nil {
			return fmt.Errorf("failed to create report file: %w", err)
		}
		defer f.Close()
		writer = f
	} else {
		writer = os.Stdout
	}

	w := tabwriter.NewWriter(writer, 0, 0, 2, ' ', 0)

	// Report header with metadata
	fmt.Fprintf(w, "=== HELM SECURE IMPORT REPORT ===\n")
	fmt.Fprintf(w, "Generated at:\t%s\n", r.Metadata.GeneratedAt.Format("2006-01-02 15:04:05 MST"))
	fmt.Fprintf(w, "Version:\t%s\n", r.Metadata.Version)
	if r.Metadata.DryRun {
		fmt.Fprintf(w, "Mode:\tDRY RUN\n")
	}
	fmt.Fprintf(w, "\n")

	// Summary statistics
	fmt.Fprintf(w, "=== SUMMARY ===\n")
	fmt.Fprintf(w, "Chart pushed:\t%v\n", r.Summary.ChartPushed)
	fmt.Fprintf(w, "Total images:\t%d\n", r.Summary.TotalImages)
	fmt.Fprintf(w, "Images pushed:\t%d\n", r.Summary.ImagesPushed)
	fmt.Fprintf(w, "Images skipped:\t%d\n", r.Summary.ImagesSkipped)
	if r.Summary.TotalVulnerabilities > 0 {
		fmt.Fprintf(w, "Total vulnerabilities found:\t%d\n", r.Summary.TotalVulnerabilities)
	}
	fmt.Fprintf(w, "\n")

	// Detailed artifact list
	fmt.Fprintf(w, "=== ARTIFACTS ===\n")
	fmt.Fprintln(w, "ARTIFACT\tPUSHED\tVULNERABILITIES\tSTATUS")

	// Chart entry
	status := "✅ Pushed"
	if !r.Chart.Pushed {
		status = "⏭️ Skipped (already exists)"
	}
	fmt.Fprintf(w, "%s\t%v\t-\t%s\n", r.Chart.Name, r.Chart.Pushed, status)

	// Image entries
	for _, img := range r.Images {
		vulnText := "-"
		if img.VulnerabilitiesFound > 0 {
			vulnText = fmt.Sprintf("%d", img.VulnerabilitiesFound)
		}

		status := "✅ Pushed"
		if !img.Pushed {
			status = "⏭️ Skipped (already exists)"
		}
		if img.Patched {
			status += " & 🔧 Patched"
		}
		if img.Signed {
			status += " & ✍️ Signed"
		}

		fmt.Fprintf(w, "%s\t%v\t%s\t%s\n", img.Name, img.Pushed, vulnText, status)
	}

	return w.Flush()
}

func (r *Report) generateJSONReport(file string) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report to JSON: %w", err)
	}

	if file != "" {
		return os.WriteFile(file, data, 0644)
	}

	fmt.Println(string(data))
	return nil
}
