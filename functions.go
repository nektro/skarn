package main

import (
	"database/sql"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gorilla/sessions"
	etc "github.com/nektro/go.etc"

	. "github.com/nektro/go-util/alias"
	. "github.com/nektro/go-util/util"
)

func isLoggedIn(r *http.Request) bool {
	return isLoggedInS(etc.GetSession(r))
}

func isLoggedInS(sess *sessions.Session) bool {
	_, ok := sess.Values["user"]
	return ok
}

func saveOAuth2Info(w http.ResponseWriter, r *http.Request, provider string, id string, name string, oa2resp map[string]interface{}) {
	sess := etc.GetSession(r)
	sess.Values["user"] = id
	sess.Save(r, w)
	queryUserBySnowflake(id)
	etc.Database.QueryDoUpdate("users", "username", name, "snowflake", id)
}

func queryUserBySnowflake(snowflake string) *User {
	rows := etc.Database.QueryDoSelect("users", "snowflake", snowflake)
	if rows.Next() {
		ru := scanUser(rows)
		rows.Close()
		return &ru
	}
	// else
	id := etc.Database.QueryNextID("users")
	etc.Database.QueryPrepared(true, F("insert into users values ('%d', '%s', '%s', 0, 0, '', '', '')", id, snowflake, T()))
	return queryUserBySnowflake(snowflake)
}

func scanUser(rows *sql.Rows) User {
	var u User
	rows.Scan(&u.ID, &u.Snowflake, &u.JoinedOn, &u.IsMember, &u.IsBanned, &u.IsAdmin, &u.Username, &u.Nickname, &u.Avatar)
	if len(u.Nickname) > 0 {
		u.RealName = u.Nickname
	} else {
		u.RealName = u.Username
	}
	return u
}

func pageInit(r *http.Request, w http.ResponseWriter, method string, requireLogin bool, requireMember bool, requireAdmin bool) (*sessions.Session, *User, error) {
	if r.Method != method {
		writeResponse(r, w, "Forbidden Method", F("%s is not allowed on this endpoint.", r.Method), "", "")
		return nil, nil, E("bad http method")
	}
	if method == http.MethodPost {
		r.ParseForm()
	}
	if !requireLogin {
		return nil, nil, nil
	}

	s := etc.GetSession(r)
	if !isLoggedInS(s) {
		writeResponse(r, w, "Authentication Required", "You must log in to access this site.", "/login", "Please Log In")
		return s, nil, E("not logged in")
	}

	if !requireMember {
		return s, nil, nil
	}
	u := queryUserBySnowflake(s.Values["user"].(string))
	if !u.IsMember {
		writeResponse(r, w, "Access Forbidden", "You must be a member to view this page.", "", "")
		return s, u, E("not a member")
	}
	if requireAdmin && !u.IsAdmin {
		writeResponse(r, w, "Access Forbidden", "You must be an admin to view this page.", "", "")
		return s, u, E("not an admin")
	}

	return s, u, nil
}

func doDiscordAPIRequest(endpoint string) ([]byte, int) {
	par := url.Values{}
	req, _ := http.NewRequest(http.MethodGet, "https://discordapp.com/api/v6"+endpoint, strings.NewReader(par.Encode()))
	req.Header.Set("User-Agent", "nektro/skarn")
	req.Header.Set("Authorization", "Bot "+config.BotToken)
	return doHttpRequest(req)
}

func doHttpRequest(req *http.Request) ([]byte, int) {
	resp, _ := http.DefaultClient.Do(req)
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	return body, resp.StatusCode
}

func containsAny(haystack []string, needle []string) bool {
	for _, item := range needle {
		if Contains(haystack, item) {
			return true
		}
	}
	return false
}

func writeResponse(r *http.Request, w http.ResponseWriter, title string, message string, url string, link string) {
	etc.WriteHandlebarsFile(r, w, "/hbs/response.hbs", map[string]interface{}{
		"base":    "/",
		"title":   title,
		"message": message,
		"url":     url,
		"link":    link,
	})
}

func assertPostFormValuesExist(r *http.Request, args ...string) error {
	for _, item := range args {
		v, o := r.PostForm[item]
		if !o {
			return E(F("form[%s] not sent", item))
		}
		if len(v) == 0 {
			return E(F("form[%s] empty", item))
		}
	}
	return nil
}

func assertURLValidity(toTest string) error {
	_, err := url.Parse(toTest)
	return err
}

func scanRowsRequests(rows *sql.Rows) []Request {
	result := []Request{}
	for rows.Next() {
		var rq Request
		var qual string
		rows.Scan(&rq.ID, &rq.Owner, &rq.Category, &rq.AddedOn, &rq.Title, &qual, &rq.Link, &rq.Description, &rq.Points, &rq.Filler, &rq.FilledOn, &rq.Response)
		rq.Quality = strings.Split(qual, ",")
		rq.Filled = rq.Filler > -1
		result = append(result, rq)
	}
	rows.Close()
	return result
}

func scanRowsUsers(rows *sql.Rows) []User {
	result := []User{}
	for rows.Next() {
		u := scanUser(rows)
		result = append(result, u)
	}
	rows.Close()
	return result
}

func scanInt(row *sql.Rows) int {
	var s int
	if row.Next() {
		row.Scan(&s)
	}
	row.Close()
	return s
}

func scanRowsUsersComplete(rows *sql.Rows) []UserComplete {
	result := []UserComplete{}
	users := scanRowsUsers(rows)
	for _, u := range users {
		uid := strconv.FormatInt(int64(u.ID), 10)
		var uc UserComplete
		uc.U = u
		uc.Fills = scanInt(etc.Database.QuerySelectFunc("requests", "count", "points", "filler", uid))
		uc.Points = scanInt(etc.Database.QuerySelectFunc("requests", "sum", "points", "filler", uid))
		result = append(result, uc)
	}
	return result
}

func writePage(r *http.Request, w http.ResponseWriter, user *User, path string, page string, title string, data map[string]interface{}) {
	etc.WriteHandlebarsFile(r, w, "/hbs/_header.hbs", map[string]interface{}{
		"base":  "/",
		"user":  user,
		"page":  page,
		"title": title,
	})
	etc.WriteHandlebarsFile(r, w, path, map[string]interface{}{
		"base":  "/",
		"user":  user,
		"page":  page,
		"title": title,
		"data":  data,
	})
}

func isInt(s string) bool {
	_, err := strconv.ParseInt(s, 10, 32)
	return err == nil
}
