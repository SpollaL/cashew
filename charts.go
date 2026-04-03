package main

import (
	"fmt"

	"github.com/NimbleMarkets/ntcharts/barchart"

	"github.com/charmbracelet/lipgloss"
)

func createExpensesBarChartData(expensesSummary map[string]float64) []barchart.BarData {
	blockStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("2")).
		Background(lipgloss.Color("2"))
	barData := make([]barchart.BarData, 0, len(expensesSummary))
	for category, value := range expensesSummary {
		barValue := []barchart.BarValue{{Name: category, Value: value, Style: blockStyle}}
		barData = append(
			barData,
			barchart.BarData{Label: fmt.Sprintf("%s %.2f $", category, value), Values: barValue},
		)
	}
	return barData
}

func createExpensesBarChart(barData []barchart.BarData) barchart.Model {
	chart := barchart.New(60, 20, barchart.WithDataSet(barData), barchart.WithHorizontalBars())
	return chart
}
