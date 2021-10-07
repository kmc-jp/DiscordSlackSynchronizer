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
    document.onclick = hide_menues;
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

const hide_menues = () => {
    let menues = document.querySelectorAll(".dropdown.menu")
    for (let i = 0; i < menues.length; i++) {
        document.body.removeChild(menues[i])
    }
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

    let accordion_div = document.querySelector("#channels");
    accordion_div.innerHTML = "";

    let settings = await get_guild_setting(guild_id)
    let settings_index = 0;

    for (let setting of settings.channel) {
        // 右クリックのメニュー生成
        let onclick_menu = document.createElement("table");
        onclick_menu.class = "dropdown menu"
        onclick_menu.style.display = "none";
        onclick_menu.className = "dropdown menu";
        onclick_menu.style.backgroundColor = "white";

        let onclick_menu_add = document.createElement("tr");
        onclick_menu_add.className = "dropdown item";
        onclick_menu_add.innerHTML = "<td><i class='fas fa-plus'></i></td><td>Add</td>";
        onclick_menu_add.style.textAlign = "left";
        onclick_menu_add.onclick = (index => event => {
            settings.channel.splice(index, 0, new ChannelSettings({ "Setting": {} }))
            make_settings_list(guild_id, discord_channel_list, slack_channel_list);
        })(settings_index)

        let onclick_menu_delete = document.createElement("tr");
        onclick_menu_delete.className = "dropdown item";
        onclick_menu_delete.style.color = "Red";
        onclick_menu_delete.innerHTML = "<td><i class='fas fa-trash-alt'></i></td><td>Delete</td>";
        onclick_menu_delete.style.textAlign = "left";
        onclick_menu_delete.onclick = (index => () => {
            settings.channel.splice(index, 1);
            make_settings_list(guild_id, discord_channel_list, slack_channel_list);
        })(settings_index)

        onclick_menu.appendChild(onclick_menu_add);
        onclick_menu.appendChild(onclick_menu_delete);

        // 一段を作成
        let accordion_item = document.createElement("div")
        accordion_item.className = "accordion-item"

        // メニュータイトル
        let h2 = document.createElement("h2")
        h2.className = "accordion-header"
        h2.id = "channel-" + settings_index;

        let button = document.createElement("button");
        button.className = "accordion-button collapsed";
        button.type = button;
        button.setAttribute("data-bs-toggle", "collapse");
        button.setAttribute("data-bs-target", "#setting-" + settings_index);
        button.setAttribute("area-expanded", "false");
        button.setAttribute("area-controls", "setting-" + settings_index);

        h2.oncontextmenu = (onclick_menu => event => {
            event.preventDefault();
            hide_menues();
            onclick_menu.style.left = event.pageX + "px";
            onclick_menu.style.top = event.pageY + "px";
            onclick_menu.style.display = "unset";
            if (!isInWindow(onclick_menu)) {
                onclick_menu.style.top = window.scrollY + document.documentElement.clientHeight - onclick_menu.getBoundingClientRect().height + "px";
            }
            document.body.appendChild(onclick_menu);
        })(onclick_menu)

        let discord_channel = discord_channel_list.find(chan => { return chan.id == setting.DiscordChannel });

        if (discord_channel !== undefined) {
            switch (discord_channel.type) {
                case 0:
                    {
                        let icon = document.createElement("i");
                        icon.className = "fas fa-hashtag";

                        let span = document.createElement("span");
                        span.innerText = discord_channel.name;
                        span.id = "button-title-" + settings_index

                        button.appendChild(icon);
                        button.appendChild(span);

                    }
                    break;
                case 2:
                    {
                        let icon = document.createElement("i");
                        icon.className = "fas fa-volume-up";

                        let span = document.createElement("span");
                        span.innerText = discord_channel.name;
                        span.id = "button-title-" + settings_index

                        button.appendChild(icon);
                        button.appendChild(span);

                    }
                    break;
                default:
                    {
                        let icon = document.createElement("i");
                        icon.className = "fas fa-hashtag";

                        let span = document.createElement("span");
                        span.innerText = discord_channel.name;
                        span.id = "button-title-" + settings_index

                        button.appendChild(icon);
                        button.appendChild(span);
                    }
                    break;
            }
        } else {
            let icon = document.createElement("i");
            icon.className = "fas fa-hashtag";

            let span = document.createElement("span");
            span.innerText = "New Setting";
            span.id = "button-title-" + settings_index

            button.appendChild(icon);
            button.appendChild(span);
        }

        h2.appendChild(button);
        accordion_item.appendChild(h2);

        // 本質
        let accordion_collapse = document.createElement("div")
        accordion_collapse.id = "setting-" + settings_index;
        accordion_collapse.className = "accordion-collapse collapse";
        accordion_collapse.setAttribute("aria-labelledby", "channel-01");
        accordion_collapse.setAttribute("data-bs-parent", "#channels");

        let accordion_body = document.createElement("div");
        accordion_body.className = "accordion-body";

        // Channel情報の入力欄
        let settings_row = document.createElement("div");
        settings_row.className = "row g-2";

        // Discord
        let discord_channel_col = document.createElement("div");
        discord_channel_col.className = "col-md";

        let discord_channel_formfloat = document.createElement("div");
        discord_channel_formfloat.className = "form-floating";

        let input_discord_channel_label = document.createElement("label")
        input_discord_channel_label.setAttribute("for", "discord-channel-" + settings_index)
        input_discord_channel_label.innerText = "Discord Channel Name";

        let select_discord = document.createElement("select");
        // Discordのチャンネルの選択項目生成
        select_discord.className = "form-select";
        select_discord.id = "discord-channel-" + settings_index;
        select_discord.setAttribute("aria-label", "discord-channel")

        let make_discord_select = (index => (isTextChannel) => {
            select_discord.innerHTML = "";
            // select channelが空の時はSelect...を選択
            let option = document.createElement("option")

            option.innerText = "Select..."
            if (!setting.DiscordChannel) {
                option.selected = true;
                selected_discord_channel = option;
            }

            select_discord.appendChild(option)

            for (channel of discord_channel_list) {
                let option = document.createElement("option")
                if (setting.DiscordChannel == channel.id) {
                    option.selected = true;
                }

                if ((isTextChannel && channel.type === 0) || (!isTextChannel && channel.type === 2)) {
                    option.innerText = channel.name;
                    option.setAttribute("discordid", channel.id)
                    select_discord.appendChild(option)
                }
            }

            select_discord.onchange = (event) => {
                if (!event.target.value) {
                    setting.Comment = "";
                    setting.DiscordChannel = "";
                    document.querySelector("#button-title-" + index).textContent = " NewSetting"
                    return;
                }
                setting.Comment = "#" + event.target.value;
                setting.DiscordChannel = event.target.options[event.target.selectedIndex].getAttribute("discordid");
                document.querySelector("#button-title-" + index).innerText = event.target.value;
            }
        })(settings_index)


        make_discord_select(!(discord_channel && discord_channel.type !== 0));

        discord_channel_formfloat.appendChild(select_discord);
        discord_channel_formfloat.appendChild(input_discord_channel_label)

        discord_channel_col.appendChild(discord_channel_formfloat);
        settings_row.appendChild(discord_channel_col);


        // Slack
        let slack_channel_col = document.createElement("div");
        slack_channel_col.className = "col-md";

        let slack_channel_formfloat = document.createElement("div");
        slack_channel_formfloat.className = "form-floating";

        let input_slack_channel_label = document.createElement("label")
        input_slack_channel_label.setAttribute("for", "slack-channel-" + settings_index)
        input_slack_channel_label.innerText = "Slack Channel Name";

        let select_slack = document.createElement("select");
        // slackのチャンネルの選択項目生成
        select_slack.className = "form-select";
        select_slack.id = "slack-channel-" + settings_index;
        select_slack.setAttribute("aria-label", "slack-channel")

        let make_slack_select = (index => () => {
            select_slack.innerHTML = "";
            // select channelが空の時はSelect...を選択
            let option = document.createElement("option")

            option.innerText = "Select..."
            if (!setting.SlackChannel) {
                option.selected = true;
            }

            select_slack.appendChild(option)

            for (channel of slack_channel_list) {
                let option = document.createElement("option")
                if (setting.SlackChannel == channel.id) {
                    option.selected = true;
                }

                option.innerText = channel.name;
                option.setAttribute("slackid", channel.id)
                select_slack.appendChild(option)
            }

            select_slack.onchange = (event) => {
                if (!event.target.value) {
                    setting.SlackChannel = "";
                    return;
                }
                setting.SlackChannel = event.target.options[event.target.selectedIndex].getAttribute("slackid");
            }
        })(settings_index)

        make_slack_select();

        slack_channel_formfloat.appendChild(select_slack);
        slack_channel_formfloat.appendChild(input_slack_channel_label)

        slack_channel_col.appendChild(slack_channel_formfloat);
        settings_row.appendChild(slack_channel_col);

        accordion_body.appendChild(settings_row);

        // チェック項目の作成
        // VoiceState
        let voice_state_check = document.createElement("div");
        voice_state_check.className = "form-check";

        let voice_state_input = document.createElement("input");
        voice_state_input.className = "form-check-input";
        voice_state_input.type = "checkbox";
        voice_state_input.id = "send-voice-state-" + settings_index;

        if (setting.SendVoiceState) {
            voice_state_input.checked = "checked"
        }

        voice_state_input.onchange = ((index, button) => {
            return (event) => {
                setting.SendVoiceState = event.target.checked == true
                setting.DiscordChannel = "";

                button.innerHTML = "";

                if (!setting.SendVoiceState) {
                    make_discord_select(true);

                    let icon = document.createElement("i");
                    icon.className = "fas fa-hashtag";

                    let span = document.createElement("span");
                    span.innerText = " NewSetting"
                    span.id = "button-title-" + index

                    button.appendChild(icon);
                    button.appendChild(span);
                } else {
                    make_discord_select(false);

                    let icon = document.createElement("i");
                    icon.className = "fas fa-volume-up";

                    let span = document.createElement("span");
                    span.innerText = " NewSetting"
                    span.id = "button-title-" + index

                    button.appendChild(icon);
                    button.appendChild(span);
                }

                let mute_state = document.querySelector("#send-mute-state-" + index);
                if (mute_state) {
                    mute_state.disabled = setting.SendVoiceState == false;
                    if (!setting.SendVoiceState) {
                        mute_state.checked = false;
                    }
                }
            }
        })(settings_index, button)

        let voice_state_input_label = document.createElement("label");
        voice_state_input_label.className = "form-check-label";
        voice_state_input_label.setAttribute("for", "send-voice-state-" + settings_index);
        voice_state_input_label.innerText = "ボイスチャンネルとして監視・送信"

        voice_state_check.appendChild(voice_state_input);
        voice_state_check.appendChild(voice_state_input_label);

        accordion_body.appendChild(voice_state_check);

        // MuteState
        let mute_state_check = document.createElement("div");
        mute_state_check.className = "form-check";

        let mute_state_input = document.createElement("input");
        mute_state_input.className = "form-check-input";
        mute_state_input.type = "checkbox";
        mute_state_input.id = "send-mute-state-" + settings_index;

        mute_state_input.disabled = setting.SendVoiceState == false

        if (setting.SendMuteState) {
            mute_state_input.checked = "checked"
        }

        mute_state_input.onchange = (event) => {
            setting.SendMuteState = event.target.checked == true
        }

        let mute_state_input_label = document.createElement("label");
        mute_state_input_label.className = "form-check-label";
        mute_state_input_label.setAttribute("for", "send-mute-state-" + settings_index);
        mute_state_input_label.innerText = "ミュート／消音状態変化も通知";

        mute_state_check.appendChild(mute_state_input);
        mute_state_check.appendChild(mute_state_input_label);

        accordion_body.appendChild(mute_state_check);

        // Slack to Discord 
        let slack_to_discord_check = document.createElement("div");
        slack_to_discord_check.className = "form-check";

        let slack_to_discord_input = document.createElement("input");
        slack_to_discord_input.className = "form-check-input";
        slack_to_discord_input.type = "checkbox";
        slack_to_discord_input.id = "slack-to-discord-" + settings_index;

        if (setting.SlackToDiscord) {
            slack_to_discord_input.checked = "checked"
        }

        slack_to_discord_input.onchange = (event) => {
            setting.SlackToDiscord = event.target.checked == true
        }

        let slack_to_discord_input_label = document.createElement("label");
        slack_to_discord_input_label.className = "form-check-label";
        slack_to_discord_input_label.setAttribute("for", "slack-to-discord-" + settings_index);
        slack_to_discord_input_label.innerText = "SlackからDiscordへ転送"

        slack_to_discord_check.appendChild(slack_to_discord_input);
        slack_to_discord_check.appendChild(slack_to_discord_input_label);

        accordion_body.appendChild(slack_to_discord_check);

        // Discord to Slack
        let discord_to_slack_check = document.createElement("div");
        discord_to_slack_check.className = "form-check";

        let discord_to_slack_input = document.createElement("input");
        discord_to_slack_input.className = "form-check-input";
        discord_to_slack_input.type = "checkbox";
        discord_to_slack_input.id = "discord-to-slack-" + settings_index;

        if (setting.DiscordToSlack) {
            discord_to_slack_input.checked = "checked"
        }

        discord_to_slack_input.onchange = (event) => {
            setting.DiscordToSlack = event.target.checked == true
        }

        let discord_to_slack_input_label = document.createElement("label");
        discord_to_slack_input_label.className = "form-check-label";
        discord_to_slack_input_label.setAttribute("for", "discord-to-slack-" + settings_index);

        discord_to_slack_input_label.innerText = "DiscordからSlackへ転送"

        discord_to_slack_check.appendChild(discord_to_slack_input);
        discord_to_slack_check.appendChild(discord_to_slack_input_label);

        accordion_body.appendChild(discord_to_slack_check);

        let add_channel_name_check = document.createElement("div");
        add_channel_name_check.className = "form-check";

        let add_channel_name_input = document.createElement("input");
        add_channel_name_input.className = "form-check-input";
        add_channel_name_input.type = "checkbox";
        add_channel_name_input.id = "add-channel-name-" + settings_index;

        if (setting.ShowChannelName) {
            add_channel_name_input.checked = "checked"
        }

        add_channel_name_input.onchange = (event) => {
            setting.ShowChannelName = event.target.checked == true
        }

        let add_channel_name_input_label = document.createElement("label");
        add_channel_name_input_label.className = "form-check-label";
        add_channel_name_input_label.setAttribute("for", "add-channel-name-" + settings_index);
        add_channel_name_input_label.innerText = "チャンネル名を付加"


        add_channel_name_check.appendChild(add_channel_name_input);
        add_channel_name_check.appendChild(add_channel_name_input_label);

        accordion_body.appendChild(add_channel_name_check);

        accordion_collapse.appendChild(accordion_body);
        accordion_item.appendChild(accordion_collapse);
        accordion_div.appendChild(accordion_item);

        settings_index += 1;
    }
}