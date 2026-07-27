package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
)

const (
	defaultMaxTorrents = 3
	warnSizeBytes      = 200 * 1024 * 1024 // 200 MB — warn if torrent exceeds this
	metadataTimeout    = 2 * time.Minute    // timeout waiting for magnet metadata
)

// ManagedTorrent wraps a torrent with management state.
type ManagedTorrent struct {
	ID        string
	Torrent   *torrent.Torrent
	State     TorrentState
	AddedAt   time.Time
	Error     string
	cancel    chan struct{}
	lastBytes int64
	speed     int64
	mu        sync.RWMutex
}

// TorrentManager manages all torrent downloads and broadcasts progress.
type TorrentManager struct {
	client    *torrent.Client
	torrents  map[string]*ManagedTorrent
	mu        sync.RWMutex
	hub       *Hub
	startTime time.Time
	dataDir   string
	bandwidth int64 // atomic — total bytes served for file downloads
}

// NewTorrentManager initializes the torrent engine and returns a manager.
func NewTorrentManager(hub *Hub) (*TorrentManager, error) {
	dir := os.Getenv("DOWNLOAD_DIR")
	if dir == "" {
		dir = "/tmp/downloads"
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating download directory: %v", err)
	}

	cfg := torrent.NewDefaultClientConfig()
	cfg.DataDir = dir
	cfg.Seed = false    // Don't seed — save bandwidth on free tier
	cfg.NoUpload = true // Save bandwidth

	client, err := torrent.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("initializing torrent client: %v", err)
	}

	log.Printf("[torrent] client initialized, data dir: %s", dir)

	return &TorrentManager{
		client:    client,
		torrents:  make(map[string]*ManagedTorrent),
		hub:       hub,
		startTime: time.Now(),
		dataDir:   dir,
	}, nil
}

// AddFromFile accepts raw .torrent file bytes and starts the download.
func (tm *TorrentManager) AddFromFile(data []byte) (*TorrentInfo, error) {
	if err := tm.checkCapacity(); err != nil {
		return nil, err
	}

	mi, err := metainfo.Load(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("invalid torrent file: %v", err)
	}

	t, err := tm.client.AddTorrent(mi)
	if err != nil {
		return nil, fmt.Errorf("adding torrent: %v", err)
	}

	mt := tm.registerTorrent(t)
	go tm.manageTorrent(mt)

	return tm.buildTorrentInfo(mt), nil
}

// AddFromMagnet accepts a magnet URI and starts the download.
func (tm *TorrentManager) AddFromMagnet(uri string) (*TorrentInfo, error) {
	if err := tm.checkCapacity(); err != nil {
		return nil, err
	}

	t, err := tm.client.AddMagnet(uri)
	if err != nil {
		return nil, fmt.Errorf("adding magnet link: %v", err)
	}

	mt := tm.registerTorrent(t)
	go tm.manageTorrent(mt)

	return tm.buildTorrentInfo(mt), nil
}

// ListTorrents returns info about all torrents.
func (tm *TorrentManager) ListTorrents() []TorrentInfo {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	result := make([]TorrentInfo, 0, len(tm.torrents))
	for _, mt := range tm.torrents {
		result = append(result, *tm.buildTorrentInfo(mt))
	}
	return result
}

// GetTorrent returns info about a single torrent by ID.
func (tm *TorrentManager) GetTorrent(id string) (*TorrentInfo, error) {
	tm.mu.RLock()
	mt, ok := tm.torrents[id]
	tm.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("torrent not found: %s", id)
	}
	return tm.buildTorrentInfo(mt), nil
}

// CancelTorrent stops and removes a torrent.
func (tm *TorrentManager) CancelTorrent(id string) error {
	tm.mu.Lock()
	mt, ok := tm.torrents[id]
	if !ok {
		tm.mu.Unlock()
		return fmt.Errorf("torrent not found: %s", id)
	}
	tm.mu.Unlock()

	mt.mu.Lock()
	if mt.State == StateDownloading || mt.State == StateAdding {
		close(mt.cancel)
	}
	mt.State = StateCancelled
	mt.mu.Unlock()

	mt.Torrent.Drop()
	log.Printf("[torrent] cancelled: %s (%s)", mt.ID, mt.Torrent.Name())
	return nil
}

// GetTorrentFiles returns the list of files in a torrent.
func (tm *TorrentManager) GetTorrentFiles(id string) ([]FileInfo, error) {
	tm.mu.RLock()
	mt, ok := tm.torrents[id]
	tm.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("torrent not found: %s", id)
	}

	// Info might not be available yet (e.g. magnet still resolving)
	select {
	case <-mt.Torrent.GotInfo():
	default:
		return []FileInfo{}, nil
	}

	files := mt.Torrent.Files()
	result := make([]FileInfo, 0, len(files))
	for _, f := range files {
		progress := float64(0)
		if f.Length() > 0 {
			progress = float64(f.BytesCompleted()) / float64(f.Length()) * 100
		}
		result = append(result, FileInfo{
			Path:          f.DisplayPath(),
			Size:          f.Length(),
			SizeFormatted: formatBytes(f.Length()),
			Progress:      progress,
		})
	}
	return result, nil
}

// GetFilePath returns the absolute filesystem path for a file within a torrent.
func (tm *TorrentManager) GetFilePath(id, filePath string) (string, error) {
	tm.mu.RLock()
	mt, ok := tm.torrents[id]
	tm.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("torrent not found: %s", id)
	}

	mt.mu.RLock()
	state := mt.State
	mt.mu.RUnlock()

	if state != StateCompleted && state != StateDownloading {
		return "", fmt.Errorf("torrent is not ready for download (state: %s)", state)
	}

	// Find the matching file in the torrent
	select {
	case <-mt.Torrent.GotInfo():
	default:
		return "", fmt.Errorf("torrent metadata not available yet")
	}

	for _, f := range mt.Torrent.Files() {
		if f.DisplayPath() == filePath {
			if f.BytesCompleted() < f.Length() {
				return "", fmt.Errorf("file not fully downloaded yet (%.1f%%)", float64(f.BytesCompleted())/float64(f.Length())*100)
			}
			// Construct the full path
			fullPath := filepath.Join(tm.dataDir, f.Path())
			// Verify the file exists
			if _, err := os.Stat(fullPath); os.IsNotExist(err) {
				return "", fmt.Errorf("file not found on disk (may have been cleaned up)")
			}
			return fullPath, nil
		}
	}
	return "", fmt.Errorf("file not found in torrent: %s", filePath)
}

// GetSystemInfo returns current server runtime info.
func (tm *TorrentManager) GetSystemInfo() SystemInfo {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	tm.mu.RLock()
	active := 0
	completed := 0
	for _, mt := range tm.torrents {
		mt.mu.RLock()
		switch mt.State {
		case StateDownloading, StateAdding:
			active++
		case StateCompleted:
			completed++
		}
		mt.mu.RUnlock()
	}
	total := len(tm.torrents)
	tm.mu.RUnlock()

	bw := atomic.LoadInt64(&tm.bandwidth)
	uptime := time.Since(tm.startTime)

	return SystemInfo{
		Uptime:             formatDuration(uptime),
		MemoryUsedMB:       int(memStats.Alloc / 1024 / 1024),
		MemoryTotalMB:      int(memStats.Sys / 1024 / 1024),
		ActiveTorrents:     active,
		CompletedTorrents:  completed,
		TotalTorrents:      total,
		BandwidthUsed:      bw,
		BandwidthFormatted: formatBytes(bw),
	}
}

// AddBandwidth adds bytes to the bandwidth counter (called when serving files).
func (tm *TorrentManager) AddBandwidth(n int64) {
	atomic.AddInt64(&tm.bandwidth, n)
}

// Close shuts down the torrent client gracefully.
func (tm *TorrentManager) Close() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	for _, mt := range tm.torrents {
		mt.mu.Lock()
		if mt.State == StateDownloading || mt.State == StateAdding {
			select {
			case <-mt.cancel:
			default:
				close(mt.cancel)
			}
		}
		mt.mu.Unlock()
	}
	tm.client.Close()
	log.Println("[torrent] client shut down")
}

// --- Private methods ---

func (tm *TorrentManager) checkCapacity() error {
	maxStr := os.Getenv("MAX_TORRENTS")
	maxT := defaultMaxTorrents
	if maxStr != "" {
		fmt.Sscanf(maxStr, "%d", &maxT)
	}

	tm.mu.RLock()
	active := 0
	for _, mt := range tm.torrents {
		mt.mu.RLock()
		if mt.State == StateDownloading || mt.State == StateAdding {
			active++
		}
		mt.mu.RUnlock()
	}
	tm.mu.RUnlock()

	if active >= maxT {
		return fmt.Errorf("maximum concurrent torrents reached (%d). Cancel one first", maxT)
	}
	return nil
}

func (tm *TorrentManager) registerTorrent(t *torrent.Torrent) *ManagedTorrent {
	id := generateID()
	mt := &ManagedTorrent{
		ID:      id,
		Torrent: t,
		State:   StateAdding,
		AddedAt: time.Now(),
		cancel:  make(chan struct{}),
	}

	tm.mu.Lock()
	tm.torrents[id] = mt
	tm.mu.Unlock()

	log.Printf("[torrent] registered: %s", id)
	return mt
}

func (tm *TorrentManager) manageTorrent(mt *ManagedTorrent) {
	// Wait for torrent metadata (info) to become available
	select {
	case <-mt.Torrent.GotInfo():
		log.Printf("[torrent] %s: got info — %s (%s)", mt.ID, mt.Torrent.Name(), formatBytes(mt.Torrent.Length()))
	case <-mt.cancel:
		mt.mu.Lock()
		mt.State = StateCancelled
		mt.mu.Unlock()
		return
	case <-time.After(metadataTimeout):
		mt.mu.Lock()
		mt.State = StateError
		mt.Error = "timed out waiting for torrent metadata"
		mt.mu.Unlock()
		tm.hub.Broadcast(WSMessage{
			Type: "torrent_error",
			Payload: map[string]interface{}{
				"id":    mt.ID,
				"error": mt.Error,
			},
		})
		mt.Torrent.Drop()
		return
	}

	// Warn if torrent is large
	if mt.Torrent.Length() > warnSizeBytes {
		tm.hub.Broadcast(WSMessage{
			Type: "warning",
			Payload: map[string]interface{}{
				"id":      mt.ID,
				"message": fmt.Sprintf("Large torrent (%s). May approach memory limits on free tier.", formatBytes(mt.Torrent.Length())),
			},
		})
	}

	// Start downloading
	mt.mu.Lock()
	mt.State = StateDownloading
	mt.mu.Unlock()

	mt.Torrent.DownloadAll()

	// Broadcast that metadata is resolved
	tm.hub.Broadcast(WSMessage{
		Type:    "torrent_info",
		Payload: tm.buildTorrentInfo(mt),
	})

	// Progress loop
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-mt.cancel:
			mt.mu.Lock()
			mt.State = StateCancelled
			mt.mu.Unlock()
			tm.hub.Broadcast(WSMessage{
				Type:    "torrent_cancelled",
				Payload: map[string]interface{}{"id": mt.ID},
			})
			return

		case <-ticker.C:
			completed := mt.Torrent.BytesCompleted()
			total := mt.Torrent.Length()

			mt.mu.Lock()
			mt.speed = completed - mt.lastBytes
			mt.lastBytes = completed
			mt.mu.Unlock()

			if total > 0 && completed >= total {
				mt.mu.Lock()
				mt.State = StateCompleted
				mt.speed = 0
				mt.mu.Unlock()

				tm.hub.Broadcast(WSMessage{
					Type:    "torrent_completed",
					Payload: tm.buildTorrentInfo(mt),
				})
				log.Printf("[torrent] %s: completed — %s", mt.ID, mt.Torrent.Name())
				return
			}

			tm.hub.Broadcast(WSMessage{
				Type:    "progress",
				Payload: tm.buildTorrentInfo(mt),
			})
		}
	}
}

func (tm *TorrentManager) buildTorrentInfo(mt *ManagedTorrent) *TorrentInfo {
	mt.mu.RLock()
	state := mt.State
	speed := mt.speed
	errStr := mt.Error
	mt.mu.RUnlock()

	info := &TorrentInfo{
		ID:      mt.ID,
		State:   state,
		AddedAt: mt.AddedAt,
		Error:   errStr,
	}

	// If we have torrent info (metadata resolved), populate the details
	select {
	case <-mt.Torrent.GotInfo():
		total := mt.Torrent.Length()
		completed := mt.Torrent.BytesCompleted()
		progress := float64(0)
		if total > 0 {
			progress = float64(completed) / float64(total) * 100
		}
		remaining := total - completed

		stats := mt.Torrent.Stats()

		info.Name = mt.Torrent.Name()
		info.Size = total
		info.SizeFormatted = formatBytes(total)
		info.Progress = progress
		info.BytesCompleted = completed
		info.Speed = speed
		info.SpeedFormatted = formatSpeed(speed)
		info.Peers = stats.ActivePeers
		info.ETA = formatETA(remaining, speed)
	default:
		info.Name = "Resolving metadata..."
		info.SizeFormatted = "—"
		info.SpeedFormatted = "—"
		info.ETA = "—"
	}

	return info
}

// formatDuration returns a human-friendly duration string.
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
