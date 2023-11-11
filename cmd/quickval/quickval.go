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

var (
	apiKey  string
	country string
	ticker  string
)

var (
	countryCodes    = []string{"US", "AT", "AU", "BE", "CA", "CH", "DE", "DK", "ES", "FI", "FI", "FR", "GR", "IT", "LN", "MM", "NL", "NO", "NZ", "PL", "SE"}
	searchTickers   []string
	cacheDir        = "/tmp/quickval"
	ghTickersURLFmt = "https://raw.githubusercontent.com/shanehull/quickval-wip/main/tickers/%s.json"
)

var (
	rfrPromptInfo       = "Enter a risk-free rate (e.g a 10-year average of a 10-year treasury bond yield)."
	erpPromptInfo       = "Enter an ERP (Equity Risk Premium)."
	discPromptInfo      = "Enter an explicit discount rate."
	growthPromptInfo    = "Enter a reasonable growth rate, or accept the default (derived from a CAGR of the FCF or dividend history)."
	fcfPromptInfo       = "Enter a current FCF (e.g. a normalised figure) or accept the most recent reported figure."
	dividendsPromptInfo = "Enter a current Cash Paid for Dividends value (e.g. a normalised figure) or accept the most recent reported figure."
	exitPromptInfo      = "Enter an appropriate exit multiple."
	perpetualGrowthInfo = "Enter a growth rate for the perpetual/terminal growth stage."
	fyHistoryPromptInfo = "Enter a FY history to retrieve for financial reports."
)

func main() {

	app := &cli.App{
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "api-key",
				Value: "",
				Usage: "api key for QuickFS API",
			},
			&cli.StringFlag{
				Name:  "country",
				Value: "",
				Usage: "country code for the ticker",
			},
			&cli.StringFlag{
				Name:  "ticker",
				Value: "",
				Usage: "ticker to base our valuation on",
			},
		},
		Commands: []*cli.Command{
			{
				Name:        "growth-exit",
				Aliases:     []string{"dcf", "dcfe"},
				Description: "a growth-exit model with a high-growth stage and an exit mulitple",
				Flags: []cli.Flag{
					&cli.Float64Flag{
						Name:  "risk-free",
						Value: 0.00,
						Usage: "the risk free rate in decimal format",
					},
					&cli.Float64Flag{
						Name:  "risk-premium",
						Value: 0.00,
						Usage: "the equity risk premium rate in decimal format",
					},
					&cli.IntFlag{
						Name:  "current-fcf",
						Value: 0,
						Usage: "override the current FCF with a normalized number",
					},
					&cli.Float64Flag{
						Name:  "growth-rate",
						Value: 0.00,
						Usage: "override the growth rate with your own number",
					},
					&cli.IntFlag{
						Name:  "fy-history",
						Value: 0,
						Usage: "override the growth rate with your own number",
					},
				},
				Action: func(cCtx *cli.Context) error {
					writer := output.NewWriter(os.Stdout)

					data, fyHistory, discountRate, err := doCommonSetup(cCtx, writer, quickfs.WithFCF())
					if err != nil {
						return err
					}

					growthRate := cCtx.Float64("growth-rate")
					if growthRate == 0.00 {
						cagr, err := calc.CAGR(data.FCFHistory)
						if err != nil {
							return err
						}

						growthRate = promptFloat("Growth Rate", cagr, growthPromptInfo)
					}

					currentFCF := cCtx.Int("current-fcf")
					var recentFCF int
					if currentFCF == 0 {
						recentFCF = data.FCFHistory[len(data.FCFHistory)-1]
						currentFCF = promptInt("Current FCF", recentFCF, fcfPromptInfo)
					}

					exitMultiple := cCtx.Float64("exit-multiple")
					if exitMultiple == 0.00 {
						exitMultiple = promptFloat("Exit Multiple", 16.0, exitPromptInfo)
					}

					fairValue, projectedFCF, err := calc.DCFGrowthExit(
						currentFCF,
						growthRate,
						exitMultiple,
						fyHistory,
						data.Shares,
						discountRate,
					)
					if err != nil {
						return err
					}

					writer.Projected(projectedFCF, growthRate)
					writer.FairValue(fairValue)
					writer.Render()

					return nil
				},
			},
			{
				Name:        "two-stage",
				Aliases:     []string{"dcf2", "dcfp"},
				Description: "a perpetuity growth model with a high-growth stage and a perpetual growth stage",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "ticker",
						Value: "",
						Usage: "ticker to base our valuation on",
					},
					&cli.Float64Flag{
						Name:  "risk-free",
						Value: 0.00,
						Usage: "the risk free rate in decimal format",
					},
					&cli.Float64Flag{
						Name:  "risk-premium",
						Value: 0.00,
						Usage: "the equity risk premium rate in decimal format",
					},
					&cli.IntFlag{
						Name:  "current-fcf",
						Value: 0,
						Usage: "current free cash flow of the company",
					},
					&cli.Float64Flag{
						Name:  "growth-rate",
						Value: 0.00,
						Usage: "annual growth rate of the free cash flow during the high-growth stage",
					},
					&cli.Float64Flag{
						Name:  "perpetual-rate",
						Value: 0.00,
						Usage: "perpetual growth rate of the free cash flow after the high-growth stage",
					},
					&cli.IntFlag{
						Name:  "num-years",
						Value: 0,
						Usage: "number of years in the high-growth stage",
					},
					&cli.IntFlag{
						Name:  "fy-history",
						Value: 0,
						Usage: "FY history to retrieve for financial reports",
					},
				},
				Action: func(cCtx *cli.Context) error {
					writer := output.NewWriter(os.Stdout)

					data, fyHistory, discountRate, err := doCommonSetup(cCtx, writer, quickfs.WithFCF())
					if err != nil {
						return err
					}

					growthRate := cCtx.Float64("growth-rate")
					if growthRate == 0.00 {
						cagr, err := calc.CAGR(data.FCFHistory)
						if err != nil {
							return err
						}

						growthRate = promptFloat("Growth Rate", cagr, growthPromptInfo)
					}

					currentFCF := cCtx.Int("current-fcf")
					var recentFCF int
					if currentFCF == 0 {
						recentFCF = data.FCFHistory[len(data.FCFHistory)-1]
						currentFCF = promptInt("Current FCF", recentFCF, fcfPromptInfo)
					}

					perpetualRate := cCtx.Float64("perpetual-rate")
					if perpetualRate == 0.00 {
						perpetualRate = promptFloat("Perpetual Growth Rate", 0.02, perpetualGrowthInfo)
					}

					fairValue, projectedFCF, err := calc.DCFTwoStage(
						currentFCF,
						growthRate,
						perpetualRate,
						fyHistory,
						data.Shares,
						discountRate,
					)
					if err != nil {
						return err
					}

					writer.Projected(projectedFCF, growthRate)
					writer.FairValue(fairValue)
					writer.Render()
					return nil
				},
			},
			{
				Name:        "dividend",
				Aliases:     []string{"ddm"},
				Description: "a two-stage dividend discount model",
				Flags: []cli.Flag{
					&cli.Float64Flag{
						Name:  "risk-free",
						Value: 0.00,
						Usage: "the risk free rate in decimal format",
					},
					&cli.Float64Flag{
						Name:  "risk-premium",
						Value: 0.00,
						Usage: "the equity risk premium rate in decimal format",
					},
					&cli.IntFlag{
						Name:  "current-dividends",
						Value: 0,
						Usage: "current cash paid for dividends paid of the company",
					},
					&cli.Float64Flag{
						Name:  "growth-rate",
						Value: 0.00,
						Usage: "annual growth rate of the free cash flow during the high-growth stage",
					},
					&cli.Float64Flag{
						Name:  "perpetual-rate",
						Value: 0.00,
						Usage: "perpetual growth rate of the free cash flow after the high-growth stage",
					},
					&cli.IntFlag{
						Name:  "num-years",
						Value: 0,
						Usage: "number of years in the high-growth stage",
					},
					&cli.IntFlag{
						Name:  "fy-history",
						Value: 0,
						Usage: "FY history to retrieve for financial reports",
					},
				},
				Action: func(cCtx *cli.Context) error {
					writer := output.NewWriter(os.Stdout)

					data, fyHistory, discountRate, err := doCommonSetup(cCtx, writer, quickfs.WithCFFDividends())
					if err != nil {
						return err
					}

					growthRate := cCtx.Float64("growth-rate")
					if growthRate == 0.00 {
						cagr, err := calc.CAGR(data.CFFDividends)
						if err != nil {
							return err
						}

						growthRate = promptFloat("Growth Rate", cagr, growthPromptInfo)
					}

					fmt.Printf("%+v", data)

					if len(data.CFFDividends) < 1 {
						return errors.New("no dividend history")
					}

					currentFCF := cCtx.Int("current-dividends")
					var recentDividends int
					if currentFCF == 0 {
						recentDividends = data.CFFDividends[len(data.CFFDividends)-1]
						currentFCF = promptInt("Current Cash Paid for Dividends", recentDividends, dividendsPromptInfo)
					}

					perpetualRate := cCtx.Float64("perpetual-rate")
					if perpetualRate == 0.00 {
						perpetualRate = promptFloat("Perpetual Growth Rate", 0.02, perpetualGrowthInfo)
					}

					fairValue, projectedDividends, err := calc.DDMTwoStage(
						currentFCF,
						growthRate,
						perpetualRate,
						fyHistory,
						data.Shares,
						discountRate,
					)
					if err != nil {
						return err
					}

					writer.Projected(projectedDividends, growthRate)
					writer.FairValue(fairValue)
					writer.Render()
					return nil
				},
			},
		},
		Before: func(cCtx *cli.Context) error {
			if cCtx.NumFlags() > 0 && cCtx.NArg() == 0 {
				err := cli.ShowAppHelp(cCtx)
				if err != nil {
					return err
				}
				return errors.New("a command is required")
			}

			return setCommonVars(cCtx)
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

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
				riskFreeRate = promptFloat("Risk Free Rate", 0.02, rfrPromptInfo)
			}

			mergedOpts := append(opts,
				quickfs.WithAPIKey(apiKey),
				quickfs.WithFYHistory(fyHistory),
				quickfs.WithBeta(),
				quickfs.WithDebtToEquity(),
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
		case "FCF-CV Weighted WACC":
			equityRiskPremium = cCtx.Float64("risk-premium")
			if equityRiskPremium == 0.0 {
				equityRiskPremium = promptFloat("Equity Risk Premium", 0.05, erpPromptInfo)
			}

			riskFreeRate = cCtx.Float64("risk-free")
			if riskFreeRate == 0.0 {
				riskFreeRate = promptFloat("Risk Free Rate", 0.02, rfrPromptInfo)
			}

			qfs := quickfs.NewQuickFS(
				quickfs.WithAPIKey(apiKey),
				quickfs.WithFYHistory(fyHistory),
				quickfs.WithFCF(),
				quickfs.WithDebtToEquity(),
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
			qfs := quickfs.NewQuickFS(
				quickfs.WithAPIKey(apiKey),
				quickfs.WithFYHistory(fyHistory),
				quickfs.WithFCF(),
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
		if err := json.Unmarshal(data, &searchTickers); err == nil {
			// data successfully loaded from local cache

			// refresh local cache in the background
			if err := updateLocalCache(country, cacheFilePath); err != nil {
				log.Printf("failed to update local cache: %s", err)
			}
			return searchTickers, nil
		}
	}

	// if no local cache, get them from the repo - quickfs is too slow
	searchTickers, err := fetchTickersFromGH(country)
	if err != nil {
		return nil, err
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

	var tickers []string
	if err := json.NewDecoder(resp.Body).Decode(&tickers); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return tickers, nil
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
	validate := func(input string) error {
		r, _ := regexp.Compile("^[a-z0-9]{40}$")

		m := r.MatchString(input)
		if !m {
			return errors.New("invalid api key")
		}
		return nil
	}

	s := promptui.Prompt{
		Label:    "API Key",
		Validate: validate,
		Mask:     '*',
		// HideEntered: true,
	}

	fmt.Println()
	fmt.Println("Enter a valid API key for QuickFS.")

	response, err := s.Run()

	if err != nil {
		log.Fatalf("an error occurred when setting the api key: %s", err)
	}
	return response
}

func selectTicker(country string, apiKey string) string {
	tickers, err := fetchTickers(country)
	if err != nil {
		log.Fatalf("an error occurred when fetching ticker: %s", err)
	}

	searcher := func(input string, index int) bool {
		ticker := strings.ToLower(tickers[index])
		input = strings.Replace(strings.ToLower(input), " ", "", -1)
		return strings.Contains(ticker, input)
	}

	fmt.Println()
	fmt.Println("Start typing to find your ticker.")

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

	searcher := func(input string, index int) bool {
		ticker := strings.ToLower(countryCodes[index])
		input = strings.Replace(strings.ToLower(input), " ", "", -1)
		return strings.Contains(ticker, input)
	}

	s := promptui.Select{
		Label:             "Country",
		Items:             countryCodes,
		Searcher:          searcher,
		StartInSearchMode: true,
	}

	fmt.Println()
	fmt.Println("Select the country that your ticker trades in.")

	_, response, err := s.Run()
	if err != nil {
		log.Fatalf("an error occurred when setting the country: %s", err)
	}

	return response
}

func promptInt(label string, def int, info string) int {
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

	if info != "" {
		fmt.Println()
		fmt.Println(info)
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
		Default:   string(sDef),
	}

	if info != "" {
		fmt.Println()
		fmt.Println(info)
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

func selectDiscountRateOpt() string {
	s := promptui.Select{
		Label: "Discount Rate Options",
		Items: []string{"WACC", "FCF-CV Weighted WACC", "Custom Input"},
	}

	fmt.Println()
	fmt.Println("There are a few options for calculating a discount rate. Choose which one you would like to use.")

	_, response, err := s.Run()
	if err != nil {
		log.Fatalf("an error occurred when selecting a discount rate option: %s", err)
	}

	return response
}
