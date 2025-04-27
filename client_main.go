package main

import (
	"fmt"
	"os"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
)

func runTorrentClient(torrentFile, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %v", err)
	}

	cfg := torrent.NewDefaultClientConfig()
	cfg.DataDir = outputDir
	client, err := torrent.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("initializing client: %v", err)
	}
	defer client.Close()

	metaInfo, err := metainfo.LoadFromFile(torrentFile)
	if err != nil {
		return fmt.Errorf("loading torrent file: %v", err)
	}

	t, err := client.AddTorrent(metaInfo)
	if err != nil {
		return fmt.Errorf("adding torrent: %v", err)
	}

	<-t.GotInfo()

	fmt.Printf("Torrent: %s\n", t.Name())
	fmt.Printf("Size: %s\n", formatBytes(t.Length()))
	fmt.Printf("Saving to: %s\n", outputDir)

	t.DownloadAll()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-t.Closed():
			fmt.Println("\nDownload completed or torrent closed!")
			return nil
		case <-ticker.C:
			bytesCompleted := t.BytesCompleted()
			totalBytes := t.Length()
			progress := float64(bytesCompleted) / float64(totalBytes) * 100
			stats := t.Stats()

			fmt.Printf("\rProgress: %.2f%% | Downloaded: %s | Peers: %d",
				progress,
				formatBytes(bytesCompleted),
				stats.ActivePeers)
		}
	}
}
