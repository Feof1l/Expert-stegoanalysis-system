package stego

import (
	"math"
)

type Features struct {
	LSBTransitions float64
	BitRun00       float64
	BitRun01       float64
	BitRun10       float64
	BitRun11       float64
	NeighborDiff   float64
	ChiSquare      float64
	EntropyLSB     float64
	R, S, Rm, Sm   float64
	RmR, SmS       float64
}

func CountLSBTransitions(pixels []uint8) float64 {
	var transitions int
	var prevLSB byte

	for i, p := range pixels {
		lsb := p & 1
		if i > 0 && lsb != prevLSB {
			transitions++
		}
		prevLSB = lsb
	}

	if len(pixels) == 0 {
		return 0
	}
	return float64(transitions) / float64(len(pixels))
}

func BitRunStats(pixels []uint8) (float64, float64, float64, float64) {
	counts := map[string]int{
		"00": 0, "01": 0, "10": 0, "11": 0,
	}

	var prev byte
	for i, p := range pixels {
		lsb := p & 1
		if i > 0 {
			key := string([]byte{prev + '0', lsb + '0'})
			counts[key]++
		}
		prev = lsb
	}

	total := float64(len(pixels) - 1)
	if total == 0 {
		return 0, 0, 0, 0
	}
	return float64(counts["00"]) / total,
		float64(counts["01"]) / total,
		float64(counts["10"]) / total,
		float64(counts["11"]) / total
}

func NeighborLSBDiff(pixels []uint8) float64 {
	if len(pixels) < 2 {
		return 0
	}
	var diffCount int
	for i := 1; i < len(pixels); i++ {
		lsb1 := pixels[i-1] & 1
		lsb2 := pixels[i] & 1
		if lsb1 != lsb2 {
			diffCount++
		}
	}
	return float64(diffCount) / float64(len(pixels)-1)
}

func ChiSquareLSB(pixels []uint8) float64 {
	if len(pixels) == 0 {
		return 0
	}
	observed := [2]int{0, 0}
	for _, p := range pixels {
		observed[p&1]++
	}

	n := float64(len(pixels))
	expected := n / 2.0
	var chi2 float64
	for _, o := range observed {
		diff := float64(o) - expected
		chi2 += diff * diff / expected
	}
	return chi2
}

func EntropyLSB(pixels []uint8) float64 {
	if len(pixels) == 0 {
		return 0
	}
	counts := [2]int{0, 0}
	for _, p := range pixels {
		counts[p&1]++
	}

	var entropy float64
	for _, c := range counts {
		if c > 0 {
			p := float64(c) / float64(len(pixels))
			entropy -= p * math.Log2(p)
		}
	}
	return entropy
}

func Smoothness(group []uint8) float64 {
	var s float64
	for i := 1; i < len(group); i++ {
		s += math.Abs(float64(group[i]) - float64(group[i-1]))
	}
	return s
}

func ClassifyGroup(group []uint8, threshold float64) byte {
	s := Smoothness(group)
	if s < threshold {
		return 'R'
	}
	return 'S'
}

func RSFeatures(pixels []uint8) (r, s, rm, sm float64) {
	const groupSize = 2
	threshold := 10.0

	var rCount, sCount, rmCount, smCount int

	for i := 0; i+groupSize <= len(pixels); i += groupSize {
		group := pixels[i : i+groupSize]

		// Original
		typ := ClassifyGroup(group, threshold)
		if typ == 'R' {
			rCount++
		} else {
			sCount++
		}

		// Flipped LSB
		flipped := make([]uint8, groupSize)
		for j, p := range group {
			flipped[j] = p ^ 1
		}
		typFlip := ClassifyGroup(flipped, threshold)
		if typFlip == 'R' {
			rmCount++
		} else {
			smCount++
		}
	}

	total := float64(rCount + sCount)
	if total == 0 {
		return 0, 0, 0, 0
	}
	r = float64(rCount) / total
	s = float64(sCount) / total
	rm = float64(rmCount) / total
	sm = float64(smCount) / total
	return
}

func Extract(pixels []uint8) Features {
	r, s, rm, sm := RSFeatures(pixels)
	br00, br01, br10, br11 := BitRunStats(pixels)

	return Features{
		LSBTransitions: CountLSBTransitions(pixels),
		BitRun00:       br00,
		BitRun01:       br01,
		BitRun10:       br10,
		BitRun11:       br11,
		NeighborDiff:   NeighborLSBDiff(pixels),
		ChiSquare:      ChiSquareLSB(pixels),
		EntropyLSB:     EntropyLSB(pixels),
		R:              r,
		S:              s,
		Rm:             rm,
		Sm:             sm,
		RmR:            rm - r,
		SmS:            sm - s,
	}
}
