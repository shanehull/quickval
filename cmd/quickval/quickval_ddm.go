package main

import (
	"errors"
	"os"

	"github.com/shanehull/quickval/internal/calc"
	"github.com/shanehull/quickval/internal/output"
	"github.com/shanehull/quickval/internal/quickfs"
	"github.com/urfave/cli/v2"
)

var dividendDiscountCommand = &cli.Command{
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

		if len(data.CFFDividends) < 1 {
			return errors.New("no dividend history")
		}

		growthRate := getFlagOrPromptGrowthRate(cCtx, "growth-rate", "Growth Rate", growthPromptInfo, data.CFFDividends)
		currentDividends := getFlagOrPromptInt(cCtx, "current-dividends", "Current Cash Paid for Dividends", dividendsPromptInfo, data.CFFDividends[len(data.CFFDividends)-1])
		perpetualRate := getFlagOrPromptFloat(cCtx, "perpetual-rate", "Perpetual Growth Rate", perpetualGrowthInfo, 0.02)

		fairValue, projectedDividends, err := calc.DDMTwoStage(
			currentDividends,
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
}
