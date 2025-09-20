package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/quic-go/quic-go/http3"
	"github.com/spf13/cobra"
	"github.com/tus/tusd/v2/pkg/filelocker"
	"github.com/tus/tusd/v2/pkg/filestore"
	tusd "github.com/tus/tusd/v2/pkg/handler"
)

//go:embed ui/dist/*
var webUIFiles embed.FS
var webUIFS, _ = fs.Sub(webUIFiles, "ui/dist")

var (
	port       int
	uploadsDir string
	certFile   string
	keyFile    string
)

var rootCmd = &cobra.Command{
	Use:   "simple-upload",
	Short: "A simple file upload server using TUS protocol",
	Long: `Simple Upload is a web-based file upload server that uses the TUS protocol
for resumable file uploads. It includes a web UI for easy file management.`,
	Run: runServer,
}

func init() {
	rootCmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to listen on")
	rootCmd.Flags().StringVarP(&uploadsDir, "uploads-dir", "d", "./uploads", "Directory to store uploaded files")
	rootCmd.Flags().StringVarP(&certFile, "cert", "c", "", "Path to TLS certificate file (enables HTTPS and HTTP/3)")
	rootCmd.Flags().StringVarP(&keyFile, "key", "k", "", "Path to TLS private key file (enables HTTPS and HTTP/3)")
}

// altSvcMiddleware adds Alt-Svc header to advertise HTTP/3 availability
func altSvcMiddleware(next http.Handler, port int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor < 3 {
			// Add Alt-Svc header to advertise HTTP/3 on the same port
			w.Header().Set("Alt-Svc", fmt.Sprintf(`h3=":%d"; ma=300`, port))
		}
		next.ServeHTTP(w, r)
	})
}

// sanitizeFilename removes or replaces unsafe characters in filenames
func sanitizeFilename(filename string) string {
	// Replace path separators and other potentially dangerous characters
	unsafe := []string{"/", "\\", "..", ":", "*", "?", "\"", "<", ">", "|"}
	sanitized := filename

	for _, char := range unsafe {
		sanitized = strings.ReplaceAll(sanitized, char, "_")
	}

	// Remove leading/trailing spaces and dots
	sanitized = strings.Trim(sanitized, " .")

	// If filename becomes empty after sanitization, return a default
	if sanitized == "" {
		return "unkown-file"
	}

	return sanitized
}

// getUniqueFilename ensures the filename is unique in the target directory
func getUniqueFilename(dir, filename string) string {
	sanitized := sanitizeFilename(filename)
	targetPath := filepath.Join(dir, sanitized)

	// If file doesn't exist, use the sanitized filename
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return sanitized
	}

	// File exists, add a counter
	ext := filepath.Ext(sanitized)
	base := strings.TrimSuffix(sanitized, ext)

	for i := 1; ; i++ {
		newFilename := fmt.Sprintf("%s_%d%s", base, i, ext)
		newPath := filepath.Join(dir, newFilename)

		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newFilename
		}
	}
}

func handleCompletedUploads(handler *tusd.Handler) {
	go func() {
		for {
			event := <-handler.CompleteUploads

			originalFilename := event.Upload.MetaData["filename"]
			uploadID := event.Upload.ID

			slog.Info("Upload finished",
				"upload_id", uploadID,
				"filename", originalFilename)

			if originalFilename == "" {
				slog.Warn("No filename in metadata, keeping file with upload ID",
					"upload_id", uploadID)
				continue
			}

			oldPath := filepath.Join(uploadsDir, uploadID)

			finalFilename := getUniqueFilename(uploadsDir, originalFilename)
			newPath := filepath.Join(uploadsDir, finalFilename)

			// Check if the file with the upload ID exists
			if _, err := os.Stat(oldPath); err != nil {
				slog.Warn("Upload file not found for renaming",
					"upload_id", uploadID,
					"filename", originalFilename,
					"path", oldPath)
				continue
			}

			if err := os.Rename(oldPath, newPath); err != nil {
				slog.Error("Failed to rename uploaded file",
					"upload_id", uploadID,
					"original_filename", originalFilename,
					"final_filename", finalFilename,
					"error", err)
				continue
			}
			slog.Info("File renamed successfully",
				"from", uploadID,
				"original_filename", originalFilename,
				"final_filename", finalFilename)
		}
	}()
}

func runServer(cmd *cobra.Command, args []string) {
	// Create uploads directory if it doesn't exist
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		slog.Error("unable to create uploads directory", "error", err)
		os.Exit(1)
	}

	store := filestore.New(uploadsDir)
	locker := filelocker.New(uploadsDir)

	composer := tusd.NewStoreComposer()
	store.UseIn(composer)
	locker.UseIn(composer)

	handler, err := tusd.NewHandler(tusd.Config{
		BasePath:              "/files/",
		StoreComposer:         composer,
		NotifyCompleteUploads: true,
	})
	if err != nil {
		slog.Error("unable to create handler", "error", err)
		os.Exit(1)
	}

	handleCompletedUploads(handler)

	http.Handle("/files/", http.StripPrefix("/files/", handler))
	http.Handle("/files", http.StripPrefix("/files", handler))
	http.Handle("/", http.FileServer(http.FS(webUIFS)))

	addr := fmt.Sprintf(":%d", port)

	// Create HTTP server
	var server *http.Server

	// Determine if we should use HTTPS or HTTP
	if certFile != "" && keyFile != "" {
		// Always enable HTTP/3 when TLS is configured
		slog.Info("Starting HTTPS server with HTTP/3 support", "addr", addr)
		slog.Info("Configuration", "uploads_dir", uploadsDir, "cert_file", certFile, "key_file", keyFile, "http3", true)

		// Create HTTP server with Alt-Svc middleware to advertise HTTP/3
		server = &http.Server{
			Addr:    addr,
			Handler: altSvcMiddleware(http.DefaultServeMux, port),
		}

		// Start HTTP/3 server
		h3Server := &http3.Server{
			Addr:    addr,
			Handler: http.DefaultServeMux, // HTTP/3 server uses the original mux without Alt-Svc header
		}

		// Start HTTP/3 server in a goroutine
		go func() {
			if err := h3Server.ListenAndServeTLS(certFile, keyFile); err != nil {
				slog.Error("HTTP/3 server failed", "error", err)
			}
		}()

		// Start HTTP/1.1 and HTTP/2 server (for fallback)
		err = server.ListenAndServeTLS(certFile, keyFile)
	} else {
		// Create HTTP server without Alt-Svc middleware
		server = &http.Server{
			Addr:    addr,
			Handler: http.DefaultServeMux,
		}

		slog.Info("Starting HTTP server", "addr", addr)
		slog.Info("Configuration", "uploads_dir", uploadsDir)
		if certFile != "" || keyFile != "" {
			slog.Warn("Both --cert and --key must be provided for HTTPS")
		}
		err = server.ListenAndServe()
	}

	if err != nil {
		slog.Error("unable to listen", "error", err)
		os.Exit(1)
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
