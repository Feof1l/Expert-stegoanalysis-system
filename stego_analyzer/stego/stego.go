package stego

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

type AnalyzeResponse struct {
	StegoProb float64 `json:"stego_prob"`
	Result    string  `json:"result"`
}

type PredictionResponse struct {
	StegoProb float64 `json:"stego_prob"`
}

func ClassifyInNN(features Features) (float64, error) {
	data := []float64{
		features.LSBTransitions,
		features.BitRun00,
		features.BitRun01,
		features.BitRun10,
		features.BitRun11,
		features.NeighborDiff,
		features.ChiSquare,
		features.EntropyLSB,
		features.R,
		features.S,
		features.Rm,
		features.Sm,
		features.RmR,
		features.SmS,
	}

	reqBody, err := json.Marshal(data)
	if err != nil {
		return 0, err
	}

	resp, err := http.Post("http://localhost:8000/predict", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("server returned %d", resp.StatusCode)
	}

	var result PredictionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	return result.StegoProb, nil
}

func ProcessDataset(rootDir string, outFile string) error {
	f, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	header := []string{
		"path", "label",
		"lsb_transitions",
		"bit_run_00", "bit_run_01", "bit_run_10", "bit_run_11",
		"neighbor_diff",
		"chi_square",
		"entropy_lsb",
		"r", "s", "rm", "sm", "rm_r", "sm_s",
	}
	w.Write(header)

	for _, dir := range []string{"clean", "33%", "50%", "66%", "100%"} {
		label := 0.0
		if dir != "clean" {
			switch dir {
			case "33%":
				label = 0.33
			case "50%":
				label = 0.50
			case "66%":
				label = 0.66
			case "100%":
				label = 1.00
			}
		}

		filepath.Walk(filepath.Join(rootDir, dir), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return nil
			}

			pixels, err := LoadPixels(path)
			if err != nil {
				return nil
			}

			features := Extract(pixels)

			record := []string{
				path,
				fmt.Sprintf("%.2f", label),
				fmt.Sprintf("%.6f", features.LSBTransitions),
				fmt.Sprintf("%.6f", features.BitRun00),
				fmt.Sprintf("%.6f", features.BitRun01),
				fmt.Sprintf("%.6f", features.BitRun10),
				fmt.Sprintf("%.6f", features.BitRun11),
				fmt.Sprintf("%.6f", features.NeighborDiff),
				fmt.Sprintf("%.6f", features.ChiSquare),
				fmt.Sprintf("%.6f", features.EntropyLSB),
				fmt.Sprintf("%.6f", features.R),
				fmt.Sprintf("%.6f", features.S),
				fmt.Sprintf("%.6f", features.Rm),
				fmt.Sprintf("%.6f", features.Sm),
				fmt.Sprintf("%.6f", features.RmR),
				fmt.Sprintf("%.6f", features.SmS),
			}
			w.Write(record)
			return nil
		})
	}

	return nil
}
