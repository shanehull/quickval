package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/shanehull/quickval/internal/quickfs"
)

var _apiKey string

func init() {
	flag.StringVar(&_apiKey, "api-key", "", "quickfs api key")
}

func main() {
	flag.Parse()

	qfs := quickfs.NewQuickFS(quickfs.WithAPIKey(_apiKey))

	for _, code := range quickfs.CountryCodes {
		start := time.Now()
		comp, err := qfs.GetCompanies(code)
		if err != nil {
			log.Error().Msgf("error fetching tickers for country code %s: %s", code, err)
			continue
		}

		file, _ := json.MarshalIndent(comp, "", "   ")
		fn := fmt.Sprintf("tickers/%s.json", code)
		if err := os.WriteFile(fn, file, os.FileMode(0o664)); err != nil {
			log.Error().Msgf("error writing file for country code %s: %s", code, err)
			continue
		}

		log.Info().
			Int("took_ms", int(time.Since(start).Milliseconds())).
			Msgf("successfully fetched tickers for %s", code)
	}
}
