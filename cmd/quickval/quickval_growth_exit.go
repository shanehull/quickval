package main

import (
	"math"
	"os"

	"github.com/shanehull/quickval/internal/calc"
	"github.com/shanehull/quickval/internal/output"
	"github.com/shanehull/quickval/internal/quickfs"
	"github.com/urfave/cli/v2"
)

var growthExitCommand = &cli.Command{
	Name:        "growth-exit",
	Aliases:     []string{"dcf", "dcfe"},
	Description: "Performs a growth-exit DCF model with a high-growth stage and an exit multiple.",
	Usage:       "Performs a growth-exit DCF model.",
	Flags: []cli.Flag{
		&cli.Float64Flag{
			Name:  "risk-free",
			Value: 0.00,
			Usage: "the risk-free rate in decimal format",
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

		growthRate, err := getFlagOrPromptGrowthRate(
			cCtx,
			"growth-rate",
			"Growth Rate",
			growthPromptInfo,
			data.FCFHistory,
		)
		if err != nil {
			return err
		}

		currentFCF, err := getFlagOrPromptInt(
			cCtx,
			"current-fcf",
			"Current FCF",
			fcfPromptInfo,
			data.FCFHistory[len(data.FCFHistory)-1],
		)
		if err != nil {
			return err
		}

		currentMultipleFloor := math.Floor(
			data.Price / (float64(currentFCF) / float64(data.Shares)),
		)

		exitMultiple, err := getFlagOrPromptFloat(
			cCtx,
			"exit-multiple",
			"Exit Multiple",
			exitPromptInfo,
			currentMultipleFloor,
		)
		if err != nil {
			return err
		}

		expectedReturn, err := calc.ExpectedReturn(
			growthRate,
			float64(currentFCF)/float64(data.Shares),
			data.Price,
		)
		if err != nil {
			return err
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

		upside, err := calc.Upside(fairValue, data.Price)
		if err != nil {
			return err
		}

		writer.Projected(projectedFCF, growthRate, expectedReturn, upside)
		writer.FairValue(fairValue)
		writer.Render()

		return nil
	},
}
