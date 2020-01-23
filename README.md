# Skarn
![loc](https://sloc.xyz/github/nektro/skarn)
[![license](https://img.shields.io/github/license/nektro/skarn.svg)](https://github.com/nektro/skarn/blob/master/LICENSE)
[![discord](https://img.shields.io/discord/551971034593755159.svg)](https://discord.gg/P6Y4zQC)
[![paypal](https://img.shields.io/badge/donate-paypal-009cdf)](https://paypal.me/nektro)
[![circleci](https://circleci.com/gh/nektro/skarn.svg?style=svg)](https://circleci.com/gh/nektro/skarn)
[![goreportcard](https://goreportcard.com/badge/github.com/nektro/skarn)](https://goreportcard.com/report/github.com/nektro/skarn)
[![codefactor](https://www.codefactor.io/repository/github/nektro/skarn/badge)](https://www.codefactor.io/repository/github/nektro/skarn)

Media Request & Inventory Management System

## Getting Started
These instructions will get you a copy of the project up and running on your machine.

### Configuration
This guide assumes you want to configure Skarn to a Discord server and auto-add members that have a specific role.

- Go to https://discordapp.com/developers/applications/
- Create an application and save down the Client ID and Client Secret.
- Add a bot to the application and save down the Bot Token.
- Add the bot to the server you wish to auth this instance throuh.
- Create a `~/.config/skarn/config.json`.

```json
{
	"port": 8000,
	"id": "{CLIENT_ID}",
	"secret": "{CLIENT_SECRET}",
	"bot_token": "{BOT_TOKEN}",
	"server": "{SERVER_ID}",
	"members": ["{ROLE_SNOWFLAKE"],
	"admins": ["{ROLE_SNOWFLAKE"],
	"themes": ["{THEME_ID}"],
	"announce_webhook_url": "{}"
}
```
- It should be in the above format.
    - `"members"` is an array of role snowflakes that will be added as a member. Member can create and fill requests.
    - `"admins"` is an array of role snowflakes that will be added as an admin. Admins can adjust the amount of points each request is worth.
- Lastly run the binary as follows obtained from either the *Deployment* or *Development* sections.

```bash
$ ./skarn
```

### Themes
Skarn supports custom themes through use of the `"themes"` property in your `config.json` to identify a folder or list of folders to overwrite any of the handlebars template files. The location to place themes is at `~/.config/skarn/themes/{THEME_ID}/`

### Announcements
Using the `"announce_webhook_url"` property you can create an announcements channel that will display status updates to requests. See https://support.discordapp.com/hc/en-us/articles/228383668-Intro-to-Webhooks for more info on how to setup Discord Webhooks and get the URL.

### Deployment
By signing in with GitHub, you can download pre-built binaries from the *Artifacts* tab of https://circleci.com/gh/nektro/skarn.

### Development
- The Go Language 1.7+ (https://golang.org/dl/)
- GCC on your PATH (for the https://github.com/mattn/go-sqlite3 installation)

```bash
$ go get -u -v github.com/nektro/skarn
$ cd $GOPATH/src/github.com/nektro/skarn/
$ ./scripts/build/all.sh
```

## Contributing
[![issues](https://img.shields.io/github/issues/nektro/skarn.svg)](https://github.com/nektro/skarn/issues)
[![pulls](https://img.shields.io/github/issues-pr/nektro/skarn.svg)](https://github.com/nektro/skarn/pulls)

We take issues all the time right here on GitHub. We use labels extensively to show the progress through the fixing process. Question issues are okay but make sure to close the issue when it's been answered!

When making a pull request, please have it be associated with an issue and make a comment on the issue saying that you're working on it so everyone else knows what's going on :D


## Contact
- hello@nektro.net
- Meghan#2032 on https://discord.gg/P6Y4zQC
- https://twitter.com/nektro

## License
Apache 2.0
