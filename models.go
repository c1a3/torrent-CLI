package main

import "time"

// TorrentState represents the current state of a torrent download.
type TorrentState string

const (
	StateAdding      TorrentState = "adding"
	StateDownloading TorrentState = "downloading"
	StateCompleted   TorrentState = "completed"
	StateCancelled   TorrentState = "cancelled"
	StateError       TorrentState = "error"
)

// TorrentInfo holds all displayable information about a torrent.
type TorrentInfo struct {
	ID             string       `json:"id"`
	Name           string       `json:"name"`
	Size           int64        `json:"size"`
	SizeFormatted  string       `json:"sizeFormatted"`
	State          TorrentState `json:"state"`
	Progress       float64      `json:"progress"`
	BytesCompleted int64        `json:"bytesCompleted"`
	Speed          int64        `json:"speed"`
	SpeedFormatted string       `json:"speedFormatted"`
	Peers          int          `json:"peers"`
	ETA            string       `json:"eta"`
	Files          []FileInfo   `json:"files,omitempty"`
	AddedAt        time.Time    `json:"addedAt"`
	Error          string       `json:"error,omitempty"`
}

// FileInfo holds information about a single file within a torrent.
type FileInfo struct {
	Path          string  `json:"path"`
	Size          int64   `json:"size"`
	SizeFormatted string  `json:"sizeFormatted"`
	Progress      float64 `json:"progress"`
}

// WSMessage is the JSON envelope for all WebSocket messages.
type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
	Time    string      `json:"time"`
}

// APIResponse is a standard JSON wrapper for HTTP API responses.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// SystemInfo holds server runtime information.
type SystemInfo struct {
	Uptime             string `json:"uptime"`
	MemoryUsedMB       int    `json:"memoryUsedMB"`
	MemoryTotalMB      int    `json:"memoryTotalMB"`
	ActiveTorrents     int    `json:"activeTorrents"`
	CompletedTorrents  int    `json:"completedTorrents"`
	TotalTorrents      int    `json:"totalTorrents"`
	BandwidthUsed      int64  `json:"bandwidthUsed"`
	BandwidthFormatted string `json:"bandwidthFormatted"`
}

// MagnetRequest is the JSON body for adding a magnet link.
type MagnetRequest struct {
	URI string `json:"uri"`
}
