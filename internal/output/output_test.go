package output

import (
	"os"
	"testing"

	"github.com/shanehull/quickval/internal/quickfs"
	"github.com/stretchr/testify/assert"
)

func WriterRender_Test(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "example")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	w := NewWriter(tmpfile)

	// mock data
	data := &quickfs.Data{
		FCFHistory:   []int{100, 150, 200},
		CFFDividends: []int{50, 70, 90},
		DebtToEquity: 1.5,
		TaxRate:      0.25,
		Beta:         1.2,
	}

	w.Data(data)
	w.WACC(0, 0.05, 0.04, data)

	// run the render function
	w.Render()

	content, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	assert.Contains(t, string(content), "FY HISTORIC DATA")
	assert.Contains(t, string(content), "DISCOUNT RATE")
	assert.Contains(t, string(content), "PROJECTIONS")
	assert.Contains(t, string(content), "FAIR VALUE")
}
