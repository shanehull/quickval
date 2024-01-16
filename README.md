# quickval

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report](https://goreportcard.com/badge/github.com/shanehull/quickval)](https://goreportcard.com/report/github.com/shanehull/quickval)

[![Test Workflow](https://github.com/shanehull/quickval/actions/workflows/test.yaml/badge.svg?branch=main)](https://github.com/shanehull/quickval/actions/workflows/test.yaml/badge.svg?branch=main)
[![Lint Workflow](https://github.com/shanehull/quickval/actions/workflows/lint.yaml/badge.svg?branch=main)](https://github.com/shanehull/quickval/actions/workflows/lint.yaml/badge.svg?branch=main)
[![Release Workflow](https://github.com/shanehull/quickval/actions/workflows/release.yaml/badge.svg?branch=release)](https://github.com/shanehull/quickval/actions/workflows/release.yaml/badge.svg?branch=release)

`quickval` is an interactive CLI tool that leverages the free [QuickFS.net API](https://quickfs.net/features/public-api) to step through security valuations.

<p align="center">
    <img src="docs/images/quickval.gif" width="800" alt="quickval cmd line example">
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

```
NAME:
   quickval - Perform quick valuations using the QuickFS API

USAGE:
   quickval [global options] command [command options]

COMMANDS:
   growth-exit, dcf, dcfe  Performs a growth-exit DCF model.
   two-stage, dcf2, dcfp   Performs a two-stage DCF model.
   dividend, ddm           Performs a two-stage DDM model.
   help, h                 Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --api-key value  api key for QuickFS API
   --country value  country code for the ticker
   --ticker value   ticker to base our valuation on
   --help, -h       show help
```

Subcommands require some unique inputs and will prompt you if not supplied via CLI arguments.

E.g; the growth-exit model takes the following args, but will prompt and suggest defaults (e.g. a CAGR for the growth rate) that may or may not need to be tweaked, depending on your requirements:

```
NAME:
   quickval growth-exit - Performs a growth-exit DCF model.

USAGE:
   quickval growth-exit [command options] [arguments...]

DESCRIPTION:
   Performs a growth-exit DCF model with a high-growth stage and an exit multiple.

OPTIONS:
   --risk-free value     the risk-free rate in decimal format (default: 0)
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

It uses a Coefficient of Variance - a measure of relative variance in comparison to the mean of a set of numbers.
In this case, the set of numbers is Free Cash Flow, or Dividends paid when performing a DDM valuation model.

It is calculated like so:

$$CV = (a / X)$$

$$
Where \ a = \ Standard \ Deviation
$$

$$
and
$$

$$
X = \ Mean
$$

:warning: NOTE

This is an experimental feature, and there is quite a lot wrong with it, namely the small sample size used to calculate variance.
It may not be any better than a WACC calculated using the CAPM model.

I emailed Aswath Damodaran ("The Dean of Valuation") on the subject, and he said, quote:

"The problem with using free cash flows or accounting earnings to measure risk is both statistical and theoretical.
Statistically, you don’t have very many observations and pragmatically, in a diversified portfolio,
it is only the portion of the risk that you cannot diversify away that goes into a discount rate.
Hence, if you decide to compute your risk using it, you need to scale it to the average to get a measure of relative risk."

I tend to agree with him, however, in practice, using this method over a CAPM WACC doesn't tip the scales on your final number.

No matter the methods used to measure risk, you should not be mistaking a DCF calculation for an accurate indication of future price.
