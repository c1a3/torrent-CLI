package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// SetupRoutes configures all HTTP routes on the given ServeMux.
func SetupRoutes(mux *http.ServeMux, tm *TorrentManager, hub *Hub) {
	// API endpoints
	mux.HandleFunc("POST /api/upload", handleUpload(tm))
	mux.HandleFunc("POST /api/magnet", handleMagnet(tm))
	mux.HandleFunc("GET /api/torrents", handleListTorrents(tm))
	mux.HandleFunc("GET /api/torrents/{id}", handleGetTorrent(tm))
	mux.HandleFunc("DELETE /api/torrents/{id}", handleCancelTorrent(tm))
	mux.HandleFunc("GET /api/torrents/{id}/files", handleGetFiles(tm))
	mux.HandleFunc("GET /api/torrents/{id}/download_all", handleDownloadAll(tm))
	mux.HandleFunc("GET /api/torrents/{id}/download/{filepath...}", handleDownloadFile(tm))
	mux.HandleFunc("GET /api/system", handleSystemInfo(tm))

	// WebSocket
	mux.HandleFunc("GET /ws", func(w http.ResponseWriter, r *http.Request) {
		ServeWs(hub, w, r)
	})

	// Serve static frontend files
	staticDir := "./static"
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		log.Printf("[server] warning: static directory not found at %s", staticDir)
	}

	fs := http.FileServer(http.Dir(staticDir))
	mux.Handle("GET /", http.StripPrefix("/", fs))
}

// --- Handler factories ---

func handleUpload(tm *TorrentManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Limit upload size to 10 MB (torrent files are typically small)
		r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

		file, header, err := r.FormFile("file")
		if err != nil {
			writeError(w, http.StatusBadRequest, "failed to read uploaded file: %v", err)
			return
		}
		defer file.Close()

		if !strings.HasSuffix(strings.ToLower(header.Filename), ".torrent") {
			writeError(w, http.StatusBadRequest, "only .torrent files are accepted")
			return
		}

		data, err := io.ReadAll(file)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to read file data: %v", err)
			return
		}

		info, err := tm.AddFromFile(data)
		if err != nil {
			writeError(w, http.StatusBadRequest, "%v", err)
			return
		}

		log.Printf("[api] torrent uploaded: %s (%s)", header.Filename, info.ID)
		writeJSON(w, http.StatusOK, APIResponse{
			Success: true,
			Data:    info,
		})
	}
}

func handleMagnet(tm *TorrentManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req MagnetRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body: %v", err)
			return
		}

		if req.URI == "" {
			writeError(w, http.StatusBadRequest, "magnet URI is required")
			return
		}

		if !strings.HasPrefix(req.URI, "magnet:?") {
			writeError(w, http.StatusBadRequest, "invalid magnet URI format")
			return
		}

		info, err := tm.AddFromMagnet(req.URI)
		if err != nil {
			writeError(w, http.StatusBadRequest, "%v", err)
			return
		}

		log.Printf("[api] magnet added: %s", info.ID)
		writeJSON(w, http.StatusOK, APIResponse{
			Success: true,
			Data:    info,
		})
	}
}

func handleListTorrents(tm *TorrentManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		torrents := tm.ListTorrents()
		writeJSON(w, http.StatusOK, APIResponse{
			Success: true,
			Data:    torrents,
		})
	}
}

func handleGetTorrent(tm *TorrentManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		info, err := tm.GetTorrent(id)
		if err != nil {
			writeError(w, http.StatusNotFound, "%v", err)
			return
		}
		writeJSON(w, http.StatusOK, APIResponse{
			Success: true,
			Data:    info,
		})
	}
}

func handleCancelTorrent(tm *TorrentManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if err := tm.CancelTorrent(id); err != nil {
			writeError(w, http.StatusNotFound, "%v", err)
			return
		}
		writeJSON(w, http.StatusOK, APIResponse{
			Success: true,
			Data:    map[string]string{"id": id, "status": "cancelled"},
		})
	}
}

func handleGetFiles(tm *TorrentManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		files, err := tm.GetTorrentFiles(id)
		if err != nil {
			writeError(w, http.StatusNotFound, "%v", err)
			return
		}
		writeJSON(w, http.StatusOK, APIResponse{
			Success: true,
			Data:    files,
		})
	}
}

func handleDownloadFile(tm *TorrentManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		reqPath, err := url.PathUnescape(r.PathValue("filepath"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid file path encoding")
			return
		}

		// Security: prevent path traversal
		cleanPath := filepath.Clean(reqPath)
		if strings.Contains(cleanPath, "..") {
			writeError(w, http.StatusBadRequest, "invalid file path")
			return
		}

		fullPath, err := tm.GetFilePath(id, reqPath)
		if err != nil {
			writeError(w, http.StatusNotFound, "%v", err)
			return
		}

		// Get the file size for bandwidth tracking
		stat, err := os.Stat(fullPath)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "file stat failed: %v", err)
			return
		}

		// Track bandwidth
		tm.AddBandwidth(stat.Size())

		// Set download headers
		fileName := filepath.Base(fullPath)
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))

		log.Printf("[api] serving file download: %s (%s)", fileName, formatBytes(stat.Size()))
		http.ServeFile(w, r, fullPath)
	}
}

func handleDownloadAll(tm *TorrentManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		files, err := tm.GetTorrentFiles(id)
		if err != nil {
			writeError(w, http.StatusNotFound, "%v", err)
			return
		}

		info, err := tm.GetTorrent(id)
		if err != nil {
			writeError(w, http.StatusNotFound, "%v", err)
			return
		}

		w.Header().Set("Content-Type", "application/zip")
		// Use a safe version of the torrent name for the zip file
		safeName := strings.Map(func(r rune) rune {
			if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
				return r
			}
			return '_'
		}, info.Name)
		if safeName == "" {
			safeName = id
		}
		
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.zip\"", safeName))

		zw := zip.NewWriter(w)
		defer zw.Close()

		for _, f := range files {
			if f.Progress < 100 {
				continue // Skip files that aren't fully downloaded yet
			}

			fullPath, err := tm.GetFilePath(id, f.Path)
			if err != nil {
				continue
			}

			stat, err := os.Stat(fullPath)
			if err != nil {
				continue
			}

			// Add to zip
			fwriter, err := zw.Create(f.Path)
			if err != nil {
				log.Printf("[api] failed to create zip entry for %s: %v", f.Path, err)
				continue
			}

			file, err := os.Open(fullPath)
			if err != nil {
				log.Printf("[api] failed to open file for zip %s: %v", fullPath, err)
				continue
			}

			_, err = io.Copy(fwriter, file)
			file.Close()

			if err == nil {
				tm.AddBandwidth(stat.Size())
			}
		}
		log.Printf("[api] served zip download for torrent: %s", id)
	}
}

func handleSystemInfo(tm *TorrentManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		info := tm.GetSystemInfo()
		writeJSON(w, http.StatusOK, APIResponse{
			Success: true,
			Data:    info,
		})
	}
}

// --- Response helpers ---

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("[api] JSON encode error: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log.Printf("[api] error %d: %s", status, msg)
	writeJSON(w, status, APIResponse{
		Success: false,
		Error:   msg,
	})
}
