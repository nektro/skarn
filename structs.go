package main

import (
	oauth2 "github.com/nektro/go.oauth2"
)

type Config struct {
	Port        int               `json:"port"`
	Clients     []oauth2.AppConf  `json:"clients"`
	Providers   []oauth2.Provider `json:"providers"`
	Themes      []string          `json:"themes"`
	Members     []string          `json:"members"`
	Admins      []string          `json:"admins"`
	AnnounceURL string            `json:"announce_webhook_url"`
}

type User struct {
	ID        int    `json:"id"`
	Snowflake string `json:"snowflake" sqlite:"text"`
	JoinedOn  string `json:"joined_on" sqlite:"text"`
	IsMember  bool   `json:"is_member" sqlite:"tinyint(1)"`
	IsAdmin   bool   `json:"is_admin" sqlite:"tinyint(1)"`
	Username  string `json:"username" sqlite:"text"`
	Nickname  string `json:"nickname" sqlite:"text"`
	Avatar    string `json:"avatar" sqlite:"text"`
	RealName  string `json:"name"`
}

type UserComplete struct {
	U        User `json:"user"`
	Fills    int  `json:"fills"`
	PointsF  int  `json:"points"`
	Requests int  `json:"requests"`
	PointsR  int  `json:"points_r"`
}

type Request struct {
	ID          int      `json:"id"`
	Owner       int      `json:"owner" sqlite:"int"`
	Category    string   `json:"category" sqlite:"text"`
	AddedOn     string   `json:"added_on" sqlite:"text"`
	Title       string   `json:"title" sqlite:"text"`
	QualityRaw  string   `json:"quality" sqlite:"text"`
	Quality     []string `json:"quality_real"`
	Link        string   `json:"link" sqlite:"text"`
	Description string   `json:"description" sqlite:"text"`
	Points      int      `json:"points" sqlite:"int"`
	Filled      bool     `json:"filled"`
	Filler      int      `json:"filler" sqlite:"int"`
	FilledOn    string   `json:"filled_on" sqlite:"text"`
	Response    string   `json:"response" sqlite:"text"`
}

type CategoryMapValue struct {
	Name    string   `json:"name"`
	Icon    string   `json:"icon"`
	Quality []string `json:"quality"`
}
