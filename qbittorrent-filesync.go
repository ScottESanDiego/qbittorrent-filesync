package main

import (
	"os"
	"fmt"
	"flag"
	"sort"
	"slices"
	qbt "github.com/NullpointerW/go-qbittorrent-apiv2"
)

func main() {
	// Variables that'll be command line arguments
	var hostname string	// IP or FQDN of the qBittorrent Web API server
	var port string		// Port number of the qBittorent Web API
	var username string	// Username for qBittorrent
	var password string	// Password for qBittorrent
	var filedir string	// Directory in the filesystem where the torrents live

	flag.StringVar(&hostname, "hostname", "127.0.0.1", "IP or FQDN of the qBittorrent server")
	flag.StringVar(&port, "port", "8080", "Port the qBittorrent Web API is running on")
	flag.StringVar(&username, "username", "", "Username for qBittorrent")
	flag.StringVar(&password, "password", "", "Password for qBittorrent")
	flag.StringVar(&filedir, "directory", "/var/torrents/", "Directory in the filesystem where the torrents live with trailing slash")
	dryrun := flag.Bool("dryrun", false, "Don't actually delete files, just print what would happen")
	flag.Parse()

	// Create a new qBitTorrent Web API client
	client, err := qbt.NewCli("http://"+hostname+":"+port, username, password)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Get the list of completed torrents
	torrents, err := client.TorrentList(qbt.Optional{"filter":"completed"})
	if err != nil {
		fmt.Println(err)
		return
	}

	// Build a slice of torrent names from the list of torrents
	var torrentnames  []string
	for _, torrent := range torrents {
		torrentnames = append(torrentnames, torrent.ContentPath[len(torrent.SavePath)+1:])
	}

	// Sort slice so that we can search it later
	sort.Strings(torrentnames)

	// Iterate over all files in a directory
	files, err := os.ReadDir(filedir)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, file := range files {
		_, found := slices.BinarySearch(torrentnames, file.Name())
		if !found {
			// Super-duper extra check that file.Name() isn't null since os.RemoveAll will remove filedir otherwise!
			if len(file.Name()) > 0 {
				if *dryrun == true {
					fmt.Printf("Dry-run, not deleting: ")
				} else {
					fmt.Printf("Deleting unowned file: ")
					err := os.RemoveAll(filedir+file.Name())
					if err != nil {
						fmt.Println(err)
						return
					}
			}
			} else {
				fmt.Println("ERROR: Filename was empty!")
			}

			fmt.Printf("%s\n", filedir+file.Name())
		}
	}
}

