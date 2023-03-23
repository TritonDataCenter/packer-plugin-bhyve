package bhyve

import (
	"context"
	"fmt"
	"log"
	"os/exec"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
)

type stepBhyve struct {
	name string
}

func (step *stepBhyve) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)

	disk_args := fmt.Sprintf("1,nvme,/dev/zvol/rdsk/%s/packer0", config.ZPool)
	cd_args := fmt.Sprintf("2,ahci-cd,%s", state.Get("iso_path").(string))
	vnc_args := fmt.Sprintf("29,fbuf,vga=off,rfb=%s:%d,password=%s",
		config.VNCBindAddress,
		state.Get("vnc_port").(int),
		state.Get("vnc_password").(string))

	args := []string{
		"-D",
		"-H",
		"-c", "1",
		"-l", "bootrom,/usr/share/bhyve/uefi-rom.bin",
		"-m", "1024",
		"-s", "0,hostbridge,model=i440fx",
		"-s", disk_args,
		"-s", cd_args,
		"-s", "5,virtio-net-viona,vnic=packer0",
		"-s", vnc_args,
		"-s", "30,xhci,tablet",
		"-s", "31,lpc",
		step.name,
	}

	ui.Say(fmt.Sprintf("Starting bhyve VM %s", step.name))

	cmd := exec.Command("/usr/sbin/bhyve", args...)
	if err := cmd.Start(); err != nil {
		err = fmt.Errorf("Error starting VM: %s", err)
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
