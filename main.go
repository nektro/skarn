package main

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/nektro/go.etc"

	flag "github.com/spf13/pflag"

	. "github.com/nektro/go-util/alias"
	. "github.com/nektro/go-util/util"
)

var (
	dataRoot       string
	config         *Config
	categoryNames  = []string{"lit", "mov", "mus", "exe", "xxx", "etc"}
	categoryValues map[string]CategoryMapValue
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

	etc.InitConfig(dataRoot+"/config.json", &config)
	etc.ConfigAssertKeysNonEmpty(&config, "ID", "Secret", "BotToken", "Server")
	etc.ReadAllowedHostnames(dataRoot + "/allowed_domains.txt")
	etc.SetSessionName("session_skarn_test")

	json.Unmarshal(ReadFile("./data/categories.json"), &categoryValues)

	//

	p := strconv.Itoa(*flagPort)
	Log("Initialization complete. Starting server on port " + p)
	http.ListenAndServe(":"+p, nil)
}
