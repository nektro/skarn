package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aymerick/raymond"
	"github.com/nektro/go-util/util"
	discord "github.com/nektro/go.discord"
	etc "github.com/nektro/go.etc"
	oauth2 "github.com/nektro/go.oauth2"
	"github.com/valyala/fastjson"

	. "github.com/nektro/go-util/alias"

	_ "github.com/nektro/skarn/statik"
)

var (
	config         *Config
	categoryNames  = []string{"lit", "mov", "mus", "exe", "xxx", "etc"}
	categoryValues map[string]CategoryMapValue
)

func main() {
	util.Log("Initializing Skarn Request System...")

	//

	etc.Init("skarn", &config, "./verify", saveOAuth2Info)
	etc.ConfigAssertKeysNonEmpty(&config, "ID", "Secret", "BotToken", "Server")

	catf, _ := etc.MFS.Open("/categories.json")
	catb, _ := ioutil.ReadAll(catf)
	json.Unmarshal(catb, &categoryValues)

	etc.Database.CreateTableStruct("users", User{})
	etc.Database.CreateTableStruct("requests", Request{})

	util.RunOnClose(func() {
		util.Log("Gracefully shutting down...")

		etc.Database.Close()
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
		usrs := scanRowsUsers(QueryDoSelect("users", "id", strconv.FormatInt(int64(userID), 10)))
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
			b := int64(time.Second * 60 * 1)
			if a < b {
				if !u.IsMember {
					writeResponse(r, w, "Access Denied", "Must be a member. Please try again later.", "", "")
					return // only query once every 1 mins
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

		var dat discord.GuildMember
		json.Unmarshal(res, &dat)

		QueryDoUpdate("users", "nickname", dat.Nickname, "snowflake", snowflake)
		QueryDoUpdate("users", "avatar", dat.User.Avatar, "snowflake", snowflake)

		allowed := false
		if containsAny(dat.Roles, config.Members) {
			QueryDoUpdate("users", "is_member", "1", "snowflake", snowflake)
			allowed = true
		}
		if containsAny(dat.Roles, config.Admins) {
			QueryDoUpdate("users", "is_admin", "1", "snowflake", snowflake)
			allowed = true
		}
		if !allowed {
			QueryDoUpdate("users", "is_member", "0", "snowflake", snowflake)
			QueryDoUpdate("users", "is_admin", "0", "snowflake", snowflake)
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
		writePage(r, w, u, "/requests.hbs", "open", "Open Requests", map[string]interface{}{
			"tagline":  "All of the requests that are currently unfilled can be found from here.",
			"requests": scanRowsRequests(QueryDoSelect("requests", "filler", "-1")),
		})
	})

	http.HandleFunc("/new", func(w http.ResponseWriter, r *http.Request) {
		_, u, err := pageInit(r, w, http.MethodGet, true, true, false)
		if err != nil {
			return
		}
		writePage(r, w, u, "/new.hbs", "new", "New Request", map[string]interface{}{
			"categories": categoryValues,
		})
	})

	http.HandleFunc("/mine", func(w http.ResponseWriter, r *http.Request) {
		_, u, err := pageInit(r, w, http.MethodGet, true, true, false)
		if err != nil {
			return
		}
		id := strconv.FormatInt(int64(u.ID), 10)
		writePage(r, w, u, "/requests.hbs", "mine", "My Requests", map[string]interface{}{
			"tagline":  "All requests filed by you are here.",
			"requests": scanRowsRequests(QueryDoSelect("requests", "owner", id)),
		})
	})

	http.HandleFunc("/leaderboard", func(w http.ResponseWriter, r *http.Request) {
		_, u, err := pageInit(r, w, http.MethodGet, true, true, false)
		if err != nil {
			return
		}
		writePage(r, w, u, "/leaderboard.hbs", "users", "Leaderboard", map[string]interface{}{
			"users": scanRowsUsersComplete(QueryDoSelect("users", "is_member", "1")),
		})
	})

	http.HandleFunc("/admin/users", func(w http.ResponseWriter, r *http.Request) {
		_, u, err := pageInit(r, w, http.MethodGet, true, true, true)
		if err != nil {
			return
		}
		writePage(r, w, u, "/all_users.hbs", "a/u", "All Users", map[string]interface{}{
			"users": scanRowsUsers(QueryDoSelectAll("users")),
		})
	})

	http.HandleFunc("/admin/requests", func(w http.ResponseWriter, r *http.Request) {
		_, u, err := pageInit(r, w, http.MethodGet, true, true, true)
		if err != nil {
			return
		}
		writePage(r, w, u, "/all_requests.hbs", "a/r", "All Requests", map[string]interface{}{
			"requests": scanRowsRequests(QueryDoSelectAll("requests")),
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
		i := etc.Database.QueryNextID("requests")
		o := u.ID
		t = strings.ReplaceAll(t, "@", "@\u200D")
		t = strings.ReplaceAll(t, ":", ":\u200D")

		// success
		etc.Database.QueryPrepared(true, F("insert into requests values (%d, %d, ?, '%s', ?, ?, ?, ?, 1, -1, '', '')", i, o, T()), cat, t, q, l, d)
		makeAnnouncement(F("**[NEW]** <@%s> created a request for **%s**.", u.Snowflake, t))
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
		QueryDoUpdate("requests", "points", scr, "id", rid)
		fmt.Fprintln(w, "good")
	})

	http.HandleFunc("/api/request/fill", func(w http.ResponseWriter, r *http.Request) {
		_, u, err := pageInit(r, w, http.MethodPost, true, true, false)
		if err != nil {
			return
		}
		if assertPostFormValuesExist(r, "id", "message") != nil {
			return
		}
		rid := r.PostForm["id"][0]
		msg := r.PostForm["message"][0]
		//
		req, own, err := queryRequestById(rid)
		if err != nil {
			return
		}
		if req.Filled {
			return
		}
		//
		QueryDoUpdate("requests", "filler", strconv.Itoa(u.ID), "id", rid)
		QueryDoUpdate("requests", "filled_on", T(), "id", rid)
		QueryDoUpdate("requests", "response", msg, "id", rid)
		makeAnnouncement(F("**[FILL]** <@%s>'s request for **%s** was just filled by <@%s>.", own.Snowflake, req.Title, u.Snowflake))
		fmt.Fprintln(w, "good")
	})

	http.HandleFunc("/api/request/unfill", func(w http.ResponseWriter, r *http.Request) {
		_, u, err := pageInit(r, w, http.MethodPost, true, true, false)
		if err != nil {
			return
		}
		if assertPostFormValuesExist(r, "id") != nil {
			return
		}
		rid := r.PostForm["id"][0]
		req, own, err := queryRequestById(rid)
		if err != nil {
			return
		}
		if u.ID != own.ID && !u.IsAdmin {
			return
		}
		//
		QueryDoUpdate("requests", "filler", "-1", "id", rid)
		QueryDoUpdate("requests", "filled_on", "", "id", rid)
		QueryDoUpdate("requests", "response", "", "id", rid)
		makeAnnouncement(F("**[UNFILL]** <@%s>'s just un-filled their request for **%s**.", own.Snowflake, req.Title))
		fmt.Fprintln(w, "good")
	})

	http.HandleFunc("/api/request/delete", func(w http.ResponseWriter, r *http.Request) {
		_, u, err := pageInit(r, w, http.MethodPost, true, true, false)
		if err != nil {
			return
		}
		if assertPostFormValuesExist(r, "id") != nil {
			fmt.Fprintln(w, "missing post value")
			return
		}
		rid := r.PostForm["id"][0]
		req, own, err := queryRequestById(rid)
		if err != nil {
			return
		}
		if u.ID != own.ID && !u.IsAdmin {
			return
		}
		//
		QueryDelete("requests", "id", rid)
		makeAnnouncement(F("**[DELETE]** <@%s>'s request for **%s** was just deleted.", own.Snowflake, req.Title))
		fmt.Fprintln(w, "good")
	})

	//

	p := strconv.Itoa(config.Port)
	util.Log("Initialization complete. Starting server on port " + p)
	http.ListenAndServe(":"+p, nil)
}
