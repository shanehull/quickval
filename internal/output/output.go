package output

import (
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/shanehull/quickval/internal/quickfs"
)

type Writer struct {
	table *tablewriter.Table
}

func NewWriter(file *os.File) *Writer {
	w := &Writer{table: tablewriter.NewWriter(file)}
	w.table.SetBorders(tablewriter.Border{Left: false, Top: true, Right: false, Bottom: true})
	w.table.SetCenterSeparator("|")
	w.table.SetRowSeparator("=")

	return w
}

func (w *Writer) Data(data *quickfs.Data) {
	w.table.Append([]string{"FY HISTORIC DATA", ""})
	w.table.Append([]string{"----------------------------------------", "------------------"})

	// append FCF values
	for year, value := range data.FCFHistory {
		label := fmt.Sprintf("FCF Yr %d", year+1)
		formattedValue := fmt.Sprintf("%d", value)
		row := []string{label, formattedValue}
		w.table.Append(row)
	}

	// append CFFDividends values
	for year, value := range data.CFFDividends {
		label := fmt.Sprintf("Cash Paid for Dividends Yr %d", year+1)
		formattedValue := fmt.Sprintf("%d", value)
		row := []string{label, formattedValue}
		w.table.Append(row)
	}

	w.table.Append([]string{"", ""})
}

func (w *Writer) WACC(rate float64, erp float64, rfr float64, data *quickfs.Data) {
	w.table.Append([]string{"", ""})
	w.table.Append([]string{"DISCOUNT RATE (WACC)", ""})
	w.table.Append([]string{"----------------------------------------", "------------------"})
	w.table.Append([]string{"Equity Risk Premium", fmt.Sprintf("%.2f", erp)})
	w.table.Append([]string{"Risk Free Rate", fmt.Sprintf("%.2f", rfr)})
	if data.DebtToEquity != 0 {
		w.table.Append([]string{"Debt to Equity Ratio", fmt.Sprintf("%.2f", data.DebtToEquity)})
	}
	if data.TaxRate != 0 {
		w.table.Append([]string{"Tax Rate", fmt.Sprintf("%.2f", data.TaxRate)})
	}
	if data.Beta != 0 {
		w.table.Append([]string{"Beta", fmt.Sprintf("%.2f", data.Beta)})
	}
	w.table.Append([]string{"", ""})
	w.table.Append([]string{"Discount Rate", fmt.Sprintf("%.2f", rate)})

	w.table.Append([]string{"", ""})
}

func (w *Writer) Projected(projected []int, growthRate float64, expectedReturn float64) {
	w.table.Append([]string{"", ""})
	w.table.Append([]string{"PROJECTIONS", ""})
	w.table.Append([]string{"----------------------------------------", "------------------"})
	w.table.Append([]string{"Growth Rate", fmt.Sprintf("%.2f", growthRate)})
	w.table.Append([]string{"Expected Return", fmt.Sprintf("%.2f", expectedReturn)})
	w.table.Append([]string{"", ""})

	// append projections
	for year, value := range projected {
		label := fmt.Sprintf("Projected Yr %d", year+1)
		formattedValue := fmt.Sprintf("%d", value)
		row := []string{label, formattedValue}
		w.table.Append(row)
	}
}

func (w *Writer) DiscountRate(rate float64) {
	w.table.Append([]string{"", ""})
	w.table.Append([]string{"----------------------------------------", "------------------"})
	w.table.Append([]string{"DISCOUNT RATE", ""})
	w.table.Append([]string{"User Specified", fmt.Sprintf("%.2f", rate)})

	w.table.Append([]string{"", ""})
}

func (w *Writer) FairValue(value float64) {
	w.table.SetFooter([]string{"Fair Value", fmt.Sprintf("%.2f", value)})
	w.table.SetFooterAlignment(1)
	w.table.Append([]string{"", ""})
}

func (w *Writer) Render() {
	fmt.Println()
	w.table.Render()
}
