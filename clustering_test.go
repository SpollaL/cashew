package main

import (
	"testing"
)

func TestClusterDescriptions(t *testing.T) {
	descriptions := []string{
		"Mercadona",
		"Mercadona",
		"Aldi",
		"Mercadona Supermercado",
		"Carrefour",
		"Carrefour Express",
		"Mercadona S.A.",
		"Aldi Supermercado",
		"Aldi Market",
		"Carrefour Market",
	}
	results := ClusterDescriptions(descriptions, 0.8)
	if len(results) != 3 {
		t.Errorf("Expected 3 clusters, got %d", len(results))
	}
}

func TestClusterDescriptionsEmpty(t *testing.T) {
	descriptions := []string{}
	results := ClusterDescriptions(descriptions, 0.8)
	if len(results) != 0 {
		t.Errorf("Expected 0 clusters, got %d", len(results))
	}
}

func TestClusterDescriptionsOne(t *testing.T) {
	descriptions := []string{
		"Mercadona",
	}
	results := ClusterDescriptions(descriptions, 0.8)
	if len(results) != 1 {
		t.Errorf("Expected 1 clusters, got %d", len(results))
	}
}

func TestClusterDescriptionsDuplicates(t *testing.T) {
	descriptions := []string{
		"Mercadona",
		"Mercadona",
		"Mercadona",
	}
	results := ClusterDescriptions(descriptions, 0.8)
	if len(results) != 1 {
		t.Errorf("Expected 1 clusters, got %d", len(results))
	}
}

func TestClusterDescriptionsMaxThreshold(t *testing.T) {
	descriptions := []string{
		"Mercadona",
		"Mercadona",
		"Aldi",
		"Mercadona Supermercado",
		"Carrefour",
		"Carrefour Express",
		"Mercadona S.A.",
		"Aldi Supermercado",
		"Aldi Market",
		"Carrefour Market",
	}
	results := ClusterDescriptions(descriptions, 1)
	if len(results) != 3 {
		t.Errorf("Expected 3 clusters, got %d", len(results))
	}
}

func TestClusterDescriptionsMinThreshold(t *testing.T) {
	descriptions := []string{
		"Mercadona",
		"Mercadona",
		"Aldi",
		"Mercadona Supermercado",
		"Carrefour",
		"Carrefour Express",
		"Mercadona S.A.",
		"Aldi Supermercado",
		"Aldi Market",
		"Carrefour Market",
	}
	results := ClusterDescriptions(descriptions, 0)
	if len(results) != 1 {
		t.Errorf("Expected 1 clusters, got %d", len(results))
	}
}

func TestClusterLabel(t *testing.T) {
	cluster := []string{
		"Mercadona",
		"Mercadona Supermercado",
		"Mercadona S.A.",
	}
	label := clusterLabel(cluster)
	if label != "Mercadona" {
		t.Errorf("Expected label 'Mercadona', got '%s'", label)
	}
}

func TestClusterLabelEmpty(t *testing.T) {
	cluster := []string{}
	label := clusterLabel(cluster)
	if label != "" {
		t.Errorf("Expected label '', got '%s'", label)
	}
}
