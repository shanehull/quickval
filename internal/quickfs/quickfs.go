package quickfs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
)

var CountryCodes = []string{"US", "AT", "AU", "BE", "CA", "CH", "DE", "DK", "ES", "FI", "FI", "FR", "GR", "IT", "LN", "MM", "NL", "NO", "NZ", "PL", "SE"}

type Data struct {
	Shares       int     `json:"shares"`
	TaxRate      float64 `json:"taxRate"`
	DebtToEquity float64 `json:"debtToEquity"`
	Beta         float64 `json:"beta"`
	FCFHistory   []int   `json:"fcfHistory"`
	CFFDividends []int   `json:"cffDividends"`
}

type Companies []string

type quickFS struct {
	debtToEquity bool
	beta         bool
	fcf          bool
	cffDividends bool
	fyHistory    int
	apiKey       string
	client       *http.Client
}

type ConfigOption func(q *quickFS)

func WithAPIKey(key string) ConfigOption {
	return func(q *quickFS) {
		q.apiKey = key
	}
}

func WithFCF() ConfigOption {
	return func(q *quickFS) {
		q.fcf = true
	}
}

func WithCFFDividends() ConfigOption {
	return func(q *quickFS) {
		q.cffDividends = true
	}
}

func WithDebtToEquity() ConfigOption {
	return func(q *quickFS) {
		q.debtToEquity = true
	}
}

func WithBeta() ConfigOption {
	return func(q *quickFS) {
		q.beta = true
	}
}

func WithFYHistory(hist int) ConfigOption {
	return func(q *quickFS) {
		q.fyHistory = hist
	}
}

func NewQuickFS(opts ...ConfigOption) *quickFS {
	q := &quickFS{}
	for _, opt := range opts {
		opt(q)
	}
	q.client = &http.Client{}

	return q
}

type dataResponse struct {
	Data struct {
		Shares       []int     `json:"shares"`
		TaxRate      []float64 `json:"taxRate"`
		DebtToEquity []float64 `json:"debtToEquity"`
		Beta         float64   `json:"beta"`
		FCFHistory   []int     `json:"fcfHistory"`
		CFFDividends []int     `json:"cffDividends"`
	} `json:"data"`
}

// Gets data from QuickFS. Data points can be customized when creating the QuickFS instance, using "with" options, e.g: quickfs.NewQuickFS(quickfs.WithCFFDividends()).
func (q *quickFS) GetData(ticker string, country string) (Data, error) {
	var data Data

	type payloadData struct {
		Shares       string `json:"shares"`
		TaxRate      string `json:"taxRate"`
		DebtToEquity string `json:"debtToEquity,omitempty"`
		Beta         string `json:"beta,omitempty"`
		FCFHistory   string `json:"fcfHistory,omitempty"`
		CFFDividends string `json:"cffDividends,omitempty"`
	}

	type payload struct {
		Data payloadData `json:"data"`
	}

	pl := &payload{
		Data: payloadData{
			Shares:  q.formatQFS(ticker, country, "shares_basic"),
			TaxRate: q.formatQFS(ticker, country, "income_tax_rate"),
		},
	}

	q.formatOptionalQFS(&pl.Data.DebtToEquity, ticker, country, q.debtToEquity, "debt_to_equity")
	q.formatOptionalQFS(&pl.Data.Beta, ticker, country, q.beta, "beta")
	q.formatOptionalQFS(&pl.Data.FCFHistory, ticker, country, q.fcf, "fcf", fmt.Sprintf("FY-%d:FY", q.fyHistory-1))
	q.formatOptionalQFS(&pl.Data.CFFDividends, ticker, country, q.cffDividends, "cff_dividend_paid", fmt.Sprintf("FY-%d:FY", q.fyHistory-1))

	jsonPayload, err := json.Marshal(pl)
	if err != nil {
		log.Error().Err(err).Msg("error marshaling quickfs payload")
		return data, err
	}

	req, _ := http.NewRequest(http.MethodPost, "https://public-api.quickfs.net/v1/data/batch", bytes.NewReader(jsonPayload))
	q.setHeaders(req)

	res, err := q.client.Do(req)
	if err != nil {
		return data, fmt.Errorf("error building request to get data: %w", err)
	}

	if res.StatusCode/100 != 2 {
		return data, fmt.Errorf("status: %s", res.Status)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return data, fmt.Errorf("error reading quickfs response: %w", err)
	}

	bodyStr := string(body)
	if err := handleErrorInBody(bodyStr, ticker, country); err != nil {
		return data, err
	}

	dataResp := dataResponse{}
	if err = json.NewDecoder(bytes.NewBuffer(body)).Decode(&dataResp); err != nil {
		return data, err
	}

	data = Data{
		Shares:  dataResp.Data.Shares[0],
		TaxRate: dataResp.Data.TaxRate[0],
	}

	assignOptionalField(q.debtToEquity, &data.DebtToEquity, dataResp.Data.DebtToEquity[0])
	assignOptionalField(q.beta, &data.Beta, dataResp.Data.Beta)
	assignOptionalField(q.fcf, &data.FCFHistory, dataResp.Data.FCFHistory)

	if q.cffDividends {
		for _, c := range dataResp.Data.CFFDividends {
			data.CFFDividends = append(data.CFFDividends, reverseInt(c))
		}
	}

	return data, nil
}

// Gets all supported companies for the specified (ISO Alpha-2) country code, e.g; ["AAPL:US", ...].
func (q *quickFS) GetCompanies(country string) (Companies, error) {
	var companies Companies

	reqUrl := fmt.Sprintf("https://public-api.quickfs.net/v1/companies/%s", strings.ToLower(country))

	req, _ := http.NewRequest(http.MethodGet, reqUrl, nil)
	q.setHeaders(req)

	res, err := q.client.Do(req)
	if err != nil {
		return companies, fmt.Errorf("error building request to get companies: %w", err)
	}

	if res.StatusCode/100 != 2 {
		return companies, fmt.Errorf("error requesting companies - status: %s", res.Status)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return companies, fmt.Errorf("error reading quickfs response: %w", err)
	}

	type response struct {
		Data []string `json:"data"`
	}

	dataResp := response{}
	if err = json.NewDecoder(bytes.NewBuffer(body)).Decode(&dataResp); err != nil {
		return companies, err
	}

	return dataResp.Data, nil
}

func (q *quickFS) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-QFS-API-Key", q.apiKey)
}

func (q *quickFS) formatQFS(ticker, country, metric string) string {
	return fmt.Sprintf("QFS(%s:%s,%s)", ticker, country, metric)
}

func (q *quickFS) formatOptionalQFS(field *string, ticker, country string, condition bool, metric string, args ...interface{}) {
	if condition {
		*field = fmt.Sprintf("QFS(%s:%s,%s", ticker, country, metric)
		if len(args) > 0 {
			*field += fmt.Sprintf(",%s)", fmt.Sprintf(args[0].(string)))
		} else {
			*field += ")"
		}
	}
}

func assignOptionalField(condition bool, target interface{}, source interface{}) {
	if condition {
		switch v := target.(type) {
		case *float64:
			*v = source.(float64)
		case *int:
			*v = source.(int)
		case *[]int:
			*v = source.([]int)
		case *[]float64:
			*v = source.([]float64)
		}
	}
}

func handleErrorInBody(bodyStr, ticker, country string) error {
	if strings.Contains(bodyStr, "UnsupportedCompanyError") {
		return fmt.Errorf("unsupported company, %s:%s", ticker, country)
	} else if strings.Contains(bodyStr, "UnsupportedMetricError") {
		return fmt.Errorf("unsupported metric")
	}
	// else if strings.Contains(bodyStr, "InsufficientQuotaError") {  // this results in a 429 so is moot, I think
	//     return fmt.Errorf("insufficient quota, %s:%s", ticker, country)
	// }
	return nil
}

func reverseInt(value int) int {
	return -value
}
