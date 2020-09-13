package main

import (
	"flag"
	"log"
	"net"
	"strings"

	"./prefork"
	"./process_manager"
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
	workerCount = flag.Int(
		"workers", 1, "The amount of server workers to spawn.")

	// External process options
	workerCommand = flag.String(
		"workercmd",
		"python",
		"The command to be used to start a external worker. e.g. 'python'")

	shardsPerProc = flag.Int(
		"shardsperproc", 1, "The amount of shards per process to use.")

	processRatio = flag.Int(
		"extprocs", 1, "The amount of external workers to spawn to 1 Go worker.")

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

	var targetFile string
	splitString := strings.Split(*app, ":")
	if len(splitString) == 2 {
		targetFile = splitString[0]
	} else {
		log.Printf(
			"Cannot split %v into file and object parts, "+
				"make sure the format is `file:object` e.g. `my_file:app`", *app)
		return
	}

	free, err := getFreePort()
	if err != nil {
		log.Fatalln(err)
	}

	manager := process_manager.ExternalWorkers{
		RunnerCall:     *workerCommand,
		TargetFile:     targetFile,
		App:            *app,
		Adapter:        *adapter,
		ConnectionPort: free,
		WorkerCount:    *processRatio,
		ShardsPerProc:  *shardsPerProc,
		WorkerAuth:     "",
	}

	startServers(*host, *workerCount, free, manager)
}

// Starts the main servers, it will only start worker servers if
// the process is a child because the main thread is used for
// process management and does not connect to a socket.
func startServers(host string, workerCount int, freePort int, workerManager process_manager.ExternalWorkers) {
	if prefork.IsChild() {
		ended := make(chan error)

		go func() {
			err := server.StartWorkerServer(freePort, workerManager)
			ended <- err
		}()

		go func() {
			server.StartMainServer(host, workerCount)
			ended <- nil
		}()

		err := <-ended

		if err != nil {
			log.Fatalln(err)
		}
		log.Println("Shutting down server...")

	} else {
		server.StartMainServer(host, workerCount)
	}
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
