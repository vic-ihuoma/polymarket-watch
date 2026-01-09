package output

import (
	"encoding/json"
	"errors"
	"io"
	"os"

	"github.com/victorihuoma/polymarket-watch/internal/discovery"
	"github.com/victorihuoma/polymarket-watch/internal/models"
)

// JSONOutput formats scan results as JSON.
type JSONOutput struct {
	writer      io.Writer
	prettyPrint bool
}

// JSONOption is a functional option for configuring JSONOutput.
type JSONOption func(*JSONOutput)

// NewJSONOutput creates a new JSONOutput with the given options.
func NewJSONOutput(opts ...JSONOption) *JSONOutput {
	jo := &JSONOutput{
		writer:      os.Stdout,
		prettyPrint: true,
	}

	for _, opt := range opts {
		opt(jo)
	}

	return jo
}

// WithJSONWriter sets the output writer for JSON output.
func WithJSONWriter(w io.Writer) JSONOption {
	return func(jo *JSONOutput) {
		jo.writer = w
	}
}

// WithPrettyPrint enables or disables pretty-printed JSON output.
func WithPrettyPrint(enabled bool) JSONOption {
	return func(jo *JSONOutput) {
		jo.prettyPrint = enabled
	}
}

// Print outputs a single scan report as JSON.
func (jo *JSONOutput) Print(report *models.ScanReport) error {
	if report == nil {
		return errors.New("report cannot be nil")
	}

	return jo.encode(report)
}

// PrintSummary outputs multiple scan reports as a JSON array.
func (jo *JSONOutput) PrintSummary(reports []*models.ScanReport) error {
	if reports == nil {
		reports = []*models.ScanReport{}
	}

	return jo.encode(reports)
}

// encode marshals the given value to JSON and writes it to the writer.
func (jo *JSONOutput) encode(v interface{}) error {
	var data []byte
	var err error

	if jo.prettyPrint {
		data, err = json.MarshalIndent(v, "", "  ")
	} else {
		data, err = json.Marshal(v)
	}

	if err != nil {
		return err
	}

	data = append(data, '\n')
	_, err = jo.writer.Write(data)
	return err
}

// WriteReportToFile writes a single scan report to a JSON file.
func WriteReportToFile(report *models.ScanReport, filePath string) error {
	if report == nil {
		return errors.New("report cannot be nil")
	}

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	jo := NewJSONOutput(WithJSONWriter(file))
	return jo.Print(report)
}

// WriteSummaryToFile writes multiple scan reports to a JSON file.
func WriteSummaryToFile(reports []*models.ScanReport, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	jo := NewJSONOutput(WithJSONWriter(file))
	return jo.PrintSummary(reports)
}

// PrintDiscoveryResult outputs a discovery result as JSON.
func (jo *JSONOutput) PrintDiscoveryResult(result *discovery.DiscoveryResult) error {
	if result == nil {
		return errors.New("discovery result cannot be nil")
	}

	return jo.encode(result)
}

// WriteDiscoveryToFile writes a discovery result to a JSON file.
func WriteDiscoveryToFile(result *discovery.DiscoveryResult, filePath string) error {
	if result == nil {
		return errors.New("discovery result cannot be nil")
	}

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	jo := NewJSONOutput(WithJSONWriter(file))
	return jo.PrintDiscoveryResult(result)
}
