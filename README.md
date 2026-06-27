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

* **`link`**: Multi-node agent synchronization. Lets you control bot processes remotely on other machines via constant-time secure token authentication and request rate limits.
* **`captcha`**: User verification. Creates a verification workflow requiring new members to click buttons in Discord to solve interactive image challenges.
* **`customcommands`**: Dynamic text command mapping. Allows server staff to create static custom text triggers directly from the chat interface.
* **`economy`**: Virtual server economy. Includes virtual banks, Blackjack, High-Low, and server shop interfaces.
* **`fun`**: Basic server entertainment commands (e.g., `.coinflip`).
* **`lua_plugin`**: Gopher-Lua VM plugin runner. Scans the local `plugins/lua/` folder and loads commands written in Lua scripts, supporting hot reloads via `.reloadlua`.
* **`moon`**: Lunar cycle statistics displayed in terminal-style ASCII diagrams.
* **`vouch`**: Reputation points and vouches management system.

---

---

## Latest Updates
See [updates.md](file:///C:/Users/vir/Documents/percs1/n/prc/skyvern/updates.md) for full details on recent features:
* **Dynamic Lua Plugins:** Write commands dynamically in `plugins/lua/` and reload them on the fly with `.reloadlua`.
* **TUI DB Browser:** View and search database buckets and keys for `bots.db` and `palantir.db` inside TUI (`Tab 6`).
* **Server History RAG:** Ask AI questions about the server's history using `.asklogs`.
* **Emergency Webhook Alerts:** Configure off-channel Discord webhooks to alert you of raid/nuke attempts instantly.
* **Premium Web UI Dashboard:** Control bot instances, monitor server memory, and view commands log dynamically via a dark-mode browser portal (running on port `8080` by default).
