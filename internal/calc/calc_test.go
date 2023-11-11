package calc

import (
	"fmt"
	"reflect"
	"testing"
)

var (
	beta                = 1.274900
	fcfHistory          = []int{47149000000, 58896000000, 73365000000, 92953000000, 111443000000}
	currentDividend     = 14693400000
	debtToEquity        = 2.369500
	taxRate             = 0.162
	equityRiskPremium   = 0.06
	riskFreeRate        = 0.04
	growthRate          = 0.11689
	exitMultiple        = 32.0
	perpetualGrowthRate = 0.02
	highGrowthYears     = 5
	shares              = 15787154000
	discountRate        = 0.05
)

func Test_WACC(t *testing.T) {
	wacc := WACC(beta, debtToEquity, taxRate, equityRiskPremium, riskFreeRate)

	if wacc != 0.039228168924167195 {
		fmt.Println(wacc)
		t.Fatalf(`WACC(%f, %f, %f, %f, %f) = %f`, beta, debtToEquity, taxRate, equityRiskPremium, riskFreeRate, wacc)
	}
}

func Test_FCFCVWeightedWACC(t *testing.T) {
	wacc := FCFCVWeightedWACC(fcfHistory, debtToEquity, taxRate, equityRiskPremium, riskFreeRate)

	if wacc != 0.03698768876669731 {
		fmt.Println(wacc)
		t.Fatalf(`FCFCVWeightedWACC(%v, %f, %f, %f, %f) = %f`, fcfHistory, debtToEquity, taxRate, equityRiskPremium, riskFreeRate, wacc)
	}
}

func Test_CAGR(t *testing.T) {
	cagr, err := CAGR(fcfHistory)
	if err != nil {
		t.Fatalf("error calculating CAGR: %s", err)
	}

	if cagr != 0.18772544804873736 {
		fmt.Println(cagr)
		t.Fatalf(`CAGR(%v) = %f`, fcfHistory, cagr)
	}
}

func Test_CV(t *testing.T) {
	cv := CV(fcfHistory)

	if cv != 0.30118884965644277 {
		fmt.Println(cv)
		t.Fatalf(`CV(%v) = %f`, fcfHistory, cv)
	}
}

func Test_DCFGrowthExit(t *testing.T) {
	dcf, projected, err := DCFGrowthExit(fcfHistory[0], growthRate, exitMultiple, len(fcfHistory), shares, discountRate)
	if err != nil {
		t.Fatal(err)
	}

	if dcf != 146.29675045735823 {
		fmt.Println(dcf)
		t.Fatalf(`DCFGrowthExit(%d, %f, %f, %d, %d, %f) = %f`, fcfHistory[0], growthRate, exitMultiple, len(fcfHistory), shares, discountRate, dcf)
	}

	if !reflect.DeepEqual(projected, []int{47149000000, 52660246610, 58815702836, 65690670340, 73369252796}) {
		fmt.Println(projected)
		t.Fatalf(`DCFGrowthExit(%d, %f, %f, %d, %d, %f) = %+v`, fcfHistory[0], growthRate, exitMultiple, len(fcfHistory), shares, discountRate, projected)
	}
}

func Test_DCFTwoStage(t *testing.T) {
	dcf, projected, err := DCFTwoStage(fcfHistory[0], growthRate, perpetualGrowthRate, highGrowthYears, shares, discountRate)
	if err != nil {
		t.Fatal(err)
	}

	if dcf != 156.31884569425605 {
		fmt.Println(dcf)
		t.Fatalf(`DCFTwoStage(%d, %f, %f, %d, %d, %f) = %f`, fcfHistory[0], growthRate, perpetualGrowthRate, len(fcfHistory), shares, discountRate, dcf)
	}

	if !reflect.DeepEqual(projected, []int{52660246610, 58815702836, 65690670340, 73369252796, 81945384756}) {
		fmt.Println(projected)
		t.Fatalf(`DCFTwoStage(%d, %f, %f, %d, %d, %f) = %+v`, fcfHistory[0], growthRate, perpetualGrowthRate, len(fcfHistory), shares, discountRate, projected)
	}
}

func Test_DDMTwoStage(t *testing.T) {
	ddm, projected, err := DDMTwoStage(currentDividend, growthRate, perpetualGrowthRate, len(fcfHistory), shares, discountRate)
	if err != nil {
		t.Fatal(err)
	}

	if ddm != 58.953722212615475 {
		fmt.Println(ddm)
		t.Fatalf(`DDMTwoStage(%d, %f, %f, %d, %d, %f) = %f`, currentDividend, growthRate, perpetualGrowthRate, len(fcfHistory), shares, discountRate, ddm)
	}

	if !reflect.DeepEqual(projected, []int{16410911526, 18329182974, 20471681172, 22864615984, 25537260946}) {
		fmt.Println(projected)
		t.Fatalf(`DDMTwoStage(%d, %f, %f, %d, %d, %f) = %+v`, currentDividend, growthRate, perpetualGrowthRate, len(fcfHistory), shares, discountRate, projected)
	}
}
