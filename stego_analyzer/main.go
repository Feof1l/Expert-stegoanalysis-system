package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

type PredictionResponse struct {
	StegoProb float64 `json:"stego_prob"`
}

func classifyInNN(features Features) (float64, error) {
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

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: analyzer <image_path>")
	}

	path := os.Args[1]

	pixels, err := LoadPixels(path)
	if err != nil {
		log.Fatalf("load image: %v", err)
	}

	features := Extract(pixels)

	prob, err := classifyInNN(features)
	if err != nil {
		log.Fatalf("classify: %v", err)
	}

	fmt.Printf("Image: %s\n", path)
	fmt.Printf("Stego probability: %.4f\n", prob)
	if prob > 0.5 {
		fmt.Println("Result: likely contains hidden container")
	} else {
		fmt.Println("Result: likely clean")
	}

	// err := ProcessDataset("/home/vasilov/me/stego_analyze/expert steanalysis system/stego_analyzer/images", "features.csv")
	// if err != nil {
	// 	log.Printf("%+v", err)
	// }
}

// пример: обработка всех файлов в папке
func ProcessDataset(rootDir string, outFile string) error {
	f, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// заголовок
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

	// обход папок
	for _, dir := range []string{"clean", "33%", "50%", "66%", "100%"} {
		label := 0.0
		if dir != "clean" {
			// можно сделать метку 1.0 (бинарно) или 0.33/0.5/0.66/1.0
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
