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

type stepCreateZvol struct{}

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
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		err := fmt.Errorf("Error creating zvol: %s", strings.TrimSpace(stderr.String()))
		state.Put("error", err)
		ui.Error(err.Error())
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

	// Despite bhyvectl --destroy running before us, this will often fail
	// with EBUSY for a few seconds afterwards, so we retry a few times.
	var retries = 4
	for i := 1; i <= retries; i++ {
		cmd := exec.Command("/usr/sbin/zfs", args...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			if i == retries {
				log.Printf("Error destroying zvol: %s", strings.TrimSpace(stderr.String()))
				break
			}
			time.Sleep(5 * time.Second)
			continue
		}
		break
	}
}
