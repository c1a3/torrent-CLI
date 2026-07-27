# torrent-CLI

A simple browser-based torrent downloader with a minimial interface. Upload `.torrent` files or paste magnet links, and save any files to your device without leaving your browser.

## Terminal Commands

| Command | Description |
|---------|-------------|
| `help` | Show available commands |
| `upload` | Upload a `.torrent` file |
| `magnet <uri>` | Add a magnet link |
| `status` / `ls` | List all torrents |
| `info <id>` | Show detailed torrent info |
| `files <id>` | List files in a torrent |
| `download <id>` | Download completed torrent files |
| `cancel <id>` | Cancel an active download |
| `system` | Show server system info |
| `clear` | Clear the terminal |
| `about` | About this application |

## Project Structure

```
torrent-CLI/
├── main.go                 # HTTP server entry point
├── server.go               # Route definitions and HTTP handlers
├── torrent_manager.go      # Torrent engine (download lifecycle)
├── websocket.go            # WebSocket hub for real-time updates
├── models.go               # Shared data structures
├── util.go                 # Utility functions
├── static/
│   ├── index.html          # Terminal UI page
│   ├── style.css           # CRT retro theme
│   └── app.js              # Frontend logic
├── Dockerfile              # Multi-stage Docker build
├── render.yaml             # Render deployment blueprint
├── go.mod                  # Go module definition
└── go.sum                  # Dependency checksums
```
## Installation

## Quick Start (Local)

### Prerequisites
- **Go** 1.22 or higher ([download](https://go.dev/dl/))

### Run locally

```bash
git clone https://github.com/c1a3/torrent-CLI.git
cd torrent-CLI
go run .
```

Open [http://localhost:10000](http://localhost:10000) in your browser.

### Example

```bash
./torrent-CLI archlinux.iso.torrent ./downloads
```

This downloads the torrent content to the `./downloads` directory, displaying progress updates.

## Dependencies

- [github.com/anacrolix/torrent](github.com/anacrolix/torrent): A Go library for handling BitTorrent operations.

## Acknowledgments

- Built with the `anacrolix/torrent` library.
- Inspired by the need for a simple, lightweight torrent client in Go.

## License

This project is licensed under the GPL-3.0 License. See the [LICENSE](LICENSE) file for details.
