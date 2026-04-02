package main

import (
	"github.com/agnivade/levenshtein"
)

func ClusterDescriptions(descriptions []string, threshold float64) [][]string {
	clusterOf := make(map[string]int)
	for i, desc := range descriptions {
		clusterOf[desc] = i
	}

	for _, desc1 := range descriptions {
		for _, desc2 := range descriptions {
			if clusterOf[desc1] == clusterOf[desc2] {
				continue
			}
			similarityScore := similarity(desc1, desc2)
			if similarityScore >= threshold {
				clusterOf[desc2] = clusterOf[desc1]
			}
		}
	}
	collected := make(map[int][]string)
	for desc, cluster := range clusterOf {
		collected[cluster] = append(collected[cluster], desc)
	}
	results := make([][]string, 0, len(collected))
	for _, cluster := range collected {
		results = append(results, cluster)
	}
	return results
}

func similarity(a, b string) float64 {
	return max(editSimilarity(a, b), prefixSimilarity(a, b))
}

func editSimilarity(a, b string) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1.0 // identical
	}
	distance := levenshtein.ComputeDistance(a, b)
	return 1 - float64(distance)/float64(max(len(a), len(b)))
}

func prefixSimilarity(a, b string) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1.0 // identical
	}
	minLen := min(len(a), len(b))
	matches := 0
	for i := range minLen {
		if a[i] == b[i] {
			matches++
		} else {
			break
		}
	}
	similarity := float64(matches) / float64(minLen)
	return similarity
}

func clusterLabel(cluster []string) string {
	if len(cluster) == 0 {
		return ""
	}
	label := cluster[0]
	for _, candidate := range cluster {
		if len(candidate) < len(label) {
			label = candidate
		}
	}
	return label
}
