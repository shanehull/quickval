package main

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	"github.com/urfave/cli/v2"
)

var (
	apiKey  string
	country string
	ticker  string
)

var (
	searchTickers   []string
	cacheDir        = "/tmp/quickval"
	ghTickersURLFmt = "https://raw.githubusercontent.com/shanehull/quickval/main/tickers/%s.json"
)

var (
	rfrPromptInfo       = "Enter a Risk-Rree Rate (e.g a 10-year average of a 10-year treasury bond yield)."
	erpPromptInfo       = "Enter an ERP (Equity Risk Premium)."
	discPromptInfo      = "Enter an explicit discount rate."
	growthPromptInfo    = "Enter a reasonable growth rate, or accept the default (derived from a CAGR of the FCF or dividend history)."
	fcfPromptInfo       = "Enter a current FCF (e.g. a normalised figure) or accept the most recent reported figure."
	dividendsPromptInfo = "Enter a current Cash Paid for Dividends value (e.g. a normalised figure) or accept the most recent reported figure."
	exitPromptInfo      = "Enter an appropriate exit multiple."
	perpetualGrowthInfo = "Enter a growth rate for the perpetual/terminal growth stage."
	fyHistoryPromptInfo = "Enter a FY history to retrieve for financial reports."
)

var bashCompletionsMode bool

func main() {
	if os.Args[len(os.Args)-1] == "--generate-bash-completion" {
		bashCompletionsMode = true
	}

	app := &cli.App{
		Usage:                "Perform quick valuations using the QuickFS API",
		EnableBashCompletion: true,
		Compiled:             time.Now(),
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
		Before: func(cCtx *cli.Context) error {
			if bashCompletionsMode {
				return nil
			}

			if cCtx.NArg() == 0 {
				err := cli.ShowAppHelp(cCtx)
				if err != nil {
					return err
				}
				return errors.New("a command is required")
			}

			return setCommonVars(cCtx)
		},
		Commands: []*cli.Command{
			growthExitCommand,
			twoStageCommand,
			dividendDiscountCommand,
		},
	}

	if err := app.RunContext(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
