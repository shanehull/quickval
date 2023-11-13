# quickval

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report](https://goreportcard.com/badge/github.com/shanehull/quickval)](https://goreportcard.com/report/github.com/shanehull/quickval)

[![Lint Workflow](https://github.com/shanehull/quickval/actions/workflows/lint.yaml/badge.svg?branch=release)](https://github.com/shanehull/quickval/actions/workflows/lint.yaml/badge.svg?branch=release)
[![Release Workflow](https://github.com/shanehull/quickval/actions/workflows/release.yaml/badge.svg?branch=release)](https://github.com/shanehull/quickval/actions/workflows/release.yaml/badge.svg?branch=release)

`quickval` is an interactive CLI tool that leverages the free [QuickFS.net API](https://quickfs.net/features/public-api) to step through security valuations.

<p align="center">
    <img src="docs/images/quickval.gif" width="700" alt="quickval cmd line example">
</p>

### Supported Valuation Models:

- DCF Growth-Exit Model
- DCF Two-Stage Perpetual Growth Model
- DDM Two-Stage Perpetual Growth Model

### Disclaimer:

Like any valuation model that attempts to predict future business prospects, `quickval` is not an accurate representation of future value. It serves as a yardstick measure of future value based on historical inputs, not future value.

If you're looking to determine the true value of a company, well that's just not possible, so only use this as one of many inputs to determine a best guess.

### Usage:

You can simply run `quickval` with no arguments to get started, however, to avoid being prompted for certain inputs, you can add arguments to the global command, e.g:

```bash
NAME:
   quickval - Perform quick valuations using QuickFS API

USAGE:
   quickval [global options] command [command options] [arguments...]

COMMANDS:
   growth-exit, dcf, dcfe
   two-stage, dcf2, dcfp
   dividend, ddm
   help, h                 Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --api-key value  api key for QuickFS API
   --country value  country code for the ticker
   --ticker value   ticker to base our valuation on
   --help, -h       show help
```

Subcommands require some unique inputs and will prompt you if not supplied via CLI arguments.

E.g; the growth-exit model takes the following args, but will prompt and suggest defaults (e.g. a CAGR for the growth rate) that may or may not need to be tweaked, depending on your requirements:

```bash
NAME:
   quickval growth-exit

USAGE:
   quickval growth-exit [command options] [arguments...]

DESCRIPTION:
   a growth-exit model with a high-growth stage and an exit multiple

OPTIONS:
   --risk-free value     the risk free rate in decimal format (default: 0)
   --risk-premium value  the equity risk premium rate in decimal format (default: 0)
   --current-fcf value   override the current FCF with a normalized number (default: 0)
   --growth-rate value   override the growth rate with your own number (default: 0)
   --fy-history value    override the growth rate with your own number (default: 0)
   --help, -h            show help
```

### CV (Coefficient of Variance) Weighted WACC:

You may notice an option when selecting the Discount Rate calculation method called "CV Weighted WACC".

This is an alternative, experimental option for weighing the Cost of Capital. It's a replacement for the "preposterous" (in Seth Klarman's words) use of Beta as a measure of risk.

It aims to gain a value edge, ignoring price altogether.

It uses a Coefficient of Variance - a measure of relative variance in comparison to the mean of a set of numbers. In this case, the set of numbers is Free Cash Flow, or Dividends paid when performing a DDM valuation model.

It is calculated like so:

$$CV = (a / X) * 100$$

$$
Where \ a = \ Standard \ Deviation
\\\
and
\\\
X = \ Mean
$$

More studies to come on this concept in the future...

$$
$$
