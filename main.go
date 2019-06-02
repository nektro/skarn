package main

import (
	"net/http"
	"path/filepath"
	"strconv"

	flag "github.com/spf13/pflag"

	. "github.com/nektro/go-util/alias"
	. "github.com/nektro/go-util/util"
)

var (
	dataRoot       string
)
func main() {
	Log("Initializing Skarn Request System...")

	flagRoot := flag.String("root", "", "Path of root directory for files")
	flagPort := flag.Int("port", 8000, "Port to open server on")
	flag.Parse()

	//

	dataRoot, _ = filepath.Abs(*flagRoot)
	DieOnError(Assert(DoesFileExist(dataRoot), "Please pass a valid directory as a --root parameter!"))
	Log("Saving to", dataRoot)

	//

	p := strconv.Itoa(*flagPort)
	Log("Initialization complete. Starting server on port " + p)
	http.ListenAndServe(":"+p, nil)
}
