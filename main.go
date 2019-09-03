package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/aymerick/raymond"
	"github.com/nektro/go-util/sqlite"
	"github.com/nektro/go.etc"
	"github.com/nektro/go.oauth2"
	"github.com/nektro/go-util/util"
	"github.com/valyala/fastjson"

	flag "github.com/spf13/pflag"

	. "github.com/nektro/go-util/alias"
)

var (
	dataRoot       string
	config         *Config
	categoryNames  = []string{"lit", "mov", "mus", "exe", "xxx", "etc"}
	categoryValues map[string]CategoryMapValue
	database       *sqlite.DB
)

func main() {
	flagRoot := flag.String("root", "", "Path of root directory for files")
	flag.Parse()
	util.Log("Initializing Skarn Request System...")

	//

	dataRoot, _ = filepath.Abs(*flagRoot)
	DieOnError(Assert(DoesFileExist(dataRoot), "Please pass a valid directory as a --root parameter!"))
	Log("Saving to", dataRoot)

	//

	etc.InitConfig(dataRoot+"/config.json", &config)
	etc.ConfigAssertKeysNonEmpty(&config, "ID", "Secret", "BotToken", "Server")

	etc.SetSessionName("session_skarn")

	json.Unmarshal(ReadFile("./data/categories.json"), &categoryValues)

	//

	database = sqlite.Connect(dataRoot)
	CheckErr(database.Ping())

	database.CreateTableStruct("users", User{})
	database.CreateTableStruct("requests", Request{})

	etc.RunOnClose(func() {
		util.Log("Gracefully shutting down...")

		database.Close()
		util.Log("Saved database to disk")

		os.Exit(0)
	})

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


	http.HandleFunc("/", http.FileServer(etc.MFS).ServeHTTP)
	http.HandleFunc("/login", oauth2.HandleOAuthLogin(isLoggedIn, "./verify", oauth2.ProviderDiscord, config.ID))
	http.HandleFunc("/callback", oauth2.HandleOAuthCallback(oauth2.ProviderDiscord, config.ID, config.Secret, saveOAuth2Info, "./verify"))

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

	http.HandleFunc("/requests", func(w http.ResponseWriter, r *http.Request) {
		_, u, err := pageInit(r, w, http.MethodGet, true, true, false)
		if err != nil {
			return
		}
		writePage(r, w, u, "./hbs/requests.hbs", "open", "Open Requests", map[string]interface{}{
			"tagline":  "All of the requests that are currently unfilled can be found from here.",
			"requests": scanRowsRequests(database.Select().All().From("requests").WhereEq("filler", "-1").Run(false)),
		})
	})

	http.HandleFunc("/new", func(w http.ResponseWriter, r *http.Request) {
		_, u, err := pageInit(r, w, http.MethodGet, true, true, false)
		if err != nil {
			return
		}
		writePage(r, w, u, "./hbs/new.hbs", "new", "New Request", map[string]interface{}{
			"categories": categoryValues,
		})
	})

	http.HandleFunc("/mine", func(w http.ResponseWriter, r *http.Request) {
		_, u, err := pageInit(r, w, http.MethodGet, true, true, false)
		if err != nil {
			return
		}
		id := strconv.FormatInt(int64(u.ID), 10)
		writePage(r, w, u, "./hbs/requests.hbs", "mine", "My Requests", map[string]interface{}{
			"tagline":  "All requests filed by you are here.",
			"requests": scanRowsRequests(database.Select().All().From("requests").WhereEq("owner", id).Run(false)),
		})
	})

	http.HandleFunc("/leaderboard", func(w http.ResponseWriter, r *http.Request) {
		_, u, err := pageInit(r, w, http.MethodGet, true, true, false)
		if err != nil {
			return
		}
		writePage(r, w, u, "./hbs/leaderboard.hbs", "users", "Leaderboard", map[string]interface{}{
			"users": scanRowsUsersComplete(database.Select().All().From("users").WhereEq("is_member", "1").Run(false)),
		})
	})

	http.HandleFunc("/admin/users", func(w http.ResponseWriter, r *http.Request) {
		_, u, err := pageInit(r, w, http.MethodGet, true, true, true)
		if err != nil {
			return
		}
		writePage(r, w, u, "./hbs/all_users.hbs", "a/u", "All Users", map[string]interface{}{
			"users": scanRowsUsers(database.Select().All().From("users").Run(false)),
		})
	})

	http.HandleFunc("/admin/requests", func(w http.ResponseWriter, r *http.Request) {
		_, u, err := pageInit(r, w, http.MethodGet, true, true, true)
		if err != nil {
			return
		}
		writePage(r, w, u, "./hbs/all_requests.hbs", "a/r", "All Requests", map[string]interface{}{
			"requests": scanRowsRequests(database.Select().All().From("requests").Run(false)),
		})
	})

	//

	http.HandleFunc("/api/request/create", func(w http.ResponseWriter, r *http.Request) {
		_, u, err := pageInit(r, w, http.MethodPost, true, true, false)
		if err != nil {
			return
		}
		if assertPostFormValuesExist(r, "category") != nil {
			writeResponse(r, w, "Missing POST Value", "", "./../../new", "Go back to /new")
			return
		}
		cat := r.PostForm["category"][0]
		if !util.Contains(categoryNames, cat) {
			writeResponse(r, w, "Invalid Category", "", "./../../new", "Go back to /new")
			return
		}
		if assertPostFormValuesExist(r, "quality_"+cat, "title", "link", "description") != nil {
			writeResponse(r, w, "Missing POST Values", "Request description items are required.", "./../../new", "Go back to /new")
			return // post value not found
		}
		q := r.PostForm["quality_"+cat][0]
		t := r.PostForm["title"][0]
		l := r.PostForm["link"][0]
		d := r.PostForm["description"][0]
		lerr := assertURLValidity(l)
		if lerr != nil {
			fmt.Fprintln(w, "E", "link", lerr.Error())
			return // link is not a url
		}
		i := database.QueryNextID("requests")
		o := u.ID

		// success
		database.QueryPrepared(true, F("insert into requests values (%d, %d, ?, '%s', ?, ?, ?, ?, 1, -1, '', '')", i, o, T()), cat, t, q, l, d)
		fmt.Println("R", "A", i, o, t)
		writeResponse(r, w, "Success!", F("Added your request for %s", t), "./../../requests", "Back to home")
	})

	http.HandleFunc("/api/request/update_score", func(w http.ResponseWriter, r *http.Request) {
		_, _, err := pageInit(r, w, http.MethodPost, true, true, true)
		if err != nil {
			return
		}
		if assertPostFormValuesExist(r, "id", "score") != nil {
			fmt.Fprintln(w, "missing post value")
			return
		}
		rid := r.PostForm["id"][0]
		scr := r.PostForm["score"][0]
		//
		if !isInt(rid) || !isInt(scr) {
			fmt.Fprintln(w, "invalid value")
			return
		}
		//
		database.QueryDoUpdate("requests", "points", scr, "id", rid)
		fmt.Fprintln(w, "good")
	})

	http.HandleFunc("/api/request/fill", func(w http.ResponseWriter, r *http.Request) {
		_, u, err := pageInit(r, w, http.MethodPost, true, true, false)
		if err != nil {
			return
		}
		if assertPostFormValuesExist(r, "id", "message") != nil {
			fmt.Fprintln(w, "missing post value")
			return
		}
		rid := r.PostForm["id"][0]
		msg := r.PostForm["message"][0]
		//
		if !isInt(rid) {
			fmt.Fprintln(w, "invalid request id")
			return
		}
		uid := strconv.FormatInt(int64(u.ID), 10)
		//
		database.QueryDoUpdate("requests", "filler", uid, "id", rid)
		database.QueryDoUpdate("requests", "filled_on", T(), "id", rid)
		database.QueryDoUpdate("requests", "response", msg, "id", rid)
		fmt.Fprintln(w, "good")
	})

	//

	p := strconv.Itoa(config.Port)
	util.Log("Initialization complete. Starting server on port " + p)
	http.ListenAndServe(":"+p, nil)
}
