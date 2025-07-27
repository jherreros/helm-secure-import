package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
)

// Report holds the data for the final report.

type Report struct {
	Chart  ChartReport   `json:"chart"`
	Images []ImageReport `json:"images"`
}

// ChartReport holds the data for the chart section of the report.

type ChartReport struct {
	Name   string `json:"name"`
	Pushed bool   `json:"pushed"`
}

// ImageReport holds the data for each image in the report.

type ImageReport struct {
	Name   string `json:"name"`
	Pushed bool   `json:"pushed"`
}

// GenerateReport generates the report in the specified format.
func (r *Report) GenerateReport(format, file string) error {
	switch format {
	case "table":
		return r.generateTableReport(file)
	case "json":
		return r.generateJSONReport(file)
	default:
		return fmt.Errorf("unsupported report format: %s", format)
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
	fmt.Fprintln(w, "ARTIFACT\tPUSHED")
	fmt.Fprintf(w, "%s\t%v\n", r.Chart.Name, r.Chart.Pushed)
	for _, img := range r.Images {
		fmt.Fprintf(w, "%s\t%v\n", img.Name, img.Pushed)
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