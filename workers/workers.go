package workers

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"
)

func StartChildren(isChild bool, workerCount int, app string, adapter string, port int, shards int) {
	if !isChild {
		startGoWorkers(workerCount)
	}
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
		fmt.Sprintf("--child %v", isChild),
		fmt.Sprintf("--app %v", app),
		fmt.Sprintf("--adapter %v", adapter),
		fmt.Sprintf("--port %v", port),
		fmt.Sprintf("--shards %v", shards),
	)
	child.Stderr = os.Stderr
	time.Sleep(1 * time.Second)
	err := child.Start()
	if err != nil {
		log.Fatal(err)
	}

	go logPython(child)
	return *child
}

func logPython(child *exec.Cmd) {
	stdout, err := child.StdoutPipe()
	if err != nil {
		log.Fatalln("Fatal pipe error connecting to python worker.")
	}
	reader := bufio.NewReader(stdout)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalln("Failed to read stdout of python worker.")
		}
		log.Println(msg)
	}
}

type ChildResult struct {
	pid int
	err error
}

func startGoWorkers(workerCount int) {
	activeProcesses := make(map[int]*exec.Cmd)
	finishedCallback := make(chan ChildResult) // The worker callback for when a process exits.

	defer func() {
		for _, proc := range activeProcesses {
			_ = proc.Process.Kill()
		}
	}()

	startProcesses(workerCount, finishedCallback, activeProcesses)

	lastUpdated, failedProcesses := time.Now(), 0
	for incomingSig := range finishedCallback {
		delete(activeProcesses, incomingSig.pid)

		if lastUpdated.Sub(time.Now()).Minutes() > 5 {
			failedProcesses = 1
		} else {
			failedProcesses += 1
		}

		log.Printf("A child has exited with error: %v."+
			" Restarting process now.\n", incomingSig.err)

		if failedProcesses >= 5 {
			log.Fatalln("Process has exited too many times")
		}

		startProcesses(workerCount, finishedCallback, activeProcesses)
	}
	return
}

func startProcesses(workerCount int, callback chan ChildResult, activeProcesses map[int]*exec.Cmd) {
	for i := 0; i < workerCount; i++ {
		proc, err := forkCurrent()
		if err != nil {
			log.Fatalf("Failed to start forks. %v\n", err)
		}
		activeProcesses[proc.Process.Pid] = proc
		go func() {
			callback <- ChildResult{proc.Process.Pid, proc.Wait()}
		}()
	}
}

func forkCurrent() (*exec.Cmd, error) {
	fmt.Printf("%v\n", os.Args)
	cmd := exec.Command(os.Args[0], "--child true", "--prefork-child true")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd, cmd.Start()
}
