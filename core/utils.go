package core

import (
	"encoding/json"
	"fmt"
	"os"
)

var IsLambda = os.Getenv("AWS_LAMBDA_RUNTIME_API") != ""

func Die(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if IsLambda {
		panic(msg)
	} else {
		fmt.Fprintln(os.Stderr, msg)
		os.Exit(1)
	}
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
