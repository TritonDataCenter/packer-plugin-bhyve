package bhyve

import (
	"fmt"
	"log"
	"os/exec"
	"sync"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

type Driver interface {
	Start() error
	Stop() error
	WaitForShutdown(<-chan struct{}) bool
}

type BhyveDriver struct {
	config  *Config
	state   multistep.StateBag
	vmCmd   *exec.Cmd
	vmEndCh <-chan int
	lock    sync.Mutex
}

func (d *BhyveDriver) Start() error {
	// Use the same slots as pci_slot_t in
	// illumos-joyent usr/src/lib/brand/bhyve/zone/boot.c
	const (
		SlotHostBridge int = 0
		SlotCDROM          = 3
		SlotBootDisk       = 4
		SlotNIC            = 6
		SlotFBuf           = 30
		SlotLPC            = 31
	)

	common_args := []string{
		"-D",
		"-H",
		"-c", "1",
		"-l", "bootrom,/usr/share/bhyve/uefi-rom.bin",
		"-m", "1024",
		"-s", fmt.Sprintf("%d,hostbridge,model=i440fx", SlotHostBridge),
		"-s", fmt.Sprintf("%d,virtio-blk,/dev/zvol/rdsk/%s/packer0",
			SlotBootDisk, d.config.ZPool),
		"-s", fmt.Sprintf("%d,virtio-net-viona,vnic=packer0", SlotNIC),
		"-s", fmt.Sprintf("%d:0,fbuf,vga=off,rfb=%s:%d,password=%s",
			SlotFBuf, d.config.VNCBindAddress,
			d.state.Get("vnc_port").(int),
			d.state.Get("vnc_password").(string)),
		"-s", fmt.Sprintf("%d:1,xhci,tablet", SlotFBuf),
		"-s", fmt.Sprintf("%d,lpc", SlotLPC),
	}

	// Set up two argument lists, one for the initial install with the
	// CDROM attached, and one without so we boot from disk for the
	// post-install steps.
	var boot_args []string = append(
		common_args,
		"-s", fmt.Sprintf("%d,ahci-cd,%s", SlotCDROM,
			d.state.Get("iso_path").(string)),
		d.config.VMName,
	)
	var reboot_args []string = append(
		common_args,
		d.config.VMName,
	)

	d.lock.Lock()
	defer d.lock.Unlock()

	if d.vmCmd != nil {
		panic("Existing VM state found")
	}

	log.Printf("Starting bhyve VM %s", d.config.VMName)
	log.Printf("boot_args %v", boot_args)
	log.Printf("reboot_args %v", reboot_args)

	cmd := exec.Command("/usr/sbin/bhyve", boot_args...)
	if err := cmd.Start(); err != nil {
		err = fmt.Errorf("Error starting VM: %s", err)
		return err
	}

	// bhyve exits when a VM reboots which is a bit annoying in this
	// context.  We need to check for this and restart it on success so
	// that the post-install provisioning step can run.  Once complete
	// the VM is powered off which is a non-zero exit status.
	endCh := make(chan int, 1)
	go func() {
		for {
			var rc int = 0
			if err := cmd.Wait(); err == nil {
				log.Printf("Restarting bhyve VM %s after reboot", d.config.VMName)
				cmd := exec.Command("/usr/sbin/bhyve", reboot_args...)
				if err := cmd.Start(); err != nil {
					// XXX: Report this as failing to packer
					log.Printf("Error restarting VM: %s", err)
				}
				d.lock.Lock()
				defer d.lock.Unlock()
				d.vmCmd = cmd
			} else {
				if status, ok := err.(*exec.ExitError); ok {
					rc = status.ExitCode()
				}
				endCh <- rc
				d.lock.Lock()
				defer d.lock.Unlock()
				d.vmCmd = nil
				d.vmEndCh = nil
				break
			}
		}
	}()

	d.vmCmd = cmd
	d.vmEndCh = endCh

	return nil
}

func (d *BhyveDriver) Stop() error {
	d.lock.Lock()
	defer d.lock.Unlock()

	if d.vmCmd != nil {
		if err := d.vmCmd.Process.Kill(); err != nil {
			return err
		}
	}

	return nil
}

func (d *BhyveDriver) WaitForShutdown(cancelCh <-chan struct{}) bool {
	d.lock.Lock()
	endCh := d.vmEndCh
	d.lock.Unlock()

	if endCh == nil {
		return true
	}

	select {
	case <-endCh:
		return true
	case <-cancelCh:
		return false
	}
}
