package bhyve

import (
	"bufio"
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

// Based on packer-plugin-qemu's implementation, but modified to dig MAC
// address out of dladm and parse arp(8) output for matching IP address.
//
// This step waits for the guest address to become available on the network,
// then it sets the guestAddress state property.
type stepWaitGuestAddress struct {
	timeout time.Duration
}

func (s *stepWaitGuestAddress) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	// Firstly get the VNIC MAC address, as this should be immediately
	// available and not change.
	vnic_mac := get_vnic_mac(config.VNICName)
	if vnic_mac == "" {
		err := fmt.Errorf("Error getting VNIC MAC address")
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	ui.Say(fmt.Sprintf("Waiting for the guest address to become available..."))
	for {
		guestAddress := get_vnic_ip(vnic_mac)
		if guestAddress != "" {
			log.Printf("Found guest address %s", guestAddress)
			state.Put("guestAddress", guestAddress)
			return multistep.ActionContinue
		}
		select {
		case <-time.After(10 * time.Second):
			continue
		case <-ctx.Done():
			return multistep.ActionHalt
		}
	}
}

func (s *stepWaitGuestAddress) Cleanup(state multistep.StateBag) {
}

func get_vnic_mac(vnic string) string {
	// First up get the MAC address of the VNIC.  Unfortunately dladm(8)
	// strips leading zeros so we need to reinsert them to match against
	// arp(8) output.
	args := []string{
		"show-vnic",
		"-p",
		"-o", "macaddress",
		vnic,
	}
	cmd := exec.Command("/usr/sbin/dladm", args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Printf("Could not retrieve VNIC MAC address: %s", strings.TrimSpace(stderr.String()))
		return ""
	}

	vnic_mac := ""
	for _, item := range strings.Split(strings.TrimSpace(stdout.String()), ":") {
		if vnic_mac == "" {
			vnic_mac = fmt.Sprintf("%02s", item)
		} else {
			vnic_mac = fmt.Sprintf("%s:%02s", vnic_mac, item)
		}
	}

	return vnic_mac
}

func get_vnic_ip(vnic_mac string) string {
	args := []string{"-a", "-n"}

	cmd := exec.Command("/usr/sbin/arp", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return ""
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Could not run arp: %s", strings.TrimSpace(stderr.String()))
		return ""
	}

	s := bufio.NewScanner(stdout)
	// Move past the 3 header lines
	s.Scan()
	s.Scan()
	s.Scan()

	for s.Scan() {
		fields := strings.Fields(s.Text())
		// The "Flags" field does not always include any output, so
		// we need to match for both potential "Phys Addr" locations.
		switch len(fields) {
		case 4:
			if fields[3] == vnic_mac {
				return fields[1]
			}
		case 5:
			if fields[4] == vnic_mac {
				return fields[1]
			}
		default:
			log.Printf("XXX: Invalid line! '%s'", s.Text())
		}
	}

	cmd.Wait()
	return ""
}
