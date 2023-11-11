package calc

import (
	"fmt"
	"math"
)

// WACC calculates a weighted average cost of capital (WACC) using beta as the measure of risk.
//
// The WACC is a calculation of a company's cost of capital in which each capital source is weighted according to its proportion of the company's capital structure. The WACC is then used to discount future cash flows to calculate the present value of a company.
//
// Arguments:
//
//	beta: The company's beta. Beta is a measure of a stock's volatility in relation to the market as a whole.
//	debtToEquityRatio: The company's debt-to-equity ratio. This is a measure of the company's financial leverage.
//	taxRate: The company's effective tax rate. This is the percentage of income that the company pays in taxes.
//	equityRiskPremium: The equity risk premium. This is the additional return that investors demand for equity investments over and above the risk-free rate.
//	riskFreeRate: The risk-free rate. This is the return that investors can expect to earn on a risk-free investment, such as a government bond.
//
// Returns:
// The company's WACC as a float64.
func WACC(beta float64, debtToEquityRatio float64, taxRate float64, equityRiskPremium float64, riskFreeRate float64) float64 {
	unleveredBeta := beta / (1 + (1-taxRate)*debtToEquityRatio)

	costOfEquity := riskFreeRate + (unleveredBeta * equityRiskPremium)
	costOfDebt := riskFreeRate * (1 - taxRate)

	equityWeight := 1 / (1 + debtToEquityRatio)
	debtWeight := 1 - equityWeight

	wacc := (equityWeight * costOfEquity) + (debtWeight * costOfDebt * (1 - taxRate))

	return wacc
}

// FCFCVWeightedWACC calculates a WACC using the coeficient of variance of FCF in place of beta for a measure of risk.
//
// Arguments:
//
//	fcfHistory: The company's Free Cash Flow History as an array of type int.
//	debtToEquityRatio: The company's debt-to-equity ratio. This is a measure of the company's financial leverage.
//	taxRate: The company's effective tax rate. This is the percentage of income that the company pays in taxes.
//	equityRiskPremium: The equity risk premium. This is the additional return that investors demand for equity investments over and above the risk-free rate.
//	riskFreeRate: The risk-free rate. This is the return that investors can expect to earn on a risk-free investment, such as a government bond.
//
// Returns:
// The company's WACC as a float64.
//
// Note: The use of the coefficient of variance of FCF as a measure of risk in the WACC calculation is a relatively new approach. It is not yet widely accepted.
// TODO: insert study of FCF CV here
func FCFCVWeightedWACC(fcfHistory []int, debtToEquityRatio float64, taxRate float64, equityRiskPremium float64, riskFreeRate float64) float64 {
	fvfCoeficientOfVariance := CV(fcfHistory)

	costOfEquity := riskFreeRate + (fvfCoeficientOfVariance * equityRiskPremium)
	costOfDebt := riskFreeRate * (1 - taxRate)

	equityWeight := 1 / (1 + debtToEquityRatio)
	debtWeight := 1 - equityWeight

	wacc := (equityWeight * costOfEquity) + (debtWeight * costOfDebt * (1 - taxRate))

	return wacc
}

// DCFGrowthExit calculates a DCF analysis using the growth-exit model, taking a standard growth rate and an exit multiple.
//
// Arguments:
//
//	currentFCF: The current free cash flow of the company.
//	growthRate: The annual growth rate of the free cash flow.
//	exitMultiple: The exit multiple to apply to the last year's free cash flow.
//	numYears: The number of years in the growth period.
//	sharesOutstanding: The number of shares outstanding.
//	discountRate: The discount rate to use for the present value calculation.
//
// Returns:
//
//	The intrinsic value of the company, or an error if the number of shares outstanding is zero.
func DCFGrowthExit(currentFCF int, growthRate float64, exitMultiple float64, numYears int, sharesOutstanding int, discountRate float64) (float64, []int, error) {
	if sharesOutstanding <= 0 {
		return 0, nil, fmt.Errorf("number of shares outstanding must be greater than zero")
	}

	var fcfProjections []int
	totalValue := 0.0

	// projected cash flows for each year in the growth period
	for i := 0; i < numYears; i++ {
		projectedFCF := float64(currentFCF) * math.Pow(1+growthRate, float64(i))
		fcfProjections = append(fcfProjections, int(projectedFCF))
		presentValue := projectedFCF / math.Pow(1+discountRate, float64(i+1))
		totalValue += presentValue
	}

	// terminal value using the exit multiple on the last year's FCF
	lastYearFCF := float64(currentFCF) * math.Pow(1+growthRate, float64(numYears))
	terminalValue := lastYearFCF * exitMultiple
	pvTerminalValue := terminalValue / math.Pow(1+discountRate, float64(numYears))

	totalValue += pvTerminalValue

	intrinsicValue := totalValue / float64(sharesOutstanding)

	return intrinsicValue, fcfProjections, nil
}

// DCFTwoStage calculates a DCF analysis using a two-stage model, taking into account a high-growth stage and a terminal, perpetual-growth rate.
//
// Arguments:
//
//	currentFCF: The current free cash flow of the company.
//	growthRate: The annual growth rate of the free cash flow during the high-growth stage.
//	perpetualGrowthRate: The perpetual growth rate of the free cash flow after the high-growth stage.
//	numYears: The number of years in the high-growth stage.
//	sharesOutstanding: The number of shares outstanding.
//	discountRate: The discount rate to use for the present value calculation.
//
// Returns:
//
//	The intrinsic value of the company, or an error if the number of shares outstanding is zero, the discount rate is less than or equal to the perpetual growth rate, or the numYears argument is negative.
func DCFTwoStage(currentFCF int, growthRate float64, perpetualGrowthRate float64, numYears int, sharesOutstanding int, discountRate float64) (float64, []int, error) {
	if sharesOutstanding <= 0 {
		return 0, nil, fmt.Errorf("number of shares outstanding must be greater than zero")
	}
	if discountRate <= perpetualGrowthRate {
		return 0, nil, fmt.Errorf("discount rate must be greater than the perpetual growth rate")
	}

	var fcfProjections []int

	totalValue := 0.0

	// high growth phase
	for i := 1; i <= numYears; i++ {
		projectedFCF := float64(currentFCF) * math.Pow(1+growthRate, float64(i))
		fcfProjections = append(fcfProjections, int(projectedFCF))
		presentValue := projectedFCF / math.Pow(1+discountRate, float64(i))
		totalValue += presentValue
	}

	// stable growth phase
	lastYearFCF := float64(currentFCF) * math.Pow(1+growthRate, float64(numYears))
	terminalValue := (lastYearFCF * (1 + perpetualGrowthRate)) / (discountRate - perpetualGrowthRate)
	pvTerminalValue := terminalValue / math.Pow(1+discountRate, float64(numYears))

	totalValue += pvTerminalValue

	// per share value
	intrinsicValue := totalValue / float64(sharesOutstanding)

	return intrinsicValue, fcfProjections, nil
}

// DDMTwoStage calculates a two-stage DDM using a standard growth rate and a terminal growth rate for the stable-growth period.
//
// Arguments:
//
//	currentDividend: The current dividend per share.
//	growthRate: The annual growth rate of the dividend per share during the high-growth stage.
//	perpetualGrowthRate: The perpetual growth rate of the dividend per share after the high-growth stage.
//	numYears: The number of years in the high-growth stage.
//	sharesOutstanding: The number of shares outstanding.
//	discountRate: The discount rate to use for the present value calculation.
//
// Returns:
//
//	The intrinsic value of the company per share, or an error if the number of shares outstanding is zero, the discount rate is less than or equal to the perpetual growth rate, or the numYears argument is negative.
func DDMTwoStage(currentDividend int, growthRate float64, perpetualGrowthRate float64, numYears int, sharesOutstanding int, discountRate float64) (float64, []int, error) {
	if sharesOutstanding <= 0 {
		return 0, nil, fmt.Errorf("number of shares outstanding must be greater than zero")
	}
	if discountRate <= perpetualGrowthRate {
		return 0, nil, fmt.Errorf("discount rate must be greater than the perpetual growth rate")
	}

	totalValue := 0.0
	var projectedDividends []int

	// Calculate present value of dividends for the high-growth stage
	dividends := float64(currentDividend)
	for i := 1; i <= numYears; i++ {
		presentValue := dividends / math.Pow(1+discountRate, float64(i))
		totalValue += presentValue
		dividends *= (1 + growthRate)
		projectedDividends = append(projectedDividends, int(dividends))
	}

	// Calculate terminal value using the Gordon Growth Model
	terminalValue := dividends / (discountRate - perpetualGrowthRate)

	// Calculate intrinsic value per share
	intrinsicValue := (totalValue + terminalValue) / float64(sharesOutstanding)

	return intrinsicValue, projectedDividends, nil
}

// CV calculates the coefficient of variance of an array of type int.
//
// Arguments:
//
//	values: An array of type int.
//
// Returns:
//
//	The coefficient of variance of the array.
func CV(values []int) float64 {
	var std, sum float64

	for i := 1; i <= len(values); i++ {
		sum += float64(values[i-1])
	}

	mean := sum / float64(len(values))

	for i := 0; i < len(values); i++ {
		std += math.Pow(float64(float64(values[i])-mean), 2)
	}

	std = math.Sqrt(std / float64(len(values)))

	cv := (std / float64(mean))

	return cv
}

// CAGR calculates the compounded annual growth rate of an array of ints (we assume it to be annual).
//
// Arguments:
//
//	values: An array of ints.
//
// Returns:
//
//	cagr: The compounded annual growth rate, as a float64.
//	err: An error, if any.
func CAGR(values []int) (float64, error) {
	n := len(values)

	if n < 2 {
		return 0, fmt.Errorf("CAGR cannot be calculated with less than 2 values - check input: %v", values)
	}

	initialValue := float64(values[0])
	finalValue := float64(values[n-1])
	numYears := float64(n)

	if initialValue == 0 {
		return 0, fmt.Errorf("initial value is zero, CAGR calculation is not possible - check input: %v", values)
	}

	var cagr float64

	switch {
	case initialValue > 0 && finalValue >= 0:
		cagr = math.Pow(finalValue/initialValue, 1.0/numYears) - 1
	case initialValue < 0 && finalValue <= 0:
		cagr = -1 * (math.Pow(math.Abs(finalValue)/math.Abs(initialValue), 1.0/numYears) - 1)
	case initialValue < 0 && finalValue > 0:
		cagr = math.Pow((finalValue+2*math.Abs(initialValue))/math.Abs(initialValue), 1.0/numYears) - 1
	case initialValue > 0 && finalValue < 0:
		cagr = -1 * (math.Pow((math.Abs(finalValue)+2*initialValue)/initialValue, 1.0/numYears) - 1)
	default:
		cagr = 0
	}

	return cagr, nil
}
