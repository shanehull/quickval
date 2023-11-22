package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/shanehull/quickval/internal/calc"
	"github.com/shanehull/quickval/internal/output"
	"github.com/shanehull/quickval/internal/quickfs"
	"github.com/urfave/cli/v2"
)

func doCommonSetup(cCtx *cli.Context, writer *output.Writer, opts ...quickfs.ConfigOption) (quickfs.Data, int, float64, error) {
	var (
		data              quickfs.Data
		equityRiskPremium float64
		riskFreeRate      float64
		err               error
	)

	fyHistory := cCtx.Int("fy-history")
	if fyHistory == 0 {
		fyHistory = promptInt("FY History", 5, fyHistoryPromptInfo)
	}

	discountRate := cCtx.Float64("discount-rate")
	var discountRateOpt string
	if discountRate == 0.00 {
		discountRateOpt = selectDiscountRateOpt()

		switch discountRateOpt {
		case "WACC":
			equityRiskPremium = cCtx.Float64("risk-premium")
			if equityRiskPremium == 0.0 {
				equityRiskPremium = promptFloat("Equity Risk Premium", 0.05, erpPromptInfo)
			}

			riskFreeRate = cCtx.Float64("risk-free")
			if riskFreeRate == 0.0 {
				riskFreeRate = promptFloat("Risk-Free Rate", 0.042, rfrPromptInfo)
			}

			mergedOpts := append(opts,
				quickfs.WithAPIKey(apiKey),
				quickfs.WithFYHistory(fyHistory),
				quickfs.WithDebtToEquity(),
				quickfs.WithBeta(),
			)

			qfs := quickfs.NewQuickFS(
				mergedOpts...,
			)

			data, err = qfs.GetData(ticker, country)
			if err != nil {
				log.Fatalf("error getting data: %s", err)
			}

			wacc := calc.WACC(data.Beta, data.DebtToEquity, data.TaxRate, equityRiskPremium, riskFreeRate)

			discountRate = wacc

			writer.Data(&data)
			writer.WACC(discountRate, equityRiskPremium, riskFreeRate, &data)
		case "CV Weighted WACC":
			equityRiskPremium = cCtx.Float64("risk-premium")
			if equityRiskPremium == 0.0 {
				equityRiskPremium = promptFloat("Equity Risk Premium", 0.05, erpPromptInfo)
			}

			riskFreeRate = cCtx.Float64("risk-free")
			if riskFreeRate == 0.0 {
				riskFreeRate = promptFloat("Risk Free Rate", 0.02, rfrPromptInfo)
			}

			mergedOpts := append(opts,
				quickfs.WithAPIKey(apiKey),
				quickfs.WithFYHistory(fyHistory),
				quickfs.WithDebtToEquity(),
			)

			qfs := quickfs.NewQuickFS(
				mergedOpts...,
			)

			data, err = qfs.GetData(ticker, country)
			if err != nil {
				log.Fatalf("error getting data: %s", err)
			}

			wacc := calc.FCFCVWeightedWACC(
				data.FCFHistory,
				data.DebtToEquity,
				data.TaxRate,
				equityRiskPremium,
				riskFreeRate,
			)

			discountRate = wacc

			writer.Data(&data)
			writer.WACC(discountRate, equityRiskPremium, riskFreeRate, &data)
		case "Custom Input":

			mergedOpts := append(opts,
				quickfs.WithAPIKey(apiKey),
				quickfs.WithFYHistory(fyHistory),
			)

			qfs := quickfs.NewQuickFS(
				mergedOpts...,
			)

			data, err = qfs.GetData(ticker, country)
			if err != nil {
				log.Fatalf("error getting data: %s", err)
			}

			discountRate = promptFloat("Discount Rate", 0.06, discPromptInfo)

			writer.Data(&data)
			writer.DiscountRate(discountRate)
		default:
			err := cli.Exit("unsupported discount rate option", 127)
			if err != nil {
				return data, 0, 0, err
			}
		}
	}

	return data, fyHistory, discountRate, nil
}

func fetchTickers(country string) ([]string, error) {
	cacheFilePath := filepath.Join(cacheDir, fmt.Sprintf("%s.json", country))

	// try to load from local cache first
	data, err := os.ReadFile(cacheFilePath)
	if err == nil {
		// data was successfully loaded from cache
		if err := json.Unmarshal(data, &searchTickers); err == nil {
			// ignore errors

			// refresh local cache in the background
			go func() {
				if err := updateLocalCache(country, cacheFilePath); err != nil {
					log.Printf("failed to update local cache: %s", err)
				}
			}()
			return searchTickers, nil
		}
	}

	// if no local cache, get them from the repo - quickfs is too slow
	searchTickers, err := fetchTickersFromGH(country)
	if err != nil {
		// we should never get here, but if we do, it should throw an error
		return nil, errors.New("error retrieving tickers")
	}

	// refresh local cache in the background
	go func() {
		if err := updateLocalCache(country, cacheFilePath); err != nil {
			log.Printf("failed to update local cache: %s", err)
		}
	}()

	return searchTickers, nil
}

func fetchTickersFromGH(country string) ([]string, error) {
	url := fmt.Sprintf(ghTickersURLFmt, country)

	client := &http.Client{
		Timeout: time.Second * 10,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("received non-200 status code: %d", resp.StatusCode)
	}

	var fetchedTickers []string
	if err := json.NewDecoder(resp.Body).Decode(&fetchedTickers); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	var ghTickers []string
	for _, t := range fetchedTickers {
		tickerSplit := strings.Split(t, ":")
		ghTickers = append(ghTickers, tickerSplit[0])
	}

	return ghTickers, nil
}

func updateLocalCache(country, cacheFilePath string) error {
	var localTickers []string
	qfs := quickfs.NewQuickFS(quickfs.WithAPIKey(apiKey))
	availTickers, err := qfs.GetCompanies(country)
	if err != nil {
		log.Fatalf("an error occurred when requesting tickers list: %s", err)
	}

	for _, t := range availTickers {
		tickerSplit := strings.Split(t, ":")
		localTickers = append(localTickers, tickerSplit[0])
	}

	data, err := json.Marshal(localTickers)
	if err != nil {
		return err
	}

	return atomicWrite(cacheFilePath, data, 0755)
}

func atomicWrite(filename string, data []byte, perms fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(filename), perms); err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(filename), "tmp-cache-")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name()) // Cleanup the temporary file

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	return os.Rename(tmpFile.Name(), filename)
}

func setCommonVars(cCtx *cli.Context) error {
	envVal, ok := os.LookupEnv("QFS_API_KEY")
	if ok {
		apiKey = envVal
	}

	flagVal := cCtx.String("api-key")
	if flagVal != "" {
		apiKey = flagVal
	}

	if apiKey == "" {
		apiKey = promptKey()
	}

	country = cCtx.String("country")
	if country == "" {
		country = selectCountry()
	}

	ticker = cCtx.String("ticker")
	if ticker == "" {
		ticker = selectTicker(country, apiKey)
	}

	return nil
}

func promptKey() string {
	fmt.Println()
	fmt.Println("Enter a valid API key for QuickFS.")

	validate := func(input string) error {
		r, _ := regexp.Compile("^[a-z0-9]{40}$")

		m := r.MatchString(input)
		if !m {
			return errors.New("invalid api key")
		}
		return nil
	}

	s := promptui.Prompt{
		Label:       "API Key",
		Validate:    validate,
		Mask:        '*',
		HideEntered: true,
	}

	response, err := s.Run()

	if err != nil {
		log.Fatalf("an error occurred when setting the api key: %s", err)
	}
	return response
}

func selectTicker(country string, apiKey string) string {
	fmt.Println()
	fmt.Println("Start typing to find your ticker.")

	tickers, err := fetchTickers(country)
	if err != nil {
		log.Fatalf("an error occurred when fetching ticker: %s", err)
	}

	searcher := func(input string, index int) bool {
		ticker := strings.ToLower(tickers[index])
		input = strings.Replace(strings.ToLower(input), " ", "", -1)
		return strings.Contains(ticker, input)
	}

	s := promptui.Select{
		Label:             "Ticker",
		Items:             tickers,
		Searcher:          searcher,
		StartInSearchMode: true,
	}

	_, response, err := s.Run()
	if err != nil {
		log.Fatalf("an error occurred when setting the ticker: %s", err)
	}

	return response
}

func selectCountry() string {
	fmt.Println()
	fmt.Println("Select the country that your ticker trades in.")

	searcher := func(input string, index int) bool {
		ticker := strings.ToLower(quickfs.CountryCodes[index])
		input = strings.Replace(strings.ToLower(input), " ", "", -1)
		return strings.Contains(ticker, input)
	}

	s := promptui.Select{
		Label:             "Country",
		Items:             quickfs.CountryCodes,
		Searcher:          searcher,
		StartInSearchMode: true,
	}

	_, response, err := s.Run()
	if err != nil {
		log.Fatalf("an error occurred when setting the country: %s", err)
	}

	return response
}

func promptInt(label string, def int, info string) int {
	if info != "" {
		fmt.Println()
		fmt.Println(info)
	}

	validate := func(input string) error {
		_, err := strconv.ParseInt(input, 10, 64)
		if err != nil {
			return err
		}
		return nil
	}

	s := promptui.Prompt{
		Label:     label,
		Validate:  validate,
		AllowEdit: true,
		Default:   fmt.Sprint(def),
	}

	response, err := s.Run()
	if err != nil {
		log.Fatalf("an error occurred when setting %s: %s", label, err)
	}

	val, err := strconv.ParseInt(response, 10, 0)
	if err != nil {
		log.Fatalf("an error occurred when setting %s: %s", label, err)
	}

	return int(val)
}

func promptFloat(label string, def float64, info string) float64 {
	if info != "" {
		fmt.Println()
		fmt.Println(info)
	}

	validate := func(input string) error {
		_, err := strconv.ParseFloat(input, 64)
		if err != nil {
			return err
		}
		return nil
	}

	sDef := strconv.FormatFloat(def, 'g', 5, 64)

	s := promptui.Prompt{
		Label:     label,
		Validate:  validate,
		AllowEdit: true,
		Default:   sDef,
	}

	response, err := s.Run()
	if err != nil {
		log.Fatalf("an error occurred when setting %s: %s", label, err)
	}

	val, err := strconv.ParseFloat(response, 64)
	if err != nil {
		log.Fatalf("an error occurred when setting %s: %s", label, err)
	}

	return val
}
func getFlagOrPromptFloat(cCtx *cli.Context, flagName, prompt, promptInfo string, defaultValue float64) float64 {
	value := cCtx.Float64(flagName)
	if value == 0.00 {
		return promptFloat(prompt, defaultValue, promptInfo)
	}
	return value
}

func getFlagOrPromptGrowthRate(cCtx *cli.Context, flagName, prompt, promptInfo string, series []int) float64 {
	value := cCtx.Float64(flagName)
	if value == 0.00 {
		cagr, _ := calc.CAGR(series)
		return promptFloat(prompt, cagr, promptInfo)
	}
	return value
}

func getFlagOrPromptInt(cCtx *cli.Context, flagName, prompt, promptInfo string, defaultValue int) int {
	value := cCtx.Int(flagName)
	if value == 0 {
		return promptInt(prompt, defaultValue, promptInfo)
	}
	return value
}

func selectDiscountRateOpt() string {
	fmt.Println()
	fmt.Println("There are a few options for calculating a discount rate. Choose which one you would like to use.")

	s := promptui.Select{
		Label: "Discount Rate Options",
		Items: []string{"WACC", "CV Weighted WACC", "Custom Input"},
	}

	_, response, err := s.Run()
	if err != nil {
		log.Fatalf("an error occurred when selecting a discount rate option: %s", err)
	}

	return response
}
