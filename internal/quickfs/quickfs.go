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

	type response struct {
		Data struct {
			Shares       []int     `json:"shares"`
			TaxRate      []float64 `json:"taxRate"`
			DebtToEquity []float64 `json:"debtToEquity"`
			Beta         float64   `json:"beta"`
			FCFHistory   []int     `json:"fcfHistory"`
			CFFDividends []int     `json:"cffDividends"`
		} `json:"data"`
	}

	pl := &payload{
		Data: payloadData{
			Shares:  fmt.Sprintf("QFS(%s:%s,shares_basic,FY)", ticker, country),
			TaxRate: fmt.Sprintf("QFS(%s:%s,income_tax_rate,FY)", ticker, country),
		},
	}

	if q.debtToEquity {
		pl.Data.DebtToEquity = fmt.Sprintf("QFS(%s:%s,debt_to_equity,FY)", ticker, country)
	}
	if q.beta {
		pl.Data.Beta = fmt.Sprintf("QFS(%s:%s,beta,FY)", ticker, country)
	}
	if q.fcf {
		if q.fyHistory == 1 {
			pl.Data.FCFHistory = fmt.Sprintf("QFS(%s:%s,fcf,FY)", ticker, country)
		}
		pl.Data.FCFHistory = fmt.Sprintf("QFS(%s:%s,fcf,FY-%d:FY)", ticker, country, q.fyHistory-1)
	}
	if q.cffDividends {
		if q.fyHistory == 1 {
			pl.Data.CFFDividends = fmt.Sprintf("QFS(%s:%s,cff_dividend_paid,FY)", ticker, country)
		}
		pl.Data.CFFDividends = fmt.Sprintf("QFS(%s:%s,cff_dividend_paid,FY-%d:FY)", ticker, country, q.fyHistory-1)
	}

	jsonPayload, err := json.Marshal(pl)
	if err != nil {
		log.Error().Err(err).Msg("error marshaling quickfs payload")
		return data, err
	}

	req, _ := http.NewRequest(http.MethodPost, "https://public-api.quickfs.net/v1/data/batch", bytes.NewReader(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-QFS-API-Key", q.apiKey)

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
	if strings.Contains(bodyStr, "UnsupportedCompanyError") {
		return data, fmt.Errorf("unsupported company, %s:%s", ticker, country)
	}
	if strings.Contains(bodyStr, "UnsupportedMetricError") {
		return data, fmt.Errorf("unsupported metric, body: %s", bodyStr)
	}
	// if strings.Contains(bodyStr, "InsufficientQuotaError") { // this results in a 429 so is moot, I think
	// 	return data, fmt.Errorf("insufficient quota, %s:%s", ticker, country)
	// }

	dataResp := response{}
	if err = json.NewDecoder(bytes.NewBuffer(body)).Decode(&dataResp); err != nil {
		return data, err
	}

	data = Data{
		Shares:  dataResp.Data.Shares[0],
		TaxRate: dataResp.Data.TaxRate[0],
	}

	if q.debtToEquity {
		data.DebtToEquity = dataResp.Data.DebtToEquity[0]
	}
	if q.beta {
		data.Beta = dataResp.Data.Beta
	}
	if q.fcf {
		data.FCFHistory = dataResp.Data.FCFHistory
	}
	if q.cffDividends {
		// we need to roll through these and reverse them - they're from the perspective of the firm
		for _, c := range dataResp.Data.CFFDividends {
			data.CFFDividends = append(data.CFFDividends, reverseInt(c))
		}
	}

	return data, nil
}

func (q *quickFS) GetCompanies(country string) (Companies, error) {
	var companies Companies

	reqUrl := fmt.Sprintf("https://public-api.quickfs.net/v1/companies/%s", strings.ToLower(country))

	req, _ := http.NewRequest(http.MethodGet, reqUrl, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-QFS-API-Key", q.apiKey)

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

func reverseInt(value int) int {
	return -value
}
