package workers

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"time"
)

func StartChildren(isChild bool, app string, adapter string, port int, shards int) {
	starts := 1
	for starts <= 5 {
		child := startPythonWorker(isChild, app, adapter, port, shards)
		_ = child.Wait()
		starts += 1
	}
	if starts >= 5 {
		log.Fatal("Python worker exited unexpectedly 5 times.")
	}
	return
}

func startPythonWorker(isChild bool, app string, adapter string, port int, shards int) exec.Cmd {
	child := exec.Command(
		"python",
		"./import_test.py",
		"--child", fmt.Sprintf("%v", isChild),
		"--app", app,
		"--adapter", adapter,
		"--port", fmt.Sprintf("%v", port),
		"--shards", fmt.Sprintf("%v", shards),
	)
	child.Stderr = os.Stderr
	stdout, err := child.StdoutPipe()
	if err != nil {
		log.Fatalln("Fatal pipe error connecting to python worker.")
	}
	time.Sleep(1 * time.Second)
	err = child.Start()
	if err != nil {
		log.Fatal(err)
	}

	time.Sleep(1 * time.Second)
	go logPython(stdout)
	return *child
}

func logPython(stdout io.ReadCloser) {
	reader := bufio.NewReader(stdout)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalln("Failed to read stdout of python worker.")
		}
		log.Println(msg)
	}
}

