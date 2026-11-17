// SentryWalk scan server.
//
// Receives NFC checkpoint scans from the phone app and writes them as an
// events file E in the same format the Java checker reads:
//
//	happened,arrived,guard,tag_id
//	2026-10-20T01:02:00,2026-10-20T01:02:01,G1,04A19F22
//
// So the Java side runs unchanged on live data:
//
//	java -cp out Main inputs/R_route.txt inputs/T_tags.csv data/E_live.csv
//
// Run (from the server folder):
//
//	go run .
//
// Only the Go standard library is used.
package main

import (
	"flag"
	"log"
	"net/http"
)

func main() {
	addr := flag.String("addr", ":8000", "address to listen on (\":8000\" = every network interface, port 8000)")
	tags := flag.String("tags", "../inputs/T_tags.csv", "tag list T (tag_id,checkpoint), shared with the Java checker")
	events := flag.String("events", "../data/E_live.csv", "live events file E written for the Java checker")
	scanLog := flag.String("log", "../data/scans.jsonl", "full detail of every scan, one JSON object per line")
	flag.Parse()

	srv, err := NewServer(*tags, *events, *scanLog)
	if err != nil {
		log.Fatalf("starting server: %v", err)
	}

	log.Printf("SentryWalk server listening on %s", *addr)
	log.Printf("  tags:   %s", *tags)
	log.Printf("  events: %s", *events)
	log.Fatal(http.ListenAndServe(*addr, srv.Routes()))
}
