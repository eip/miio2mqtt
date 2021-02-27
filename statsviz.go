// +build stats

package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/arl/statsviz"
)

func init() {
	fmt.Println("running statsviz at http://localhost:6060/debug/statsviz/")
	go func() {
		statsviz.RegisterDefault(statsviz.SendFrequency((10 * time.Second)))
		fmt.Println(http.ListenAndServe(":6060", nil))
		os.Exit(1)
	}()
}
