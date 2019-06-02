package main

import (
	"net/http"
	"strconv"

	flag "github.com/spf13/pflag"

	. "github.com/nektro/go-util/alias"
)

func main() {
	Log("Initializing Skarn Request System...")

	flagPort := flag.Int("port", 8000, "Port to open server on")
	flag.Parse()

	//

	p := strconv.Itoa(*flagPort)
	Log("Initialization complete. Starting server on port " + p)
	http.ListenAndServe(":"+p, nil)
}
