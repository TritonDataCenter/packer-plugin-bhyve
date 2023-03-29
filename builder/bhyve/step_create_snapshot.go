package bhyve

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
)

type stepCreateSnapshot struct{}

func (step *stepCreateSnapshot) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)

	var stderr bytes.Buffer
	snap_path := fmt.Sprintf("%s/packer0@final", config.ZPool)
	file_path := filepath.Join(config.OutputDir, config.VMName)

	ui.Say(fmt.Sprintf("Creating ZFS snapshot %s", snap_path))
	args := []string{
		"snapshot",
		snap_path,
	}
	cmd := exec.Command("/usr/sbin/zfs", args...)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		err := fmt.Errorf("Error creating snapshot: %s", strings.TrimSpace(stderr.String()))
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	ui.Say(fmt.Sprintf("Sending snapshot %s to %s", snap_path, file_path))
	args = []string{
		"send",
		snap_path,
	}
	outfile, err := os.Create(file_path)
	if err != nil {
		err = fmt.Errorf("Error creating hard drive in output dir: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}
	defer outfile.Close()
	cmd = exec.Command("/usr/sbin/zfs", args...)
	cmd.Stdout = outfile
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		err := fmt.Errorf("Error sending snapshot: %s", strings.TrimSpace(stderr.String()))
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	ui.Say(fmt.Sprintf("Deleting ZFS snapshot %s", snap_path))
	args = []string{
		"destroy",
		snap_path,
	}
	cmd = exec.Command("/usr/sbin/zfs", args...)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		err := fmt.Errorf("Error deleting snapshot: %s", strings.TrimSpace(stderr.String()))
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

func (step *stepCreateSnapshot) Cleanup(state multistep.StateBag) {}
