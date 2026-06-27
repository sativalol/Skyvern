# Skyvern Updates - June 2026

We have added three significant features to the Skyvern codebase to improve expandability, debugging visibility, and intelligence.

---

## 1. Dynamic Lua Scripting Engine
To allow scriptable command creation without recompiling the Go binary, we added a Lua scripting plugin.

* **Script Location:** `plugins/lua/*.lua`
* **Features:**
  * Auto-loads all `.lua` scripts at boot.
  * Registers Discord commands dynamically via `skyvern.register_command`.
  * Fully thread-safe runtime (instantiates a fresh Lua state for each execution).
  * Exposes command arguments and context properties directly.
* **Lua context methods:**
  * `ctx:reply(text)` - Send an embed response.
  * `ctx:send_text(text)` - Send a raw text message.
  * `ctx:guild_id()` - Get the current Guild ID.
  * `ctx:channel_id()` - Get the current Channel ID.
  * `ctx:author_id()` - Get the message author's user ID.
  * `ctx:author_tag()` - Get the author's username.
  * `ctx:args()` - Returns a list (table) of space-separated arguments.
* **Reload Command:** Execute `.reloadlua` in Discord to hot-reload all scripts on the fly.
* **Relevant files:** [lua_plugin.go](file:///C:/Users/vir/Documents/percs1/n/prc/skyvern/internal/plugins/lua_plugin/lua_plugin.go), [main.go](file:///C:/Users/vir/Documents/percs1/n/prc/skyvern/main.go)

---

## 2. TUI BoltDB Browser
An interactive database browser has been integrated directly into the Terminal User Interface (`Tab 6`).

* **Features:**
  * **Database toggle:** Press `D` / `d` to switch between `bots.db` (credential storage) and `palantir.db` (event logs).
  * **Search & Filter:** Press `/` to search and filter keys inside the selected bucket.
  * **Layout:** Displays a 3-column layout (Buckets -> Keys -> Value details) with pretty-printed JSON indents for stored objects.
  * **Navigation:** `Left`/`Right` or `h`/`l` to switch panes; `Up`/`Down` or `k`/`j` to select list entries.
* **Relevant files:** [dbbrowser.go](file:///C:/Users/vir/Documents/percs1/n/prc/skyvern/pkg/tui/dbbrowser.go), [tui.go](file:///C:/Users/vir/Documents/percs1/n/prc/skyvern/pkg/tui/tui.go), [views.go](file:///C:/Users/vir/Documents/percs1/n/prc/skyvern/pkg/tui/views.go)

---

## 3. Server History RAG (AI Integration)
To enable the bot to answer questions using logged server history, we implemented a retrieval-augmented generation pipeline.

* **Command:** `.asklogs <query>` (aliases: `.loggpt`, `.historyask`, `.raglogs`)
* **How it works:**
  1. Pulls up to 1,000 recent logs in the current Guild from the `AuditLogs` bucket inside `palantir.db`.
  2. Runs a local keyword-overlap similarity ranking against the query.
  3. Formats and injects the top 15 matching logs into the LLM system prompt context.
  4. Deducts 1 token from the user's AI balance and logs the conversation.
* **Relevant files:** [asklogs.go](file:///C:/Users/vir/Documents/percs1/n/prc/skyvern/internal/commands/utility/asklogs.go), [commands.go](file:///C:/Users/vir/Documents/percs1/n/prc/skyvern/internal/commands/commands.go)

---

## 4. Anti-Nuke Emergency Webhooks
To protect servers even if modlog channels are deleted during a raid/nuke attempt, we added support for an emergency Discord Webhook.

* **Setup:** Exposes a `NukeWebhookURL` setting in the `AntinukeCfg` schema.
* **How it works:** When protection thresholds are breached, the bot formats a payload outlining the perpetrator, action category, exact counts, and the punishment applied. It fires a concurrent POST request directly to the external webhook.
* **Relevant files:** [antinuke.go](file:///C:/Users/vir/Documents/percs1/n/prc/skyvern/internal/manager/antinuke.go), [settings.go](file:///C:/Users/vir/Documents/percs1/n/prc/skyvern/internal/storage/settings.go)

---

## 5. Web UI Dashboard
A premium dark-themed web management interface has been added.

* **Port/Config:** Binds by default to port `8080` (overridden via the `SKYVERN_WEB_PORT` environment variable).
* **Endpoints:**
  * `/` - Serves the web-panel SPA.
  * `/api/bots` - Lists all bot configurations and active statuses.
  * `/api/bot/start?id=ID` - Launches a stopped bot.
  * `/api/bot/stop?id=ID` - Stops an active bot.
  * `/api/stats` - Exposes memory, command counts, and execution metrics.
* **Aesthetic:** Features a responsive layout with a CSS glassmorphism card system, pulsating active state indicators, and auto-refresh polling intervals.
* **Relevant files:** [web.go](file:///C:/Users/vir/Documents/percs1/n/prc/skyvern/internal/web/web.go), [main.go](file:///C:/Users/vir/Documents/percs1/n/prc/skyvern/main.go)

