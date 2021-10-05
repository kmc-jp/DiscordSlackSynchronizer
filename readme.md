# DiscordSlackSynchronizer
## 概要
　Slack → Discord，Discord → Slack の同期を行う。

## 使用方法
1. settingsディレクトリを作成し、中に
   `settings.json`
   を作成
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

2. Discordに[アプリを追加](https://discordapp.com/api/oauth2/authorize?client_id=705678824087617578&permissions=33561600&scope=bot)
   
- Discordに転送する場合

1. Slackの該当チャンネルに@DiscordSyncを招待

`/invite @DiscordSync`

- Slackに転送する場合

該当チャンネルに対し、`"discord2slack": true`を有効にする。簡単。

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
