package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	qbt "github.com/NullpointerW/go-qbittorrent-apiv2"
)

func main() {
	// Variables that'll be command line arguments
	var hostname string // IP or FQDN of the qBittorrent Web API server
	var port string     // Port number of the qBittorent Web API
	var username string // Username for qBittorrent
	var password string // Password for qBittorrent
	var filedir string  // Directory in the filesystem where the torrents live
	var qbtdir string   // Directory path as qBittorrent sees it (for Docker/container path mapping)

	flag.StringVar(&hostname, "hostname", "127.0.0.1", "IP or FQDN of the qBittorrent server")
	flag.StringVar(&port, "port", "8080", "Port the qBittorrent Web API is running on")
	flag.StringVar(&username, "username", "", "Username for qBittorrent")
	flag.StringVar(&password, "password", "", "Password for qBittorrent")
	flag.StringVar(&filedir, "directory", "/var/torrents/", "Directory in the filesystem where the torrents live")
	flag.StringVar(&qbtdir, "qbt-path", "", "Directory path as qBittorrent sees it (leave empty if same as --directory)")
	dryrun := flag.Bool("dryrun", false, "Don't actually delete files, just print what would happen")
	verbose := flag.Bool("verbose", false, "Show detailed information about protected files")
	flag.Parse()

	// Clean and validate the directory paths
	filedir = filepath.Clean(filedir)
	if filedir == "." || filedir == "/" {
		fmt.Println("ERROR: Invalid directory specified")
		return
	}

	// If qbt-path is not specified, assume it's the same as the local directory
	if qbtdir == "" {
		qbtdir = filedir
	} else {
		qbtdir = filepath.Clean(qbtdir)
	}

	// Create a new qBitTorrent Web API client
	client, err := qbt.NewCli("http://"+hostname+":"+port, username, password)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Get the list of completed torrents
	torrents, err := client.TorrentList(qbt.Optional{"filter": "completed"})
	if err != nil {
		fmt.Println(err)
		return
	}

	// Build a map of protected items (files/directories that qBittorrent is managing)
	// Only include torrents that are actually saved in our target directory
	protectedItems := make(map[string]bool)

	if *verbose {
		fmt.Printf("Local filesystem directory: %s\n", filedir)
		fmt.Printf("qBittorrent's view of directory: %s\n", qbtdir)
		fmt.Printf("Found %d completed torrents\n\n", len(torrents))
	}

	for _, torrent := range torrents {
		// Get the directory where this torrent is saved (as qBittorrent sees it)
		contentPath := filepath.Clean(torrent.ContentPath)
		torrentDir := filepath.Dir(contentPath)

		if *verbose {
			fmt.Printf("Torrent: %s\n", torrent.Name)
			fmt.Printf("  ContentPath: %s\n", contentPath)
			fmt.Printf("  Parent Dir: %s\n", torrentDir)
		}

		var itemToProtect string

		// Determine what filesystem item to protect
		if torrentDir == qbtdir {
			// Single-file torrent: file is directly in the target directory
			// Protect the filename
			itemToProtect = filepath.Base(contentPath)
			if *verbose {
				fmt.Printf("  Type: Single-file torrent\n")
				fmt.Printf("  Protecting file: %s\n", itemToProtect)
			}
		} else if filepath.Dir(torrentDir) == qbtdir {
			// Multi-file torrent: file is in a subdirectory of the target directory
			// Protect the directory name
			itemToProtect = filepath.Base(torrentDir)
			if *verbose {
				fmt.Printf("  Type: Multi-file torrent\n")
				fmt.Printf("  Protecting directory: %s\n", itemToProtect)
			}
		} else {
			// Torrent is not in our target directory
			if *verbose {
				fmt.Printf("  ✗ SKIPPING (not in target directory)\n")
				fmt.Println()
			}
			continue
		}

		// Add to protected items
		protectedItems[itemToProtect] = true

		if *verbose {
			fmt.Printf("  ✓ PROTECTED\n")
			fmt.Println()
		}
	}

	if *verbose {
		fmt.Printf("\nTotal protected items in this directory: %d\n", len(protectedItems))
		fmt.Printf("Now scanning filesystem at: %s\n\n", filedir)
	}

	// Iterate over all files in a directory
	files, err := os.ReadDir(filedir)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, file := range files {
		fileName := file.Name()

		// Check if this item is protected by qBittorrent
		if protectedItems[fileName] {
			if *verbose {
				fmt.Printf("Keeping (in use by qBittorrent): %s\n", fileName)
			}
			continue
		}

		// Not protected - this file/directory is not managed by qBittorrent
		targetPath := filepath.Join(filedir, fileName)

		// Verify the path is still within the target directory (prevents path traversal)
		if !filepath.HasPrefix(targetPath, filedir) {
			fmt.Printf("ERROR: Suspicious path detected: %s\n", targetPath)
			continue
		}

		if *dryrun {
			fmt.Printf("Would delete: %s\n", targetPath)
		} else {
			fmt.Printf("Deleting: %s\n", targetPath)
			err := os.RemoveAll(targetPath)
			if err != nil {
				fmt.Printf("ERROR deleting %s: %v\n", targetPath, err)
				continue
			}
		}
	}
}
