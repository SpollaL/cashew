package main

type ReviewItem struct {
	label        string
	descriptions []string
}

func BuildReviewQueue(transactions []Transaction) []ReviewItem {
	uniqueDescriptions := make(map[string]struct{})
	for _, t := range transactions {
		if t.Category != CategoryUncategorized {
			uniqueDescriptions[t.Description] = struct{}{}
		}
	}
	uniqueList := make([]string, 0, len(uniqueDescriptions))
	for desc := range uniqueDescriptions {
		uniqueList = append(uniqueList, desc)
	}
	clusters := ClusterDescriptions(uniqueList, 0.8)
	reviewQueues := make([]ReviewItem, 0, len(clusters))
	for _, cluster := range clusters {
		reviewQueues = append(
			reviewQueues,
			ReviewItem{label: clusterLabel(cluster), descriptions: cluster},
		)
	}
	return reviewQueues
}
