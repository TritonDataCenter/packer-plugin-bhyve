package bhyve

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
)

type stepCreateVNIC struct {
	name    string
	created bool
}

func (step *stepCreateVNIC) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)

	if !config.hasNIC() {
		ui.Say("No NIC will be attached because host_nic was not set")
		return multistep.ActionContinue
	}

	args := []string{
		"create-vnic",
		"-t",
		"-l", config.VNICLink,
		config.VNICName,
	}

	ui.Say(fmt.Sprintf("Creating VNIC %s on link %s", config.VNICName, config.VNICLink))

	cmd := exec.Command("/usr/sbin/dladm", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		err := fmt.Errorf("Error creating VNIC: %s", strings.TrimSpace(stderr.String()))
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	step.created = true
	return multistep.ActionContinue
}

func (step *stepCreateVNIC) Cleanup(state multistep.StateBag) {
	if !step.created {
		return
	}

	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)

	args := []string{
		"delete-vnic",
		config.VNICName,
	}

	ui.Say(fmt.Sprintf("Deleting VNIC %s from link %s", config.VNICName, config.VNICLink))

	// Despite bhyvectl --destroy running before us, this will often fail
	// with EBUSY for a few seconds afterwards, so we retry a few times.
	var retries = 4
	for i := 1; i <= retries; i++ {
		cmd := exec.Command("/usr/sbin/dladm", args...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			if i == retries {
				log.Printf("Error deleting VNIC: %s", strings.TrimSpace(stderr.String()))
				break
			}
			time.Sleep(5 * time.Second)
			continue
		}
		break
	}
}
