package bhyve

import (
	"context"
	"fmt"
	"log"
	"os/exec"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
)

type stepCreateZvol struct {
	name string
}

func (step *stepCreateZvol) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)

	zvol_path := fmt.Sprintf("%s/packer0", config.ZPool)

	args := []string{
		"create",
		"-V", config.DiskSize,
		zvol_path,
	}

	ui.Say(fmt.Sprintf("Creating ZFS zvol %s", zvol_path))

	cmd := exec.Command("/usr/sbin/zfs", args...)
	if err := cmd.Run(); err != nil {
		err = fmt.Errorf("Error creating zvol: %s", err)
		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

func (step *stepCreateZvol) Cleanup(state multistep.StateBag) {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)

	zvol_path := fmt.Sprintf("%s/packer0", config.ZPool)

	args := []string{
		"destroy",
		zvol_path,
	}

	ui.Say(fmt.Sprintf("Destroying ZFS zvol %s", zvol_path))

	cmd := exec.Command("/usr/sbin/zfs", args...)
	if err := cmd.Run(); err != nil {
		log.Fatal("Error destroying zvol: %s", err)
	}
}
