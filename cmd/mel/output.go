package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mel-project/mel/internal/cliout"
)

func mustPrint(v any) {
	if cliGlobal.JSON {
		if err := cliout.Print(os.Stdout, v); err != nil {
			panic(err)
		}
		return
	}
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
}
