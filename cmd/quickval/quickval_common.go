package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/shanehull/quickval/internal/calc"
	"github.com/shanehull/quickval/internal/output"
	"github.com/shanehull/quickval/internal/quickfs"
	"github.com/urfave/cli/v2"
)

var (
	searchTickers   []string
	cacheDir        = "/tmp/quickval"
	ghTickersURLFmt = "https://raw.githubusercontent.com/shanehull/quickval/tickers/tickers/%s.json"
)

var (
	rfrPromptInfo       = "Enter a Risk-Rree Rate (e.g a 10-year average of a 10-year treasury bond yield)."
	erpPromptInfo       = "Enter an ERP (Equity Risk Premium)."
	discPromptInfo      = "Enter an explicit discount rate."
	growthPromptInfo    = "Enter a reasonable growth rate, or accept the default (derived from a CAGR of the FCF or dividend history)."
	fcfPromptInfo       = "Enter a current FCF (e.g. a normalised figure) or accept the most recent reported figure."
	dividendsPromptInfo = "Enter a current Cash Paid for Dividends value (e.g. a normalised figure) or accept the most recent reported figure."
	exitPromptInfo      = "Enter an exit multiple, or accept the current P/FCF (rounded down)."
	perpetualGrowthInfo = "Enter a growth rate for the perpetual/terminal growth stage."
	fyHistoryPromptInfo = "Enter a FY history to retrieve for financial reports."
)

var (
	defaultRFR           = 0.042
	defaultPerpetualRate = 0.02
	defaultERP           = 0.05
)

func doCommonSetup(
	cCtx *cli.Context,
	writer *output.Writer,
	opts ...quickfs.ConfigOption,
) (quickfs.Data, int, float64, error) {
	var (
		data              quickfs.Data
		equityRiskPremium float64
		riskFreeRate      float64
		err               error
	)

	fyHistory := cCtx.Int("fy-history")
	if fyHistory == 0 {
		fyHistory, err = promptInt("FY History", 5, fyHistoryPromptInfo)
		if err != nil {
			return data, fyHistory, equityRiskPremium, err
		}
	}

	discountRate := cCtx.Float64("discount-rate")
	var discountRateOpt string
	if discountRate == 0.00 {
		discountRateOpt = selectDiscountRateOpt()

		switch discountRateOpt {
		case "WACC":
			equityRiskPremium = cCtx.Float64("risk-premium")
			if equityRiskPremium == 0.0 {
				equityRiskPremium, err = promptFloat(
					"Equity Risk Premium",
					defaultERP,
					erpPromptInfo,
				)
				if err != nil {
					return data, fyHistory, discountRate, err
				}
			}

			riskFreeRate = cCtx.Float64("risk-free")
			if riskFreeRate == 0.0 {
				riskFreeRate, err = promptFloat("Risk-Free Rate", defaultRFR, rfrPromptInfo)
				if err != nil {
					return data, fyHistory, discountRate, err
				}
			}

			mergedOpts := append(opts,
				quickfs.WithAPIKey(apiKey),
				quickfs.WithFYHistory(fyHistory),
				quickfs.WithBeta(),
			)

			qfs := quickfs.NewQuickFS(
				mergedOpts...,
			)

			data, err = qfs.GetData(ticker, country)
			if err != nil {
				return data, fyHistory, discountRate, fmt.Errorf("error getting data: %s", err)
			}

			wacc := calc.WACC(
				data.Beta,
				data.DebtToEquity,
				data.TaxRate,
				equityRiskPremium,
				riskFreeRate,
			)

			discountRate = wacc

			writer.Data(&data)
			writer.WACC(discountRate, equityRiskPremium, riskFreeRate, &data)
		case "CV Weighted WACC":
			equityRiskPremium = cCtx.Float64("risk-premium")
			if equityRiskPremium == 0.0 {
				equityRiskPremium, err = promptFloat("Equity Risk Premium", 0.05, erpPromptInfo)
				if err != nil {
					return data, fyHistory, discountRate, err
				}
			}

			riskFreeRate = cCtx.Float64("risk-free")
			if riskFreeRate == 0.0 {
				riskFreeRate, err = promptFloat("Risk Free Rate", defaultRFR, rfrPromptInfo)
				if err != nil {
					return data, fyHistory, discountRate, err
				}
			}

			mergedOpts := append(opts,
				quickfs.WithAPIKey(apiKey),
				quickfs.WithFYHistory(fyHistory),
			)

			qfs := quickfs.NewQuickFS(
				mergedOpts...,
			)

			data, err = qfs.GetData(ticker, country)
			if err != nil {
				return data, 0, 0, fmt.Errorf("error getting data: %s", err)
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
			discountRate, err = promptFloat("Discount Rate", 0.10, discPromptInfo)
			if err != nil {
				return data, 0, 0, err
			}

			mergedOpts := append(opts,
				quickfs.WithAPIKey(apiKey),
				quickfs.WithFYHistory(fyHistory),
			)

			qfs := quickfs.NewQuickFS(
				mergedOpts...,
			)

			data, err = qfs.GetData(ticker, country)
			if err != nil {
				return data, 0, 0, fmt.Errorf("error getting data: %s", err)
			}

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
					fmt.Printf("failed to update local cache: %s", err)
					return
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
			fmt.Printf("failed to update local cache: %s", err)
			return
		}
	}()

	return searchTickers, nil
}

func fetchTickersFromGH(country string) ([]string, error) {
	url := fmt.Sprintf(ghTickersURLFmt, strings.ToUpper(country))

	client := &http.Client{
		Timeout: time.Second * 10,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

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
		return fmt.Errorf("an error occurred when requesting tickers list: %s", err)
	}

	for _, t := range availTickers {
		tickerSplit := strings.Split(t, ":")
		localTickers = append(localTickers, tickerSplit[0])
	}

	data, err := json.Marshal(localTickers)
	if err != nil {
		return err
	}

	return atomicWrite(cacheFilePath, data, 0o755)
}

func atomicWrite(filename string, data []byte, perms fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(filename), perms); err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(filename), "tmp-cache-")
	if err != nil {
		return err
	}
	defer func() {
		_ = os.Remove(tmpFile.Name()) // Cleanup the temporary file
	}()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	return os.Rename(tmpFile.Name(), filename)
}

func setCommonVars(cCtx *cli.Context) error {
	var err error
	envVal, ok := os.LookupEnv("QFS_API_KEY")
	if ok {
		apiKey = envVal
	}

	flagVal := cCtx.String("api-key")
	if flagVal != "" {
		apiKey = flagVal
	}

	if apiKey == "" {
		apiKey, err = promptKey()
		if err != nil {
			return err
		}
	}

	country = cCtx.String("country")
	if country == "" {
		country, err = selectCountry()
		if err != nil {
			return err
		}
	}

	ticker = cCtx.String("ticker")
	if ticker == "" {
		ticker, err = selectTicker(country)
		if err != nil {
			return err
		}
	}

	return nil
}

func printTip(info string) {
	fmt.Println()
	fmt.Println(info)
	fmt.Println("---")
}

func promptKey() (string, error) {
	printTip("Enter a valid API key for QuickFS.")

	var apiKeyInput string
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("API Key").
				Placeholder("").
				EchoMode(huh.EchoModePassword).
				Validate(func(s string) error {
					if s == "" {
						return errors.New("input cannot be empty")
					}
					r, _ := regexp.Compile("^[a-z0-9]{40}$")
					if !r.MatchString(s) {
						return errors.New("invalid api key format")
					}
					return nil
				}).
				Value(&apiKeyInput),
		),
	).Run()
	if err != nil {
		return "", fmt.Errorf("an error occurred when setting the api key: %s", err)
	}

	return apiKeyInput, nil
}

func selectTicker(country string) (string, error) {
	printTip("Start typing to find your ticker.")

	tickers, _ := fetchTickers(country)

	if len(tickers) == 0 {
		return "", fmt.Errorf("no tickers available for country %s", country)
	}

	huhOptions := convertToHuhOptions(tickers)

	var selectedTicker string
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Ticker").
				Options(huhOptions...).
				Filtering(true).
				// FilterFunc(func(option huh.Option[string], input string) bool {
				// 	return strings.HasPrefix(strings.ToLower(option.Label()), strings.ToLower(input))
				// }).
				Height(8).
				Value(&selectedTicker),
		),
	).Run()
	if err != nil {
		return "", fmt.Errorf("an error occurred when setting the ticker: %s", err)
	}

	fmt.Printf("Ticker selected: %s\n", selectedTicker)
	return selectedTicker, nil
}

func selectCountry() (string, error) {
	printTip("Select the country that your ticker trades in.")

	countryCodes := quickfs.CountryCodes
	huhOptions := convertToHuhOptions(countryCodes)

	var selectedCountry string
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Country").
				Options(huhOptions...).
				Filtering(true).
				Height(8).
				Value(&selectedCountry),
		),
	).Run()
	if err != nil {
		return "", fmt.Errorf("an error occurred when setting the country: %s", err)
	}

	fmt.Printf("Country selected: %s\n", selectedCountry)
	return selectedCountry, nil
}

func promptInt(label string, def int, info string) (int, error) {
	valStr := fmt.Sprint(def)

	if info != "" {
		printTip(info)
	}

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(label).
				Value(&valStr).
				Validate(func(s string) error {
					if s == "" {
						return errors.New("input cannot be empty")
					}
					if _, err := strconv.ParseInt(s, 10, 0); err != nil {
						return errors.New("please enter a valid integer")
					}
					return nil
				}),
		),
	).Run()
	if err != nil {
		return 0, err
	}

	val, _ := strconv.ParseInt(valStr, 10, 0)
	fmt.Printf("%s selected: %d\n", label, val)
	return int(val), nil
}

func promptFloat(label string, def float64, info string) (float64, error) {
	sDef := strconv.FormatFloat(def, 'g', 5, 64)
	valStr := sDef

	if info != "" {
		printTip(info)
	}

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(label).
				Value(&valStr).
				Validate(func(s string) error {
					if s == "" {
						return errors.New("input cannot be empty")
					}
					if _, err := strconv.ParseFloat(s, 64); err != nil {
						return errors.New("please enter a valid float number")
					}
					return nil
				}),
		),
	).Run()
	if err != nil {
		return 0.00, err
	}

	val, _ := strconv.ParseFloat(valStr, 64)
	fmt.Printf("%s selected: %.4f\n", label, val)
	return val, nil
}

func getFlagOrPromptFloat(
	cCtx *cli.Context,
	flagName, prompt, promptInfo string,
	defaultValue float64,
) (float64, error) {
	value := cCtx.Float64(flagName)
	if value == 0.00 {
		return promptFloat(prompt, defaultValue, promptInfo)
	}
	return value, nil
}

func getFlagOrPromptGrowthRate(
	cCtx *cli.Context,
	flagName, prompt, promptInfo string,
	series []int,
) (float64, error) {
	value := cCtx.Float64(flagName)
	if value == 0.00 {
		cagr, _ := calc.CAGR(series)
		return promptFloat(prompt, cagr, promptInfo)
	}
	return value, nil
}

func getFlagOrPromptInt(
	cCtx *cli.Context,
	flagName, prompt, promptInfo string,
	defaultValue int,
) (int, error) {
	value := cCtx.Int(flagName)
	if value == 0 {
		return promptInt(prompt, defaultValue, promptInfo)
	}
	return value, nil
}

func selectDiscountRateOpt() string {
	printTip(
		"There are a few options for calculating a discount rate. Choose which one you would like to use.",
	)

	var selected string
	options := []string{"WACC", "CV Weighted WACC", "Custom Input"}
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Discount Rate Options").
				Options(convertToHuhOptions(options)...).
				Value(&selected),
		),
	).Run()
	if err != nil {
		return ""
	}

	fmt.Printf("Discount Rate Option selected: %s\n", selected)
	return selected
}

// Helper function to convert string slices to huh.Option slices
func convertToHuhOptions(items []string) []huh.Option[string] {
	options := make([]huh.Option[string], len(items))
	for i, item := range items {
		options[i] = huh.NewOption(item, item)
	}
	return options
}
