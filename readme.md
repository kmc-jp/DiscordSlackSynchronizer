# DiscordSlackSynchronizer
## 概要
　Slack → Discord，Discord → Slack の同期を行う。

## 使用方法
### settings.jsonの追加

バイナリと同階層に次を配置。ただし、後述の環境変数を指定することで設定ファイルを他の場所に配置することも可能。

```json
[
    {
        "discord_server": "DISCORD_SERVER_ID",
        "channel": [
            {
                "slack": "SLACK_CHANNEL_ID",
                "discord": "DISCORD_CHANNEL_ID",
                "hook": "https://discordapp.com/api/webhooks/DISCORD_CHANNEL_HOOK_URL",
                "setting": {
                    "slack2discord": true,
                    "discord2slack": true,
                    "ShowChannelName": false
                }
            },
            {
                "slack": "SLACK_CHANNEL_ID",
                "discord": "all",
                "setting": {
                    "slack2discord": true,
                    "discord2slack": true,
                    "ShowChannelName": true,
                    "SendMuteState":false,
                    "SendVoiceState": true
                }
            }
        ]
    }
]
```

- `"discord":"all"`以下の設定は全てのdiscordチャンネルに反映される。
- その他の個別に指定したチャンネル設定はそちらが優先される。

### Discordへアプリ追加
追加時は、次のスコープが必要

```
ManageWebhook
ViewChannels
UseVoiceActivity
```
### Slackへアプリ追加
次のスコープが必要

```
channels:history
channels:read

chat:write
chat:write.customize
chat:write.public

files:read

groups:history

groups:read

users.profile:read
users:read
```

### チャンネルの追加

Slackの該当チャンネルルに該当Botを招待

### 環境変数

さらに、次の環境変数を追加する必要がある。

```
SLACK_API_TOKEN=xoxb-SlackToken
SLACK_EVENT_TOKEN=*****

DISCORD_BOT_TOKEN=Discord Bot Token

GYAZO_API_TOKEN=Gyazo API Token
```

## Discordの全チャンネルをSlackのそれぞれの同名のチャンネルに共有する
`CreateSlackChannelOnSend`を有効にすると、Discordの新規チャンネルにより、Slackのチャンネルも作られる。

all-allは現在1slack-discord関係にしか対応していません。
```
[
  {
    "discord_server": "768433410006974465",
    "slack_suffix": "-discord",
      {
        "slack": "all",
        "discord": "all",
        "setting": {
          "slack2discord": true,
          "discord2slack": true,
          "ShowChannelName": false,
          "SendVoiceState": false,
          "SendMuteState": false,
          "CreateSlackChannelOnSend": true
        }
      },
  }
]
```

## 参考
- WebhookURLs.jsonの内容はプログラム起動時にキャッシュされるので、設定変更した場合再起動が必要。
- 複数サーバ／複数チャンネルも対応。
- Discordに転送しない場合，`slackMap.json`の`"hook"`の記述は不要。

### WebConfigurator

追加で起動時に次の環境変数の指定が必要

```
SOCK_TYPE=tcp/unix/指定がなければ無効化
LISTEN_ADDRESS=Listen addr
```

デーモン化するときの都合などで
`settings.json`
を移動させなければいけない場合は、次の環境変数で、設定ファイルのディレクトリを指定することで設定が可能

```
STATE_DIRECTORY=/var/lib/...(例)
```

## Copyright

### noto-emoji

© 2021 The M+ FONTS Project Authors

Google Inc.
Arjen Nienhuis <a.g.nienhuis@gmail.com>
