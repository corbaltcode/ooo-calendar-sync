package core

import (
	"encoding/json"
	"fmt"
	"os"
)

func Die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}

func PrettyJSON(b []byte) (string, error) {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return "", err
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(out), nil
}
