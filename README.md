# Skyvern 

A multi-instance Discord bot runner and moderation tool built in Go, managed directly from a terminal user interface (TUI).

> **Warning:** Skyvern is currently in **early-release**. Features are subject to change, and bugs may occur.

---

## Getting Started

### Prerequisites

* Go 1.21+

### Build and Run

You can build the executable directly, or use our compiler script:

#### Builder
```bash
# Run the compiler
go run build.go
```
This prints a menu to let you choose your target platform (Windows, macOS, Linux, Android) and automatically builds the binary with correct env configuration.

#### Running on Android (Termux)
If you built the binary for Android (Termux / arm64):
1. Transfer the compiled `skyvern` binary to your Android device (e.g. into your `Download` folder).
2. Open Termux on your device.
3. Copy the binary to your Termux home directory:
   ```bash
   cp /where/you/saved/skyvern ~/skyvern
   ```
4. Make the binary executable:
   ```bash
   chmod +x ~/skyvern
   ```
5. Run Skyvern:
   ```bash
   ./skyvern
   ```

---

## TUI Navigation

Built with Bubble Tea. Use **`Tab`** to cycle through the panels:

* **`Tab 0` Dashboard** – Active bot instances, hardware usage (CPU/RAM), etc.
* **`Tab 1` Settings** – Naming, prefixes, embed structures, and theme setups.
* **`Tab 2` Palantir** – Logs and tracking cfg.
* **`Tab 3` Lavalink** - See lavalink logs.

> **Controls:** Press **`E`** to edit configurations within any tab. Use **`Tab`** or **`Enter`** to switch inputs, and **`Esc`** to discard changes.

---

## Features

* **Moderation** – `ban`, `warn`, `slowmode`, `temproles`, cleanups, plus more management features.
* **Utility** – `starboard`, `autoresponder`, `snipes`, and custom tags.
* **General & Fun** – `whois`, `birthdays`, `quotes`, MyAnimeList lookups, and lyrics tracking.

- Note, some haven't been tested fully.
---

## Palantir Logging

Saves every event (message updates/deletions, member changes, role updates, voice activity) into a `palantir.db` file.

### Layout

* **Batching:** Event logs stream to a buffered channel and commit to SQLite in batches of 100 (or every 500ms) to keep the Discord gateway loop unblocked.
* **Cache:** Prefixes, active filters, and anti-spam limits reside in memory to drop unnecessary database reads on incoming messages.

### TUI Filters (`Tab 2`)

* **Palantir Enabled** – Global logging toggle.
* **Blocked Servers** – server IDs to ignore.
* **Blocked Channels** – channel IDs to ignore.
* **Blocked Users** – user IDs to ignore.
* **Blocked Events** – Specific categories to drop (`messages`, `members`, `roles`, `channels`, `invites`, `emojis`, `voice`, `server`).

---

## Database Encryption

All sensitive credentials (such as Discord Bot Tokens and AI API keys) are stored encrypted at rest inside BoltDB (`bots.db`) using authenticated **AES-256-GCM**.

### Key Configuration

The 32-byte encryption key is derived using SHA-256 from your master key string. You can configure this key in one of two ways:

1. **Local Configuration File (Recommended for Local Use)**:
   Add a `crypt_key` field to your `tui_config.json` next to the binary:
   ```json
   {
     "storage_location": "local",
     "crypt_key": "your-secret-master-key-here"
   }
   ```
2. **Environment Variable (Recommended for Production)**:
   Set the `SKYVERN_CRYPT_KEY` environment variable in your operating system. This is safer because it prevents database decryption even if your application directory is leaked or stolen.

If neither is set, Skyvern will print a warning on boot and use a fallback default key. Plaintext credentials from older database versions are automatically migrated to encrypted form on their next write transaction.

---

## Plugins System

Skyvern features a modular, in-tree plugin registry defined in **[plugins.go](file:///C:/Users/vir/Documents/percs1/n/prc/skyvern/internal/plugins/plugins.go)**. This allows modularizing the bot's codebase and registering custom command groups and event triggers cleanly.

### How it works (`plugins.go`)
Plugins must implement the `Plugin` interface:
```go
type Plugin interface {
    Name() string
    Init(db *storage.DB, mgr *manager.Manager) error
    Commands() []*manager.Command
}
```
* **`Name() string`**: Returns the unique registry handle.
* **`Init(...)`**: Grants access to the Bolt database instance (`storage.DB`) and session manager (`manager.Manager`), allowing you to hook into Discord events or initialize databases.
* **`Commands()`**: Returns list of bot commands to mount.

Every plugin calls `plugins.Register(&MyPlugin{})` inside its self-contained `init()` function. By anonymously importing the plugins in **[main.go](file:///C:/Users/vir/Documents/percs1/n/prc/skyvern/main.go)**:
```go
import _ "skyvern/internal/plugins/economy"
```
The packages are executed at boot, appending themselves to the central loaded registry slice retrieved via `plugins.Loaded()`.

---

### Built-in Plugins

#### 1. `link` (Multi-Node Link & Peer Synchronization)
Enables running bot workers across multiple machines while orchestrating them centrally from a single Controller Web UI.
* **Features:**
  1. **Centralized Configuration Sync:** Peer nodes query `/api/link/config?bot=ID` to fetch bot tokens and credentials on-demand, removing the need for local databases.
  2. **Log Streaming Pipeline:** Peers post local status reports and execution console confirmations back to the Controller `/api/link/logs`.
  3. **Load Balancing Scheduler:** The controller dynamically schedules new bot executions on whichever connected peer has the lowest CPU and memory utilization.
  4. **Web UI Peer Visualizer:** Visualizes active peer nodes, resource allocations, up-to-date latency, and running bots directly inside the Web UI dashboard on the **Peers** tab.
* **How to run:**
  * **On Controller Node:** Start the web server:
    ```bash
    $env:SKYVERN_NODE_TOKEN="my-secret-token"
    go run main.go
    ```
  * **On Peer Node:** Run in agent mode, pointing to the Controller host:
    ```bash
    $env:SKYVERN_CONTROLLER="http://controller-ip:8080"
    $env:SKYVERN_NODE_TOKEN="my-secret-token"
    go run main.go --agent
    ```

#### 2. `captcha` (Verification Workflow)
Automates user verification on join.
* **How to use:**
  1. Create a `verification` text channel.
  2. Set up the role given to verified members.
  3. The bot registers interactive message buttons `captcha_start:*` and `captcha_select:*` for image challenges.
  4. Once resolved, the verified role is assigned automatically.

#### 3. `customcommands` (Dynamic Custom Command Triggers)
Enables server staff to set up static text replies for frequently asked questions.
* **Usage Commands:**
  * Create: `.customcmd create <trigger> [description]`
  * Delete: `.customcmd delete <trigger>`
  * List: `.customcmd list`

#### 4. `economy` (Ecosystem Games & Shop)
Hosts virtual currency wallets and betting games.
* **Usage Commands:**
  * Check wallet balance: `.balance` or `.bal`
  * Claim daily credits: `.daily`
  * Play Blackjack: `.blackjack <bet>` or `.bj <bet>`
  * High-Low guessing game: `.hl <bet>`
  * Shop operations: `.shop` (lists items to buy/redeem)

#### 5. `fun` (Entertainment Controls)
Basic server amusement commands.
* **Usage Commands:**
  * `.coinflip` (results in Heads or Tails)

#### 6. `lua_plugin` (Gopher-Lua Dynamic Extensibility)
Executes hot-reloadable Lua scripts inside `plugins/lua/` folder.
* **Usage Commands:**
  * Mount a new script: Place a `.lua` script inside `plugins/lua/` defining `skyvern.register_command`.
  * Reload changes instantly: `.reloadlua`

#### 7. `moon` (Astronomy ASCII graphics)
Prints current lunar cycle calculations inside raw markdown blocks.
* **Usage Commands:**
  * `.moon` or `.mooncycle`

#### 8. `vouch` (Reputation Scoring)
Maintains client vouch records.
* **Usage Commands:**
  * `.vouch @user +1 <reason>`
  * `.vouches [@user] [page]` (shows paginated vouches history)

---

## Latest Updates
See [updates.md](file:///C:/Users/vir/Documents/percs1/n/prc/skyvern/updates.md) for full details on recent features:
* **Distributed Peer Sync & Load Balancing:** Automated scheduling on remote workers, centralized config sync, log pipelines, and Web UI Peer Visualizer.
* **Dynamic Lua Plugins:** Write commands dynamically in `plugins/lua/` and reload them on the fly with `.reloadlua`.
* **TUI DB Browser:** View and search database buckets and keys for `bots.db` and `palantir.db` inside TUI (`Tab 6`).
* **Server History RAG:** Ask AI questions about the server's history using `.asklogs`.
* **Emergency Webhook Alerts:** Configure off-channel Discord webhooks to alert you of raid/nuke attempts instantly.
* **Premium Web UI Dashboard:** Control bot instances, monitor server memory, and view commands log dynamically via a dark-mode browser portal (running on port `8080` by default).
