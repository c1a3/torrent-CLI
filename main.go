package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: torrent_client <torrent_file> [output_dir]")
		os.Exit(1)
	}

	torrentFile := os.Args[1]
	outputDir := "."
	if len(os.Args) > 2 {
		outputDir = os.Args[2]
	}

	// This starts the client
	if err := runTorrentClient(torrentFile, outputDir); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
