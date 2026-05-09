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

type FeatureInfo struct {
	Name      string  `json:"name"`
	Value     float64 `json:"value"`
	MinNormal float64 `json:"min_normal"`
	MaxNormal float64 `json:"max_normal"`
	IsAnomaly bool    `json:"is_anomaly"`
}

type AnalyzeResponse struct {
	StegoProb float64       `json:"stego_prob"`
	Result    string        `json:"result"`
	Features  []FeatureInfo `json:"features"`
}

type PredictionResponse struct {
	StegoProb float64 `json:"stego_prob"`
}

// Normal ranges thresholds for steganalysis features
// Based on typical values distributions for clean images
var featureNormalRanges = []struct {
	Name string
	Min  float64
	Max  float64
}{
	{"LSB Transition", 0.45, 0.55},
	{"Bit Run 00", 0.20, 0.35},
	{"Bit Run 01", 0.20, 0.35},
	{"Bit Run 10", 0.20, 0.35},
	{"Bit Run 11", 0.20, 0.35},
	{"Neighbor Diff", 0.45, 0.55},
	{"Chi-Square", 0.0, 3.0},
	{"Entropy LSB", 0.95, 1.0},
	{"R (Regular)", 0.40, 0.60},
	{"S (Singular)", 0.40, 0.60},
	{"Rm (Regular flipped)", 0.40, 0.60},
	{"Sm (Singular flipped)", 0.40, 0.60},
	{"Rm-R", -0.10, 0.10},
	{"Sm-S", -0.10, 0.10},
}

func GetFeaturesWithAnalysisSlice(featureValues []float64) []FeatureInfo {
	features := make([]FeatureInfo, 14)
	for i := 0; i < 14 && i < len(featureValues); i++ {
		features[i] = FeatureInfo{
			Name:      featureNormalRanges[i].Name,
			Value:     featureValues[i],
			MinNormal: featureNormalRanges[i].Min,
			MaxNormal: featureNormalRanges[i].Max,
			IsAnomaly: featureValues[i] < featureNormalRanges[i].Min || featureValues[i] > featureNormalRanges[i].Max,
		}
	}
	return features
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
