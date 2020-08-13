package main

import (
	"flag"

	"./server"
)

// Flags
var (
	host = flag.String(
		"host",
		"0.0.0.0:8000",
		"Host addr")
	workerPort = flag.Int("worker_port", 1234, "Worker port target")
)

func main()  {
	flag.Parse()

	server.StartServers(*host, *workerPort)
}