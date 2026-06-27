package cmd

import (
	"fmt"

	"github.com/elek/catwalk-open/providers"
	"github.com/elek/rai/llm"
)

func printUsage(model llm.Model, usage llm.Usage) {
	fmt.Println("model:", model.Name())
	fmt.Println("usage:", usage)
	for _, provider := range providers.GetAll() {
		if string(provider.ID) != model.Provider() {
			continue
		}
		for _, m := range provider.Models {
			if m.ID == model.Name() {
				fmt.Println("Costs (1m):", m.CostPer1MIn, "USD in,", m.CostPer1MOut, "USD out")
				fmt.Printf("cost: %0.02f USD", m.CostPer1MIn*float64(usage.InputTokens)/1_000_000+m.CostPer1MOut*float64(usage.OutputTokens)/1_000_000)
				break
			}
		}
	}
}
