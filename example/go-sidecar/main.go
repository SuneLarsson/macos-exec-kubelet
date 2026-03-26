package main

import (
	"fmt"
	"log"
	"net"
	"os"

	nfs "github.com/willscott/go-nfs"
	nfshelper "github.com/willscott/go-nfs/helpers"
	"github.com/go-git/go-billy/v5/osfs"
)

func main() {
	// The directory to export, defaults to /nfsshare
	sharedDir := os.Getenv("SHARED_DIRECTORY")
	if sharedDir == "" {
		sharedDir = "/nfsshare"
	}

	// Port to listen on, defaults to 2049
	port := os.Getenv("NFS_PORT")
	if port == "" {
		port = "2049"
	}

	// Verify the directory exists
	info, err := os.Stat(sharedDir)
	if err != nil {
		log.Fatalf("Cannot access shared directory %s: %v", sharedDir, err)
	}
	if !info.IsDir() {
		log.Fatalf("%s is not a directory", sharedDir)
	}

	log.Printf("Exporting %s on port %s (NFS + Mount on same port)", sharedDir, port)

	// Create a billy filesystem pointing at the real directory
	fs := osfs.New(sharedDir)

	// Create NFS handler with null auth (allow all) and caching
	handler := nfshelper.NewNullAuthHandler(fs)
	cacheHelper := nfshelper.NewCachingHandler(handler, 1024)

	// Start listening
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}

	fmt.Printf("go-nfs server ready: exporting %s on %s\n", sharedDir, listener.Addr())

	// Serve NFS (this blocks forever)
	if err := nfs.Serve(listener, cacheHelper); err != nil {
		log.Fatalf("NFS server error: %v", err)
	}
}
