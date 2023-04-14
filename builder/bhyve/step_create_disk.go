package bhyve

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
)

type stepCreateDisk struct{}

func (step *stepCreateDisk) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)

	disk_path := filepath.Join(config.OutputDir, config.DiskName)

	args := []string{
		"-n", config.DiskSize,
		disk_path,
	}

	ui.Say(fmt.Sprintf("Creating disk image %s", disk_path))

	cmd := exec.Command("/usr/sbin/mkfile", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		err := fmt.Errorf("Error creating image: %s", strings.TrimSpace(stderr.String()))
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	state.Put("bhyve_disk_path", disk_path)

	return multistep.ActionContinue
}

func (step *stepCreateDisk) Cleanup(state multistep.StateBag) {}
