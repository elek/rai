package cmd

import (
	"fmt"

	"charm.land/fantasy"
	"github.com/elek/catwalk-open/providers"
)

func printUsage(fm fantasy.LanguageModel, usage fantasy.Usage) {
	fmt.Println("model:", fm.Model())
	fmt.Println("usage:", usage)
	for _, provider := range providers.GetAll() {
		if string(provider.ID) != fm.Provider() {
			continue
		}
		for _, model := range provider.Models {
			if model.ID == fm.Model() {
				fmt.Println("Costs (1m):", model.CostPer1MIn, "USD in,", model.CostPer1MOut, "USD out")
				fmt.Printf("cost: %0.02f USD", model.CostPer1MIn*float64(usage.InputTokens)/1_000_000+model.CostPer1MOut*float64(usage.OutputTokens)/1_000_000)
				break
			}
		}
	}
}
