### Slack kudos bot
#### How to use
Buiild the binary
```bash
go build -o kudosbot
```
Export the following environment variables
- sqlite filename
- slack bot token(oauth2) starts with `xoxb-...`
- slack app token starts with `xapp-...`
```bash
export KUDOS_SQLITE_FILENAME='kudos.db' 
export KUDOS_SLACK_BOT_TOKEN='xoxb-...'
export KUDOS_SLACK_APP_TOKEN='xapp-...'
```

Run the binary
```bash
./kudosbot
```
and add the bot to the channels you want it to be active in.