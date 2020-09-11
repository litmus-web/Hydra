package main

import (
	"flag"
	"log"
	"net"

	"./prefork"
	"./server"
)

// Flags
var (
	host = flag.String(
		"host", "127.0.0.1:8080", "The address for the server to bind to.")
	app = flag.String(
		"app", "", "The WSGI, ASGI or raw app name and path, e.g 'my_server:app'")
	adapter = flag.String(
		"adapter", "", "Adapter type to use. (ASGI, WSGI, RAW)")
	//shardsPerProc = flag.Int(
	//	"shardsperproc", 1, "The amount of shards per process to use.")
	workerCount = flag.Int(
		"workers", 1, "The amount of server workers to spawn.")

	// Fast Http Settings
	//name = flag.String(
	//"name", "Sandman", "Server name")
	//concurrency = flag.Int(
	//"concurrency", 0, "Set the concurrency limit.")
	//disableKeepAlive = flag.Bool(
	//	"nokeepalive", false, "Disable keep-alive.")
	//maxConnsPerIp = flag.Int(
	//	"maxconnperip",
	//	0,
	//	"The maximum number of connections allowed by server per ip.")
	//maxReqPerConn = flag.Int(
	//	"maxreqperconn",
	//	0,
	//	"The maximum number of requests allowed per connection.")
	//tcpKeepAlive = flag.Bool(
	//	"tcpkeepalive",
	//	false,
	//	"Enable/Disable TCP keep alive")
	//reduceMemory = flag.Bool(
	//	"reducememory",
	//	false,
	//	"Start the server in reduce memory mode, this will " +
	//		"try to minimise the amount of memory used. (This may hinder performance)")
	//maxRequestBodySize = flag.Int(
	//	"maxreqsize", 0, "The maximum body size allowed in a request.")
)

func main() {
	flag.Parse()

	if *app == "" {
		log.Fatalln("--app is a required flag, e.g 'myfile:app'")
	} else if *adapter == "" {
		log.Fatalln("--adapter is a required flag, e.g 'asgi'")
	}

	free, err := getFreePort()
	if err != nil {
		log.Fatalln(err)
	}
	startServers(*host, free, *workerCount)
}

// Starts the main servers, it will only start worker servers if
// the process is a child because the main thread is used for
// process management and does not connect to a socket.
func startServers(host string, freePort int, workerCount int) {
	if prefork.IsChild() {
		go server.StartWorkerServer(freePort)
	}
	server.StartMainServer(host, workerCount)
}

// Produces a free port from polling the OS.
func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
