package main

import "time"

type TorrentState string

const (
	StateAdding      TorrentState = "adding"
	StateDownloading TorrentState = "downloading"
	StateCompleted   TorrentState = "completed"
	StateCancelled   TorrentState = "cancelled"
	StateError       TorrentState = "error"
)

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

type FileInfo struct {
	Path          string  `json:"path"`
	Size          int64   `json:"size"`
	SizeFormatted string  `json:"sizeFormatted"`
	Progress      float64 `json:"progress"`
}

type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
	Time    string      `json:"time"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

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

type MagnetRequest struct {
	URI string `json:"uri"`
}
