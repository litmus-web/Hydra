package process_manager

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"time"
)

var ErrOverRecovery = errors.New("exceeding the value of RecoverThreshold")

type ExternalWorkers struct {
	// The thing to execute code, this lets us customise calls, e.g. py xyz.py
	RunnerCall string

	TargetFile string

	App            string // the file:app string
	Adapter        string // The python adapter for frameworks
	ConnectionPort int    // The unique port assigned

	WorkerCount   int // Amount of actual processes
	ShardsPerProc int // Amount of ws connections

	// The secure string to allow the worker to actually connect
	WorkerAuth string

	// The amount of times a process is able to restart from
	recoveryAllowance int
}

func (ew *ExternalWorkers) StartExternalWorkers() {
	var err error

	ew.recoveryAllowance = 2 * ew.WorkerCount

	type workerSig struct {
		pid int
		err error
	}

	sigCh := make(chan workerSig, ew.WorkerCount)
	activeProcs := make(map[int]*exec.Cmd)

	defer func() {
		for _, proc := range activeProcs {
			_ = proc.Process.Kill()
		}
	}()

	for i := 0; i < ew.WorkerCount; i++ {
		var cmd *exec.Cmd

		if cmd, err = ew.doCommand(); err != nil {
			log.Printf("failed to start a worker process, error: %v\n", err)
			return
		}

		activeProcs[cmd.Process.Pid] = cmd
		go func() {
			sigCh <- workerSig{cmd.Process.Pid, cmd.Wait()}
		}()

		time.Sleep(500 * time.Millisecond)
	}

	sig := <-sigCh
	delete(activeProcs, sig.pid)

	log.Printf(
		"one of the worker processes exited with error: %v", sig.err)
	return
}

func (ew *ExternalWorkers) doCommand() (*exec.Cmd, error) {
	proc := exec.Command(
		ew.RunnerCall,
		formatArgs(
			toString("", fmt.Sprintf("%s.py", ew.TargetFile)),
			toString("--app", ew.App),
			toString("--adapter", ew.Adapter),
			toString("--port", ew.ConnectionPort),
			toString("--shards", ew.ShardsPerProc),
			toString("--auth", ew.WorkerAuth),
		)...,
	)
	proc.Stdout = os.Stdout

	stderrReader, err := proc.StderrPipe()
	if err != nil {
		return nil, err
	}

	err = proc.Start()
	if err != nil {
		return nil, err
	}

	go ew.logPipes(stderrReader)

	return proc, nil
}

func (ew *ExternalWorkers) logPipes(r io.ReadCloser) {
	defer r.Close()

	reader := bufio.NewReader(r)

	var line []byte
	var err error

	for {
		line, _, err = reader.ReadLine()
		if err != nil {
			return
		}

		log.Println(string(line))
	}
}

func formatArgs(flagPairs ...[]string) []string {
	var fullSet []string

	for _, pair := range flagPairs {
		fullSet = append(fullSet, pair...)
	}

	return fullSet
}

func toString(flag string, v interface{}) []string {
	var temp []string
	return append(temp, flag, fmt.Sprintf("%v", v))
}
