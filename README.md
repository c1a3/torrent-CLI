# torrent-CLI

A simple command-line torrent client written in Go, utilizing the `anacrolix/torrent` library to download files from `.torrent` files. This client supports downloading torrents to a specified output directory and provides real-time progress updates.

## Features

- Download files from `.torrent` files.
- Display torrent details (name, size, and save location).
- Real-time progress monitoring (percentage, downloaded size, and active peers). 
- Human-readable file size formatting (B, KB, MB, GB, TB).

## Prerequisites

- **Go**: Version 1.16 or higher. Download from golang.org or use a package manager.
- A valid `.torrent` file for testing downloads (Already provided in the repo)

## Installation

1. **Clone the Repository**:

   ```bash
   https://github.com/c1a3/torrent-CLI.git
   cd torrent-CLI
   ```

2. **Install Dependencies**: The project uses the `github.com/anacrolix/torrent` library. Install it with:

   ```bash
   go get github.com/anacrolix/torrent
   ```

3. **Initialize Go Module** (if not already done):

   ```bash
   go mod init torrent-CLI
   ```

## Project Structure

```
torrent-CLI/
│
├── main.go                 # Main entry point of the program
├── client_main.go          # Core torrent client logic
├── util.go                 # Utility functions (e.g., formatBytes)
├── go.mod                  # Go module definition and dependencies
└── archlinux.iso.torrent   # Sample .torrent file
```

## Compilation

1. **Navigate to Project Directory**:

   ```bash
   cd torrent-CLI
   ```

2. **Compile the Program**:

   - To create an executable with the default name:

     ```bash
     go build
     ```

     This generates an executable.

   - To specify a custom output name:

     ```bash
     go build -o <preferred-name>
     ```

## Usage

Run the compiled executable with a `.torrent` file and an optional output directory:

```bash
./torrent-CLI path/to/your.torrent [output_directory]
```

- `path/to/your.torrent`: Path to a valid `.torrent` file.
- `output_directory`: (Optional) Directory to save downloaded files. Defaults to the current directory.

### Example

```bash
./torrent-CLI archlinux.iso.torrent ./downloads
```

This downloads the torrent content to the `./downloads` directory, displaying progress updates.

## Output

The client displays:

- Torrent name, total size, and save location.
- Real-time progress, including:
  - Percentage completed.
  - Amount downloaded (in human-readable format).
  - Number of active peers.
- A completion message when the download finishes.

Example output:

```
Torrent: sample-file.iso
Size: 1.23 GB
Saving to: ./downloads
Progress: 45.67% | Downloaded: 562.34 MB | Peers: 12
```

## Dependencies

- [github.com/anacrolix/torrent](github.com/anacrolix/torrent): A Go library for handling BitTorrent operations.

## License

This project is licensed under the MIT License. See the LICENSE file for details.

## Acknowledgments

- Built with the `anacrolix/torrent` library.
- Inspired by the need for a simple, lightweight torrent client in Go.

## Note
This project still requires some UI/UX changed within the CLI. Additional measures for error handling are also required to be added. 
