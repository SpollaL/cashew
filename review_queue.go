package main

type ReviewItem struct {
	label        string
	descriptions []string
}

func BuildReviewQueue(uniqueDescriptions map[string]string, clusters [][]string) []ReviewItem {
	uncategorizedClusters := make([][]string, 0)
	for _, cluster := range clusters {
		categorized := false
		for _, desc := range cluster {
			if uniqueDescriptions[desc] != CategoryUncategorized {
				categorized = true
				break
			}
		}
		if !categorized {
			uncategorizedClusters = append(uncategorizedClusters, cluster)
		}
	}
	reviewQueues := make([]ReviewItem, 0, len(uncategorizedClusters))
	for _, cluster := range uncategorizedClusters {
		reviewQueues = append(
			reviewQueues,
			ReviewItem{label: clusterLabel(cluster), descriptions: cluster},
		)
	}
	return reviewQueues
}
