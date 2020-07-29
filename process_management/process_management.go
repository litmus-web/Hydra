package process_management

import (
	"log"
	"os"
	"os/exec"
	"sync"
	"time"
)

type ActiveWorkers struct {
	workers []exec.Cmd
}

var activeProcesses = ActiveWorkers{}
var mutex = sync.Mutex{}
var newProcesses = ActiveWorkers{}

type TargetApp struct {
	Target string
	App    string
	Id     string
	Auth   string
}

func StartWorkers(amountProcesses int, target TargetApp) {
	println("Waiting for server to finish loading before booting workers")
	time.Sleep(500 * time.Millisecond)
	println("Starting Workers...")
	for i := 0; i <= amountProcesses; i++ {
		spawnedWorker := startWorker(target)
		go handleNewWorker(spawnedWorker)
	}

	println("Starting worker watcher")
	for {
		newProcesses := ActiveWorkers{}
		for i, worker := range activeProcesses.workers {
			if worker.ProcessState.Exited() {
				log.Println(
					"Worker ", i,
					" has exited with code: ", worker.ProcessState.ExitCode(),
					" attempting restart.")

				spawnedWorker := startWorker(target)
				go handleRestartWorker(spawnedWorker)
			} else {
				newProcesses.workers = append(newProcesses.workers, worker)
			}
		}
		activeProcesses = newProcesses
		time.Sleep(5 * time.Second)
	}
}

func startWorker(target TargetApp) *exec.Cmd {
	spawnedWorker := exec.Command("python", target.Target, target.App, target.Id, target.Auth)
	spawnedWorker.Stdout = os.Stdout
	spawnedWorker.Stderr = os.Stdout
	return spawnedWorker
}

func handleNewWorker(worker *exec.Cmd) {
	err := worker.Start()
	if err != nil {
		log.Fatal("Error starting Worker\nError Message: ", err)
	} else {
		time.Sleep(200 * time.Millisecond)
		println("Started worker with Pid: ", worker.Process.Pid)
		mutex.Lock()
		activeProcesses.workers = append(activeProcesses.workers, *worker)
		mutex.Unlock()
	}
}

func handleRestartWorker(worker *exec.Cmd) {
	err := worker.Start()
	if err != nil {
		panic("Unable to start worker process")
	} else {
		time.Sleep(200 * time.Millisecond)
		println("Started worker with Pid: ", worker.Process.Pid)
		mutex.Lock()
		newProcesses.workers = append(newProcesses.workers, *worker)
		mutex.Unlock()
	}
}
