package process_manager

import "os/exec"

type ExternalWorkers struct {
	WorkerCount   int
	ShardsPerProc int

	activeProcs []*exec.Cmd
}

func (ew *ExternalWorkers) StartExternalWorkers() {

}
