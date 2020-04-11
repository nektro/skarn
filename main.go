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
	"github.com/nektro/go-util/arrays/stringsu"
	"github.com/nektro/go-util/util"
	discord "github.com/nektro/go.discord"
	etc "github.com/nektro/go.etc"
	oauth2 "github.com/nektro/go.oauth2"
	"github.com/valyala/fastjson"

	. "github.com/nektro/go-util/alias"

	_ "github.com/nektro/skarn/statik"
)

var (
	config         = new(Config)
	categoryNames  = []string{"lit", "mov", "mus", "exe", "xxx", "etc"}
	categoryValues map[string]CategoryMapValue
	Version        = "vMASTER"
)

// file:///home/meghan/.config/skarn/config.json

func main() {
	etc.AppID = "skarn"
	Version = etc.FixBareVersion(Version)
	util.Log("Initializing Skarn Request System...")

	etc.PreInit()

	etc.Init("skarn", &config, "./verify", saveOAuth2Info)

	//

	catf, err := etc.MFS.Open("/categories.json")
	util.DieOnError(err, "Unable to read 'categories.json' from static resources!")
	catb, _ := ioutil.ReadAll(catf)
	json.Unmarshal(catb, &categoryValues)

	//

	etc.Database.CreateTableStruct("users", User{})
	etc.Database.CreateTableStruct("requests", Request{})

	//

	util.RunOnClose(func() {
		util.Log("Gracefully shutting down...")

		util.Log("Saving database to disk")
		etc.Database.Close()

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

	util.DieOnError(util.Assert(len(config.Clients) == 1, "'config.json' must only have 1 client"))
	util.DieOnError(util.Assert(config.Clients[0].For == "discord", "client in 'config.json' must be for discord"))

	etc.Router.HandleFunc("/login", oauth2.HandleOAuthLogin(isLoggedIn, "./verify", oauth2.ProviderIDMap["discord"], config.Clients[0].ID))
	etc.Router.HandleFunc("/callback", oauth2.HandleOAuthCallback(oauth2.ProviderIDMap["discord"], config.Clients[0].ID, config.Clients[0].Secret, saveOAuth2Info, "./verify"))

	etc.Router.HandleFunc("/verify", func(w http.ResponseWriter, r *http.Request) {
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
				w.Header().Add("location", "./requests?status=open")
				w.WriteHeader(http.StatusFound)
				return
			}
		}

		snowflake := s.Values["user"].(string)
		res, rcd := doDiscordAPIRequest(F("/guilds/%s/members/%s", config.Clients[0].Extra2, snowflake))
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

		w.Header().Add("location", "./requests?status=open")
		w.WriteHeader(http.StatusFound)
	})

	etc.Router.HandleFunc("/requests", func(w http.ResponseWriter, r *http.Request) {
		_, u, err := pageInit(r, w, http.MethodGet, true, true, false)
		if err != nil {
			return
		}
		q := etc.Database.Build().Se("*").Fr("requests")
		//
		switch r.URL.Query().Get("status") {
		case "open":
			q.Wh("filler", "-1")
		case "closed":
			q.Wr("filler", ">", "0")
		}
		//
		own := r.URL.Query().Get("owner")
		if own != "" {
			_, err := strconv.Atoi(own)
			if err == nil {
				q.Wh("owner", own)
			}
		}
		//
		fill := r.URL.Query().Get("filler")
		if fill != "" {
			_, err := strconv.Atoi(fill)
			if err == nil {
				q.Wh("filler", fill)
			}
		}
		//
		s := q.Exe()
		writePage(r, w, u, "/requests.hbs", "reqs", "Requests", map[string]interface{}{
			"requests": scanRowsRequests(s),
		})
	})

	etc.Router.HandleFunc("/new", func(w http.ResponseWriter, r *http.Request) {
		_, u, err := pageInit(r, w, http.MethodGet, true, true, false)
		if err != nil {
			return
		}
		writePage(r, w, u, "/new.hbs", "new", "New Request", map[string]interface{}{
			"categories": categoryValues,
		})
	})

	etc.Router.HandleFunc("/edit", func(w http.ResponseWriter, r *http.Request) {
		_, u, err := pageInit(r, w, http.MethodGet, true, true, false)
		if err != nil {
			return
		}
		id := r.URL.Query().Get("id")
		req, _, err := queryRequestById(id)
		if err != nil {
			writeResponse(r, w, "Unable to find request", "", "./../../requests?status=open", "Go back to /requests")
			return
		}
		if req.Owner != u.ID {
			writeResponse(r, w, "Must own request to edit", "", "./../../requests?status=open", "Go back to /requests")
			return
		}
		writePage(r, w, u, "/new.hbs", "edit", "Edit Request", map[string]interface{}{
			"categories": categoryValues,
			"req": map[string]string{
				"id":          id,
				"category":    req.Category,
				"title":       req.Title,
				"quality":     strings.Join(req.Quality, ","),
				"link":        req.Link,
				"description": req.Description,
			},
		})
	})

	etc.Router.HandleFunc("/leaderboard", func(w http.ResponseWriter, r *http.Request) {
		_, u, err := pageInit(r, w, http.MethodGet, true, true, false)
		if err != nil {
			return
		}
		writePage(r, w, u, "/leaderboard.hbs", "users", "Leaderboard", map[string]interface{}{
			"users": scanRowsUsersComplete(QueryDoSelect("users", "is_member", "1")),
		})
	})

	etc.Router.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		_, u, err := pageInit(r, w, http.MethodGet, true, true, false)
		if err != nil {
			return
		}
		writePage(r, w, u, "/stats.hbs", "stats", "Statistics", map[string]interface{}{
			//
		})
	})

	etc.Router.HandleFunc("/admin/users", func(w http.ResponseWriter, r *http.Request) {
		_, u, err := pageInit(r, w, http.MethodGet, true, true, true)
		if err != nil {
			return
		}
		writePage(r, w, u, "/all_users.hbs", "a/u", "All Users", map[string]interface{}{
			"users": scanRowsUsers(QueryDoSelectAll("users")),
		})
	})

	//

	etc.Router.HandleFunc("/api/request/create", func(w http.ResponseWriter, r *http.Request) {
		_, u, err := pageInit(r, w, http.MethodPost, true, true, false)
		if err != nil {
			return
		}
		if assertPostFormValuesExist(r, "category") != nil {
			writeResponse(r, w, "Missing POST values", "", "./../../new", "Go back to /new")
			return
		}
		cat := r.PostForm["category"][0]
		if !stringsu.Contains(categoryNames, cat) {
			writeResponse(r, w, "Invalid Category", "", "./../../new", "Go back to /new")
			return
		}
		if assertPostFormValuesExist(r, "quality_"+cat, "title", "link") != nil {
			writeResponse(r, w, "Missing POST values", "", "./../../new", "Go back to /new")
			return
		}
		q := r.PostForm.Get("quality_" + cat)
		t := r.PostForm.Get("title")
		l := r.PostForm.Get("link")
		d := r.PostForm.Get("description")
		lerr := assertURLValidity(l)
		if lerr != nil {
			writeResponse(r, w, "Link is not a valid URL", "", "./../../new", "Go back to /new")
			return
		}
		i := etc.Database.QueryNextID("requests")
		o := u.ID
		t = strings.ReplaceAll(t, "@", "@\u200D")
		t = strings.ReplaceAll(t, ":", ":\u200D")

		// success
		etc.Database.Build().Ins("requests", i, o, cat, T(), t, q, l, d, 1, -1, "", "").Exe()
		makeAnnouncement(F("**[NEW]** <@%s> created a request for **%s**.", u.Snowflake, t))
		writeResponse(r, w, "Success!", F("Added your request for %s", t), "./../../requests", "Back to home")
	})

	etc.Router.HandleFunc("/api/request/update", func(w http.ResponseWriter, r *http.Request) {
		_, u, err := pageInit(r, w, http.MethodPost, true, true, false)
		if err != nil {
			return
		}
		if assertPostFormValuesExist(r, "id", "category") != nil {
			writeResponse(r, w, "Missing POST values", "", "./../../new", "Go back to /new")
			return
		}
		id := r.PostForm.Get("id")
		req, _, err := queryRequestById(id)
		if err != nil {
			writeResponse(r, w, "Request not found", "", "./../../new", "Go back to /new")
			return
		}
		if req.Owner != u.ID {
			writeResponse(r, w, "Must own request to edit", "", "./../../edit?id="+id, "Go back to /edit")
			return
		}
		cat := r.PostForm.Get("category")
		if !stringsu.Contains(categoryNames, cat) {
			writeResponse(r, w, "Invalid Category", "", "./../../edit?id="+id, "Go back to /edit")
			return
		}
		if assertPostFormValuesExist(r, "quality_"+cat, "title", "link", "description") != nil {
			writeResponse(r, w, "Missing POST Values", "", "./../../edit?id="+id, "Go back to /edit")
			return
		}
		q := r.PostForm.Get("quality_" + cat)
		t := r.PostForm.Get("title")
		l := r.PostForm.Get("link")
		d := r.PostForm.Get("description")
		//
		t = strings.ReplaceAll(t, "@", "@\u200D")
		t = strings.ReplaceAll(t, ":", ":\u200D")
		if assertURLValidity(l) != nil {
			writeResponse(r, w, "Link is not a valid URL", "", "./../../edit?id="+id, "Go back to /edit")
			return
		}
		// success
		etc.Database.Build().Up("requests", "title", t).Wh("id", id).Exe()
		etc.Database.Build().Up("requests", "link", l).Wh("id", id).Exe()
		etc.Database.Build().Up("requests", "quality", q).Wh("id", id).Exe()
		etc.Database.Build().Up("requests", "description", d).Wh("id", id).Exe()
		makeAnnouncement(F("**[UPDATE]** <@%s> updated their request for **%s**.", u.Snowflake, t))
		writeResponse(r, w, "Success!", F("Updated your request for %s", t), "./../../requests?status=open", "Back to home")
	})

	etc.Router.HandleFunc("/api/request/fill", func(w http.ResponseWriter, r *http.Request) {
		_, u, err := pageInit(r, w, http.MethodPost, true, true, false)
		if err != nil {
			return
		}
		if assertPostFormValuesExist(r, "id", "message") != nil {
			writeResponse(r, w, "Missing POST values", "", "./../../requests", "Go back to /requests")
			return
		}
		rid := r.PostForm["id"][0]
		msg := r.PostForm["message"][0]
		//
		req, own, err := queryRequestById(rid)
		if err != nil {
			writeResponse(r, w, "Unable to find request", "", "./../../requests", "Go back to /requests")
			return
		}
		if req.Filled {
			writeResponse(r, w, "Cannot fill already filled request", "", "./../../requests", "Go back to /requests")
			return
		}
		//
		QueryDoUpdate("requests", "filler", strconv.Itoa(u.ID), "id", rid)
		QueryDoUpdate("requests", "filled_on", T(), "id", rid)
		QueryDoUpdate("requests", "response", msg, "id", rid)
		makeAnnouncement(F("**[FILL]** <@%s>'s request for **%s** was just filled by <@%s>.", own.Snowflake, req.Title, u.Snowflake))
		fmt.Fprintln(w, "good")
	})

	etc.Router.HandleFunc("/api/request/unfill", func(w http.ResponseWriter, r *http.Request) {
		_, u, err := pageInit(r, w, http.MethodPost, true, true, false)
		if err != nil {
			return
		}
		if assertPostFormValuesExist(r, "id") != nil {
			writeResponse(r, w, "Missing POST values", "", "./../../requests", "Go back to /requests")
			return
		}
		rid := r.PostForm["id"][0]
		req, own, err := queryRequestById(rid)
		if err != nil {
			writeResponse(r, w, "Unable to find request", "", "./../../requests", "Go back to /requests")
			return
		}
		if u.ID != own.ID && !u.IsAdmin {
			writeResponse(r, w, "Must own request to unfill", "", "./../../requests", "Go back to /requests")
			return
		}
		//
		QueryDoUpdate("requests", "filler", "-1", "id", rid)
		QueryDoUpdate("requests", "filled_on", "", "id", rid)
		QueryDoUpdate("requests", "response", "", "id", rid)
		makeAnnouncement(F("**[UNFILL]** <@%s>'s just un-filled their request for **%s**.", own.Snowflake, req.Title))
		fmt.Fprintln(w, "good")
	})

	etc.Router.HandleFunc("/api/request/delete", func(w http.ResponseWriter, r *http.Request) {
		_, u, err := pageInit(r, w, http.MethodPost, true, true, false)
		if err != nil {
			return
		}
		if assertPostFormValuesExist(r, "id") != nil {
			writeResponse(r, w, "Missing POST values", "", "./../../requests", "Go back to /requests")
			return
		}
		rid := r.PostForm["id"][0]
		req, own, err := queryRequestById(rid)
		if err != nil {
			writeResponse(r, w, "Unable to find request", "", "./../../requests", "Go back to /requests")
			return
		}
		if u.ID != own.ID && !u.IsAdmin {
			writeResponse(r, w, "Must own request to delete", "", "./../../requests", "Go back to /requests")
			return
		}
		//
		QueryDelete("requests", "id", rid)
		makeAnnouncement(F("**[DELETE]** <@%s>'s request for **%s** was just deleted.", own.Snowflake, req.Title))
		fmt.Fprintln(w, "good")
	})

	etc.Router.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		_, _, err := pageInit(r, w, http.MethodGet, true, true, false)
		if err != nil {
			return
		}
		bys, _ := json.Marshal(map[string]interface{}{
			"requests_over_time": requestsOverTime(),
		})
		w.Header().Add("content-type", "application/json")
		fmt.Fprintln(w, string(bys))
	})

	//

	etc.StartServer(config.Port)
}
