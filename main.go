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
	"time"

	"github.com/aymerick/raymond"
	"github.com/nektro/go-util/sqlite"
	"github.com/nektro/go.etc"
	"github.com/nektro/go.oauth2"
	"github.com/valyala/fastjson"

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

	handleLogin := oauth2.HandleOAuthLogin(isLoggedIn, "./verify", oauth2.ProviderDiscord, config.ID)
	handleCallback := oauth2.HandleOAuthCallback(oauth2.ProviderDiscord, config.ID, config.Secret, saveOAuth2Info, "./verify")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s, _, err := pageInit(r, w, http.MethodGet, false, false, false)
		if err != nil {
			return
		}
		if r.URL.Path != "/" {
			http.FileServer(http.Dir("www")).ServeHTTP(w, r)
			return
		}
		if _, ok := s.Values["user"]; ok {
			w.Header().Add("Location", "./requests")
		} else {
			w.Header().Add("location", "./login")
		}
		w.WriteHeader(http.StatusMovedPermanently)
	})

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if _, _, err := pageInit(r, w, http.MethodGet, false, false, false); err != nil {
			return
		}
		handleLogin(w, r)
	})

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if _, _, err := pageInit(r, w, http.MethodGet, false, false, false); err != nil {
			return
		}
		handleCallback(w, r)
	})

	http.HandleFunc("/verify", func(w http.ResponseWriter, r *http.Request) {
		s, u, err := pageInit(r, w, http.MethodGet, true, false, false)
		if err != nil {
			return
		}

		tm, ok := s.Values["verify_time"]
		if ok {
			a := time.Now().Unix() - tm.(int64)
			b := int64(time.Second * 60 * 5)
			if a < b {
				if !u.IsMember {
					writeResponse(r, w, "Access Denied", "Must be a member. Please try again later.", "", "")
					return // only query once every 5 mins
				}
				w.Header().Add("location", "./requests")
				w.WriteHeader(http.StatusMovedPermanently)
				return
			}
		}

		snowflake := s.Values["user"].(string)
		res, rcd := doDiscordAPIRequest(F("/guilds/%s/members/%s", config.Server, snowflake))
		if rcd >= 400 {
			writeResponse(r, w, "Discord Error", fastjson.GetString(res, "message"), "", "")
			return // discord error
		}

		var dat *DiscordMe
		json.Unmarshal(res, &dat)

		database.QueryDoUpdate("users", "nickname", dat.Nick, "snowflake", snowflake)
		database.QueryDoUpdate("users", "avatar", dat.User.Avatar, "snowflake", snowflake)

		allowed := false
		if containsAny(dat.Roles, config.Members) {
			database.QueryDoUpdate("users", "is_member", "1", "snowflake", snowflake)
			allowed = true
		}
		if containsAny(dat.Roles, config.Admins) {
			database.QueryDoUpdate("users", "is_admin", "1", "snowflake", snowflake)
			allowed = true
		}
		if !allowed {
			database.QueryDoUpdate("users", "is_member", "0", "snowflake", snowflake)
			database.QueryDoUpdate("users", "is_admin", "0", "snowflake", snowflake)
			writeResponse(r, w, "Acess Denied", "No valid Discord Roles found.", "", "")
			return
		}

		s.Values["verify_time"] = time.Now().Unix()
		s.Save(r, w)

		w.Header().Add("location", "./requests")
		w.WriteHeader(http.StatusMovedPermanently)
	})

	p := strconv.Itoa(*flagPort)
	Log("Initialization complete. Starting server on port " + p)
	http.ListenAndServe(":"+p, nil)
}
