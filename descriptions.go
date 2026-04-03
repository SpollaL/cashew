package main

func GetUniqueDescriptions(transactions []Transaction) map[string]string {
	if len(transactions) == 0 {
		return map[string]string{}
	}
	uniqueDescriptions := make(map[string]string)
	for _, t := range transactions {
		if t.Type == TypeExpense {
			uniqueDescriptions[t.Description] = t.Category
		}
	}
	return uniqueDescriptions
}

func ClusterTransactions(uniqueDescriptions map[string]string) [][]string {
	uniqueList := make([]string, 0, len(uniqueDescriptions))
	for desc := range uniqueDescriptions {
		uniqueList = append(uniqueList, desc)
	}
	clusters := ClusterDescriptions(uniqueList, 0.8)
	return clusters
} 
