package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gliderlabs/ssh"
)

func main() {
	addr := flag.String("addr", ":2222", "listen address")
	dataDir := flag.String("data", "data", "where the bar keeps its books")
	flag.Parse()

	if err := os.MkdirAll(*dataDir, 0o755); err != nil {
		log.Fatal(err)
	}
	keyPath := filepath.Join(*dataDir, "hostkey")
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-f", keyPath, "-N", "", "-q")
		if out, err := cmd.CombinedOutput(); err != nil {
			log.Fatalf("could not cut a key for the front door: %v\n%s", err, out)
		}
	}

	room := NewRoom(*dataDir)
	go room.Tick()

	ssh.Handle(func(s ssh.Session) { room.Seat(s) })

	fmt.Printf("the loopback is open on %s\n", *addr)
	fmt.Printf("    ssh -p %s anyone@localhost\n", portOf(*addr))
	log.Fatal(ssh.ListenAndServe(*addr, nil, ssh.HostKeyFile(keyPath)))
}

func portOf(addr string) string {
	if i := strings.LastIndex(addr, ":"); i >= 0 {
		return addr[i+1:]
	}
	return addr
}
