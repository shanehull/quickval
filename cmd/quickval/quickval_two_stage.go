package main

import (
	"os"

	"github.com/shanehull/quickval/internal/calc"
	"github.com/shanehull/quickval/internal/output"
	"github.com/shanehull/quickval/internal/quickfs"
	"github.com/urfave/cli/v2"
)

var twoStageCommand = &cli.Command{
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

		growthRate := getFlagOrPromptGrowthRate(cCtx, "growth-rate", "Growth Rate", growthPromptInfo, data.FCFHistory)
		currentFCF := getFlagOrPromptInt(cCtx, "current-fcf", "Current FCF", growthPromptInfo, data.FCFHistory[len(data.FCFHistory)-1])
		perpetualRate := getFlagOrPromptFloat(cCtx, "perpetual-rate", "Perpetual Growth Rate", perpetualGrowthInfo, defaultPerpetualRate)

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
}
