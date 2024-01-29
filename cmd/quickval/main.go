package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/urfave/cli/v2"
)

var (
	apiKey  string
	country string
	ticker  string
)

var bashCompletionsMode bool

func main() {
	if os.Args[len(os.Args)-1] == "--generate-bash-completion" {
		bashCompletionsMode = true
	}

	if err := app.RunContext(context.Background(), os.Args); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

var app = &cli.App{
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
