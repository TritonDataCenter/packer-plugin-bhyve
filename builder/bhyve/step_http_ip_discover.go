package bhyve

import (
	"context"
	"fmt"
	"net"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
)

// Mostly taken from packer-plugin-qemu's step_http_ip_discover but modified
// to get the IP address from the host NIC interface that will be the parent
// device of our VNIC.
type stepHTTPIPDiscover struct{}

func (s *stepHTTPIPDiscover) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)

	hostIP := ""

	nic, err := net.InterfaceByName(config.HostNIC)
	if err != nil {
		err := fmt.Errorf("Error getting the %s interface: %s", config.HostNIC, err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}
	addrs, err := nic.Addrs()
	if err != nil {
		err := fmt.Errorf("Error getting the %s interface addresses: %s", config.HostNIC, err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		if ip == nil {
			continue
		}
		ip = ip.To4()
		if ip == nil {
			continue
		}
		hostIP = ip.String()
		break
	}
	if hostIP == "" {
		err := fmt.Errorf("Error getting an IPv4 address from %s: cannot find any IPv4 address", config.HostNIC)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	ui.Say(fmt.Sprintf("Discovered Host IP address %s", hostIP))
	state.Put("http_ip", hostIP)

	return multistep.ActionContinue
}

func (s *stepHTTPIPDiscover) Cleanup(state multistep.StateBag) {}
