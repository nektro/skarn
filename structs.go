package main

type Config struct {
	Port     int      `json:"port"`
	ID       string   `json:"id"`
	Secret   string   `json:"secret"`
	BotToken string   `json:"bot_token"`
	Server   string   `json:"server"`
	Members  []string `json:"members"`
	Admins   []string `json:"admins"`
}

type DiscordMe struct {
	Nick string `json:"nick"`
	User struct {
		Username string `json:"username"`
		Avatar   string `json:"avatar"`
	} `json:"user"`
	Roles []string `json:"roles"`
}

type User struct {
	ID        int    `json:"id"`
	Snowflake string `json:"snowflake"`
	JoinedOn  string `json:"joined_on"`
	IsMember  bool   `json:"is_member"`
	IsBanned  bool   `json:"is_banned"`
	IsAdmin   bool   `json:"is_admin"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	Avatar    string `json:"avatar"`
	RealName  string `json:"name"`
}

type UserComplete struct {
	U      User `json:"user"`
	Fills  int  `json:"fills"`
	Points int  `json:"points"`
}

type Request struct {
	ID          int      `json:"id"`
	Owner       int      `json:"owner"`
	Category    string   `json:"category"`
	AddedOn     string   `json:"added_on"`
	Title       string   `json:"title"`
	Quality     []string `json:"quality"`
	Link        string   `json:"link"`
	Description string   `json:"description"`
	Points      int      `json:"points"`
	Filled      bool     `json:"filled"`
	Filler      int      `json:"filler"`
	FilledOn    string   `json:"filled_on"`
	Response    string   `json:"response"`
}

type CategoryMapValue struct {
	Name    string   `json:"name"`
	Icon    string   `json:"icon"`
	Quality []string `json:"quality"`
}
