package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/aymerick/raymond"
	"github.com/nektro/go-util/sqlite"
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
	database       *sqlite.DB
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

	database = sqlite.Connect(dataRoot)
	CheckErr(database.Ping())

	database.CreateTable("users", []string{"id", "int primary key"}, [][]string{
		{"snowflake", "text"},
		{"joined_on", "text"},
		{"is_member", "tinyint(1)"},
		{"is_banned", "tinyint(1)"},
		{"is_admin", "tinyint(1)"},
		{"name", "text"},
		{"nickname", "text"},
		{"avatar", "text"},
	})
	database.CreateTable("requests", []string{"id", "int primary key"}, [][]string{
		{"owner", "int"},
		{"category", "text"},
		{"added_on", "text"},
		{"title", "text"},
		{"quality", "text"},
		{"link", "text"},
		{"description", "text"},
		{"points", "int"},
		{"filler", "int"},
		{"filled_on", "text"},
		{"response", "text"},
	})

	//

	gracefulStop := make(chan os.Signal)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)

	go func() {
		sig := <-gracefulStop
		Log(F("Caught signal '%+v'", sig))
		Log("Gracefully shutting down...")

		database.Close()
		Log("Saved database to disk")

		os.Exit(0)
	}()

	//

	raymond.RegisterHelper("icon", func(cat string) string {
		return categoryValues[cat].Icon
	})
	raymond.RegisterHelper("domain", func(link string) string {
		u, e := url.Parse(link)
		if e != nil {
			return "WWW"
		}
		return u.Host
	})
	raymond.RegisterHelper("name", func(userID int) string {
		usrs := scanRowsUsers(database.QueryDoSelect("users", "id", strconv.FormatInt(int64(userID), 10)))
		if len(usrs) == 0 {
			return ""
		}
		return usrs[0].RealName
	})
	raymond.RegisterHelper("quality", func(cat string, item string) string {
		i, _ := strconv.ParseInt(item, 10, 32)
		return categoryValues[cat].Quality[i]
	})
	raymond.RegisterHelper("length", func(array []string) int {
		return len(array)
	})

	//

	p := strconv.Itoa(*flagPort)
	Log("Initialization complete. Starting server on port " + p)
	http.ListenAndServe(":"+p, nil)
}
