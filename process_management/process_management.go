package process_management

import (
	"fmt"
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
	Auth   string
}

type TargetWorker struct {
	Target string
	App    string
	Id     string
	Auth   string
}

func StartWorkers(amountProcesses int, target TargetApp) {
	log.Println("Waiting for server to finish loading before booting workers")
	time.Sleep(500 * time.Millisecond)
	log.Println("Starting Workers...")
	for i := 1; i <= amountProcesses; i++ {
		workerCfg := TargetWorker{
			Target: target.Target,
			App:    target.App,
			Id:     fmt.Sprintf("%v", i),
			Auth:   target.Auth,
		}
		spawnedWorker := startWorker(workerCfg)
		go handleNewWorker(spawnedWorker)
	}

	log.Println("Starting worker watcher")
	for {
		newProcesses := ActiveWorkers{}
		for i, worker := range activeProcesses.workers {
			if worker.ProcessState != nil {
				if worker.ProcessState.Exited() {
					log.Println(
						"Worker ", i,
						" has exited with code: ", worker.ProcessState.ExitCode(),
						" attempting restart.")

					workerCfg := TargetWorker{
						Target: target.Target,
						App:    target.App,
						Id:     fmt.Sprintf("%v", i),
						Auth:   target.Auth,
					}
					spawnedWorker := startWorker(workerCfg)
					go handleRestartWorker(spawnedWorker)
				}
			} else {
				newProcesses.workers = append(newProcesses.workers, worker)
			}
		}
		activeProcesses = newProcesses
		time.Sleep(5 * time.Second)
	}
}

func startWorker(target TargetWorker) *exec.Cmd {
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
		log.Println("Started worker with Pid: ", worker.Process.Pid)
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
		log.Println("Started worker with Pid: ", worker.Process.Pid)
		mutex.Lock()
		newProcesses.workers = append(newProcesses.workers, *worker)
		mutex.Unlock()
	}
}
