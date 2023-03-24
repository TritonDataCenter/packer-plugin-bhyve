package bhyve

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"sync"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
)

type stepBhyve struct {
	name    string
	vmEndCh <-chan int
	lock    sync.Mutex
}

func (step *stepBhyve) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	driver := state.Get("driver").(Driver)
	ui := state.Get("ui").(packer.Ui)

	if err := driver.Start(); err != nil {
		err := fmt.Errorf("Error launching VM: %s", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

func (step *stepBhyve) Cleanup(state multistep.StateBag) {
	ui := state.Get("ui").(packer.Ui)

	vmarg := fmt.Sprintf("--vm=%s", step.name)

	args := []string{
		vmarg,
		"--destroy",
	}

	ui.Say(fmt.Sprintf("Stopping bhyve VM %s", step.name))

	cmd := exec.Command("/usr/sbin/bhyvectl", args...)
	if err := cmd.Run(); err != nil {
		log.Printf("Error stopping VM: %s", err)
	}
}
