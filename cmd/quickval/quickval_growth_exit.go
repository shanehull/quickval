package main

import (
	"os"

	"github.com/shanehull/quickval/internal/calc"
	"github.com/shanehull/quickval/internal/output"
	"github.com/shanehull/quickval/internal/quickfs"
	"github.com/urfave/cli/v2"
)

var growthExitCommand = &cli.Command{
	Name:        "growth-exit",
	Aliases:     []string{"dcf", "dcfe"},
	Description: "a growth-exit model with a high-growth stage and an exit multiple",
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

		growthRate, err := getFlagOrPromptGrowthRate(cCtx, "growth-rate", "Growth Rate", growthPromptInfo, data.FCFHistory)
		if err != nil {
			return err
		}
		currentFCF, err := getFlagOrPromptInt(cCtx, "current-fcf", "Current FCF", fcfPromptInfo, data.FCFHistory[len(data.FCFHistory)-1])
		if err != nil {
			return err
		}
		exitMultiple, err := getFlagOrPromptFloat(cCtx, "exit-multiple", "Exit Multiple", exitPromptInfo, defaultExitMultiple)
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

		writer.Projected(projectedFCF, growthRate)
		writer.FairValue(fairValue)
		writer.Render()

		return nil
	},
}
