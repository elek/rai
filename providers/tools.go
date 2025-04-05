package providers

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

func debug(name string, payload any) {
	indent, _ := json.MarshalIndent(payload, "", "  ")
	_ = os.MkdirAll("/tmp/rai/debug", 0755)
	outFile := fmt.Sprintf("/tmp/rai/debug/%s-%s.json", time.Now().Format(time.RFC3339Nano), name)
	fmt.Println("Saving debug output to", outFile)
	err := os.WriteFile(outFile, indent, 0644)
	if err != nil {
		panic(err)
	}
}
