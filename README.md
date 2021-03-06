# Skarn
![loc](https://sloc.xyz/github/nektro/skarn)
[![license](https://img.shields.io/github/license/nektro/skarn.svg)](https://github.com/nektro/skarn/blob/master/LICENSE)
[![discord](https://img.shields.io/discord/551971034593755159.svg)](https://discord.gg/P6Y4zQC)
[![paypal](https://img.shields.io/badge/donate-paypal-009cdf)](https://paypal.me/nektro)
[![circleci](https://circleci.com/gh/nektro/skarn.svg?style=svg)](https://circleci.com/gh/nektro/skarn)
[![release](https://img.shields.io/github/v/release/nektro/skarn)](https://github.com/nektro/skarn/releases/latest)
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
- Below are the command line flags you may use to configure your Skarn instance.

| Name | Type | Default | Description |
|------|------|---------|-------------|
| `--port` | `int` | `8001` | Port for web server to bind to. |
| `--members` | `[]string` | none. | List of role snowflakes that may view this instance |
| `--admins` | `[]string` | none. | List of role snowflakes that may manage this instance |
| `--theme` | `[]string` | none. | List of theme IDs |
| `--announce-webhook-url` | `string` | none. | Discord webhook URL for announcements |

### Themes
Skarn supports custom themes through use of the `--theme` flag to identify a folder or list of folders to overwrite any of the handlebars template files. The location to place themes is at `~/.config/skarn/themes/{THEME_ID}/`

### Announcements
Using the `--announce-webhook-url` flag you can create an announcements channel that will display status updates to requests. See https://support.discordapp.com/hc/en-us/articles/228383668-Intro-to-Webhooks for more info on how to setup Discord Webhooks and get the URL.

## Development

### Prerequisites
- The Go Language 1.12+ (https://golang.org/dl/)
- Docker (https://www.docker.com/products/docker-desktop)
- Docker Compose (https://docs.docker.com/compose/install/)

### Installing
Run
```
$ git clone https://github.com/nektro/skarn
$ cd ./skarn/
$ go get -v .
$ docker-compose up
```

## Deployment
Pre-compiled binaries can be obtained from https://github.com/nektro/skarn/releases/latest.

Or you can build from source:
```
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
