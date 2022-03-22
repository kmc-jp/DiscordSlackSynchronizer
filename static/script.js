var Settings = [];

class GuildSettings {
    constructor(guild_setting) {
        this.discord_server = guild_setting.discord_server
        this.channel = []
        for (let chan of guild_setting.channel) {
            this.channel.push(new ChannelSettings(chan))
        }
    }
}

class ChannelSettings {
    constructor(channel_setting) {
        this.slack = String(channel_setting.slack);
        this.discord = String(channel_setting.discord);
        this.comment = String(channel_setting.comment);
        if (channel_setting.setting) {
            this.setting = {
                slack2discord: Boolean(channel_setting.setting.slack2discord),
                discord2slack: Boolean(channel_setting.setting.discord2slack),
                ShowChannelName: Boolean(channel_setting.setting.ShowChannelName),
                SendMuteState: Boolean(channel_setting.setting.SendMuteState),
                SendVoiceState: Boolean(channel_setting.setting.SendVoiceState)
            }
        } else {
            this.setting = {}
        }
    }

    set Comment(comment) { this.comment = String(comment) }
    set SlackChannel(slack) { this.slack = String(slack) }
    set DiscordChannel(discord) { this.discord = String(discord) }
    set SlackToDiscord(ok) { this.setting.slack2discord = Boolean(ok) }
    set DiscordToSlack(ok) { this.setting.discord2slack = Boolean(ok) }
    set ShowChannelName(ok) { this.setting.ShowChannelName = Boolean(ok) }
    set SendVoiceState(ok) { this.setting.SendVoiceState = Boolean(ok) }
    set SendMuteState(ok) { this.setting.SendMuteState = Boolean(ok) }


    get Comment() { return this.comment }
    get SlackChannel() { return this.slack }
    get DiscordChannel() { return this.discord }
    get SlackToDiscord() { return this.setting.slack2discord }
    get DiscordToSlack() { return this.setting.discord2slack }
    get ShowChannelName() { return this.setting.ShowChannelName }
    get SendVoiceState() { return this.setting.SendVoiceState }
    get SendMuteState() { return this.setting.SendMuteState }

}

window.onload = async() => {
    let save = document.querySelector("#save")
    save.onclick = async() => {
        if (!save.disabled) {
            if (window.confirm("現在の設定を保存しますか")) {
                await save_settings();
                make_alert("成功しました")
            }
        }
    }

    let user_name = await get_client_info()
    document.querySelector("#username").textContent = "ようこそ、" + user_name.UserName + "さん"

    await get_current_settings();
    await make_guild_selection();
}

const make_alert = (text, mode) => {
    let alertdiv = document.getElementById("alert");
    alertdiv.innerHTML = ""
    switch (String(mode)) {
        case "error":
            alertdiv.className = "alert alert-danger";
            alertdiv.innerText = text;
            break;
        case "info":
            alertdiv.className = "alert alert-info";
            alertdiv.innerText = text;
            break;
        case "remove":
            alertdiv.className = "";
            break;
        default:
            alertdiv.className = "alert alert-success";
            alertdiv.innerText = text;
    }
}

const get_json = async(action, params) => {
    let uri = new URL("api/", location.origin + location.pathname)

    uri.searchParams.append("action", action)

    if (params) {
        for (p in params) {
            uri.searchParams.append(p, params[p])
        }
    }

    let response = await fetch(uri, {
        method: "GET",
        credentials: 'same-origin',
    })

    if (!response.ok) {
        throw "Get Json Error"
    }

    return await response.json()
}

const post_json = async(action, data) => {
    let uri = new URL("api/", location.origin + location.pathname)

    uri.searchParams.append("action", action)
    let response = await fetch(
        uri, {
            method: "POST",
            credentials: "same-origin",
            body: JSON.stringify(data),
            header: {
                'Content-Type': 'application/json'
            }
        },
    )

    if (!response.ok) {
        throw "Post Json Error"
    }

    return await response.json()
}

const get_client_info = async() => get_json("getClientInfo")

const get_current_settings = async() => {
    let settings = await get_json("getCurrentSettings")

    for (setting of settings) {
        Settings.push(new GuildSettings(setting))
    }

    return Settings
}

const save_settings = async() => {
    let uri = new URL("api/", location.origin + location.pathname)

    uri.searchParams.append("action", "setSettings")
    let response = await fetch(
        uri, {
            method: "POST",
            credentials: "same-origin",
            body: JSON.stringify(Settings),
            header: {
                'Content-Type': 'application/json'
            }
        },
    )

    if (!response.ok) {
        throw "Post Json Error"
    }

    return response
}

const get_slack_channels = async() => await get_json("getSlackChannels")
const set_settings = async(settings) => await post_json("setSettings", settings)
const get_discord_channels = async(guild_id) => await get_json("getDiscordChannels", { "guild_id": guild_id })
const get_discord_guild_identitiy = async(guild_id) => {
    return await get_json("getDiscordGuildIdentity", { "guild_id": guild_id })
}

const restart = async() => {
    let uri = new URL("api/", location.origin + location.pathname)

    uri.searchParams.append("action", "restart")
    let response = await fetch(uri, {
        method: "GET",
        credentials: 'same-origin',
    })

    if (!response.ok) {
        console.log("Error: RestartError")
        return
    }

    return
}

const get_guild_setting = async(guild_id) => {
    return Settings.find(setting => setting.discord_server == guild_id)
}

const isInWindow = (elem) => {
    let viewTop = window.scrollY;
    let viewBottom = window.scrollY + document.documentElement.clientHeight;

    let rect = elem.getBoundingClientRect();

    let elemTop = rect.top + window.pageYOffset;
    let elemBottom = rect.bottom + window.pageYOffset;

    return ((elemBottom <= viewBottom) && (elemTop >= viewTop));
}

const make_guild_selection = async() => {
    // サーバ選択オプションの追加
    let guild_identities = []

    for (setting of Settings) {
        guild_identities.push(await get_discord_guild_identitiy(setting.discord_server))
    }

    let guild_select = document.querySelector("#guild_select");
    guild_select.innerHTML = "";

    let guild_option_init = document.createElement("option");
    guild_option_init.selected = true;
    guild_option_init.textContent = "Discord Guild...";

    guild_select.appendChild(guild_option_init)

    guild_identities.forEach(identity => {
        let option = document.createElement("option")
        option.value = identity.id
        option.textContent = identity.name

        guild_select.appendChild(option)
    })

    guild_select.onchange = event => {
        make_settings_list(event.target.value)
    }

    document.querySelector("#save").disabled = false

}

const make_settings_list = async(guild_id, discord_channel_list, slack_channel_list) => {
    if (!slack_channel_list && !discord_channel_list) {
        discord_channel_list = await get_discord_channels(guild_id);
        slack_channel_list = await get_slack_channels();
    }

    const add_setting = document.querySelector("#add");
    add_setting.onclick = event => {
        settings.channel.push(new ChannelSettings({ "Setting": {} }))
            make_settings_list(guild_id, discord_channel_list, slack_channel_list);
    };
        
    const accordion_div = document.querySelector("#channels");
    accordion_div.innerHTML = "";

    const settings = await get_guild_setting(guild_id)
    let settings_index = 0;

    const template_channel = document.querySelector("#template-channel").content;
    for (let setting of settings.channel) {
        const index = settings_index;
        const setting_channel = template_channel.cloneNode(true);
        const setting_id = `setting-${settings_index}`;
        setting_channel.querySelector(".setting").id = setting_id;

        const inner_id =`setting-inner-${settings_index}`
        setting_channel.querySelector(".setting-inner").id = inner_id;


        // タイトル
        const accordion = setting_channel.querySelector(".accordion-setting");
        accordion.setAttribute("data-bs-target", `#${inner_id}`)
        accordion.setAttribute("aria-controls", inner_id);
        
        const span = setting_channel.querySelector(".button-title-setting");
        const icon = setting_channel.querySelector(".button-icon-setting");
        
        let discord_channel = discord_channel_list.find(chan => { return chan.id == setting.DiscordChannel });
        if (discord_channel !== undefined) {
            switch (discord_channel.type) {
                case 0:                        
                    icon.classList.add("fa-hashtag");
                    span.innerText = discord_channel.name;
                    break;
                case 2:
                    icon.classList.add("fa-volume-up");
                    span.innerText = discord_channel.name;
                    break;
                default:
                    icon.classList.add("fa-hashtag");
                    span.innerText = discord_channel.name;
                    break;
                    break;
            }
        } else {
            icon.classList.add("fa-hashtag");
            span.innerText = "New Setting";
        }
        
        const this_setting = setting;

        // 順序入れ替え
        const button_up = setting_channel.querySelector(".btn-up-setting");
        if (index < 1) {
             button_up.disabled = true;
        }
        button_up.onclick = () => {
            if (index < 1) return false;
            settings.channel.splice(index - 1, 2, settings.channel[index], settings.channel[index - 1]);
            make_settings_list(guild_id, discord_channel_list, slack_channel_list);
            return false;
        }
        const button_down = setting_channel.querySelector(".btn-down-setting");
        if (index + 1 >= settings.channel.length) {
            button_down.disabled = true;
        }
        button_down.onclick = () => {
            if (index + 1 >= settings.channel.length) return false;
            settings.channel.splice(index, 2, settings.channel[index + 1], settings.channel[index]);
            make_settings_list(guild_id, discord_channel_list, slack_channel_list);
            return false;
        }
        
        // Discordのチャンネルの選択項目
        const select_discord = setting_channel.querySelector(".discord-channel-setting");
        const isTextChannel = !(discord_channel && discord_channel.type !== 0);
        const regenerate_select_discord = (select_discord, isTextChannel) => {
            for (let option of select_discord.querySelectorAll("[data-discordid]")) {
                select_discord.removeChild(option);
            }
            
            let option = select_discord.querySelector("option")
            if (!this_setting.DiscordChannel) {
                option.selected = true;
            }
    
            for (channel of discord_channel_list) {
                const option = document.createElement("option")
                if (this_setting.DiscordChannel == channel.id) {
                    option.selected = true;
                }
    
                if ((isTextChannel && channel.type === 0) || (!isTextChannel && channel.type === 2)) {
                    option.innerText = channel.name;
                    option.setAttribute("data-discordid", channel.id)
                    select_discord.appendChild(option)
                }
            }
        }
        regenerate_select_discord(select_discord, isTextChannel)

        select_discord.onchange = (event) => {
            if (!event.target.value) {
                this_setting.Comment = "";
                this_setting.DiscordChannel = "";
                document.querySelector(`#${setting_id} .button-title-setting`).textContent = " NewSetting"
                return;
            }
            this_setting.Comment = "#" + event.target.value;
            this_setting.DiscordChannel = event.target.options[event.target.selectedIndex].getAttribute("data-discordid");
            document.querySelector(`#${setting_id} .button-title-setting`).innerText = event.target.value;
        };


        // slackのチャンネルの選択項目生成
        const select_slack = setting_channel.querySelector(".slack-channel-setting");        
        option = select_slack.querySelector("option")
        if (!setting.SlackChannel) {
            option.selected = true;
        }

        for (channel of slack_channel_list) {
            const option = document.createElement("option")
            if (setting.SlackChannel == channel.id) {
                option.selected = true;
            }

            option.innerText = channel.name;
            option.setAttribute("data-slackid", channel.id)
            select_slack.appendChild(option)
        }
        
        select_slack.onchange = (event) => {
            if (!event.target.value) {
                setting.SlackChannel = "";
                return;
            }
            this_setting.SlackChannel = event.target.options[event.target.selectedIndex].getAttribute("data-slackid");
        };

        
        // チェック項目の作成
        // VoiceState
        const voice_state_input = setting_channel.querySelector(".send-voice-state-setting");
        if (setting.SendVoiceState) {
            voice_state_input.checked = "checked"
        }

        voice_state_input.onchange = (event) => {
            this_setting.SendVoiceState = event.target.checked == true
            this_setting.DiscordChannel = "";
        
            icon.classList.remove("fa-volume-up", "fa-hashtag");
            const select_discord = document.querySelector(`#${setting_id} .discord-channel-setting`) 
            if (!setting.SendVoiceState) {
                regenerate_select_discord(select_discord, true);
                icon.classList.add("fa-hashtag");
                span.innerText = " NewSetting"
            } else {
                regenerate_select_discord(select_discord, false);
                icon.classList.add("fa-volume-up");
                span.innerText = " NewSetting"
            }

            const mute_state = document.querySelector(`#${setting_id} .send-mute-state-setting`);
            if (mute_state) {
                mute_state.disabled = this_setting.SendVoiceState == false;
                if (!this_setting.SendVoiceState) {
                    mute_state.checked = false;
                }
            }
        }
        
        // MuteState
        const mute_state_input = setting_channel.querySelector(`.send-mute-state-setting`);
        mute_state_input.disabled = setting.SendVoiceState == false
        if (setting.SendMuteState) {
            mute_state_input.checked = "checked"
        }
        mute_state_input.onchange = (event) => {
            this_setting.SendMuteState = event.target.checked == true
        }

        // Slack to Discord
        const slack_to_discord_input = setting_channel.querySelector(`.slack-to-discord-setting`);
        if (setting.SlackToDiscord) {
            slack_to_discord_input.checked = "checked"
        }
        slack_to_discord_input.onchange = (event) => {
            this_setting.SlackToDiscord = event.target.checked == true
        }

        // Discord to Slack
        const discord_to_slack_input = setting_channel.querySelector(`.discord-to-slack-setting`);
        if (setting.DiscordToSlack) {
            discord_to_slack_input.checked = "checked"
        }
        discord_to_slack_input.onchange = (event) => {
            this_setting.DiscordToSlack = event.target.checked == true
        }

        // Appending Channel Name
        const add_channel_name_input = setting_channel.querySelector(`.add-channel-name-setting`);
        if (setting.ShowChannelName) {
            add_channel_name_input.checked = "checked"
        }
        add_channel_name_input.onchange = (event) => {
            this_setting.ShowChannelName = event.target.checked == true
        }

        const remove_button = setting_channel.querySelector(".remove-setting");
        remove_button.onclick = () => {
            settings.channel.splice(index, 1);
            make_settings_list(guild_id, discord_channel_list, slack_channel_list);
        }

        accordion_div.appendChild(setting_channel);

        settings_index += 1;
    }
}