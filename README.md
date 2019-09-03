# Skarn
![loc](https://tokei.rs/b1/github/nektro/skarn)
[![license](https://img.shields.io/github/license/nektro/skarn.svg)](https://github.com/nektro/skarn/blob/master/LICENSE)
[![discord](https://img.shields.io/discord/551971034593755159.svg)](https://discord.gg/P6Y4zQC)

Media Request & Inventory Management System

[![buymeacoffee](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/nektro)

## Getting Started
These instructions will get you a copy of the project up and running on your machine.

### Prerequisites
- The Go Language 1.7+ (https://golang.org/dl/)
- GCC on your PATH (for the https://github.com/mattn/go-sqlite3 installation)

### Installing
This guide assumes you want to configure Skarn to a Discord server and auto-add members that have a specific role. If this is not the case, then please continue to [Options](#options).

- Go to https://discordapp.com/developers/applications/
- Create an application and save down the Client ID and Client Secret.
- Add a bot to the application and save down the Bot Token.
- Add the bot to the server you wish to auth this instance throuh.
- Create a directory that the server may use to store some files.
- In that directory create a `config.json`.

```json
{
	"id": "{CLIENT_ID}",
	"secret": "{CLIENT_SECRET}",
	"bot_token": "{BOT_TOKEN}",
	"server": "{SERVER_ID}",
	"members": ["{ROLE_SNOWFLAKE"],
	"admins": ["{ROLE_SNOWFLAKE"]
}
```
- It should be in the above format.
    - `"members"` is an array of role snowflakes that will be added as a member. Member can create and fill requests.
    - `"admins"` is an array of role snowflakes that will be added as an admin. Admins can adjust the amount of points each request is worth.
- Next create a file called `allowed_domains.txt` and in it place the hostnames that users may connect to the server with.
    - This does not include the port number (Even if it's not 80 or 443).
    - File must have LF line endings.
- Lastly run the following.

```bash
$ go get -u github.com/nektro/skarn
$ cd $GOPATH/src/github.com/nektro/skarn/
$ go build
```

### Running
```bash
$ ./skarn --root $homedir --port 80
```

#### Options
| Argument | Type | Default | Description |
|--------|------|---------|-------------|
| `--root` | Path. | None. | A path to the directory Skarn can store some configuration and database files. |
| `--port` | `int` | `8000` | Port to have the Web UI listen on. |
| `--allow-all-hosts` | `bool` | `false` | When true, `allowed_domains.txt` is not checked and Skarn will accept requests from all hostnames. |

## Contributing
We take issues all the time right here on GitHub. We use labels extensively to show the progress through the fixing process. Question issues are okay but make sure to close the issue when it's been answered!

[![issues](https://img.shields.io/github/issues/nektro/skarn.svg)](https://github.com/nektro/skarn/issues)

When making a pull request, please have it be associated with an issue and make a comment on the issue saying that you're working on it so everyone else knows what's going on :D

[![pulls](https://img.shields.io/github/issues-pr/nektro/skarn.svg)](https://github.com/nektro/skarn/pulls)

## Contact
- hello@nektro.net
- Meghan#2032 on https://discord.gg/P6Y4zQC
- https://twitter.com/nektro

## Donate
Really like this project and want to support me and my continued development in open source? I'm on buymeacoffee.com!

[![buymeacoffee](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/nektro)

## License
Apache 2.0
