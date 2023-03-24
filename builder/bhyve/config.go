//go:generate packer-sdc mapstructure-to-hcl2 -type Config

package bhyve

import (
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/bootcommand"
	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/multistep/commonsteps"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/shutdowncommand"
	"github.com/hashicorp/packer-plugin-sdk/template/config"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

type Config struct {
	common.PackerConfig            `mapstructure:",squash"`
	commonsteps.HTTPConfig         `mapstructure:",squash"`
	commonsteps.ISOConfig          `mapstructure:",squash"`
	bootcommand.VNCConfig          `mapstructure:",squash"`
	shutdowncommand.ShutdownConfig `mapstructure:",squash"`

	BootSteps      [][]string `mapstructure:"boot_steps" required:"false"`
	CommConfig     CommConfig `mapstructure:",squash"`
	DiskSize       string     `mapstructure:"disk_size" required:"false"`
	HostNIC        string     `mapstructure:"host_nic" required:"true"`
	VMName         string     `mapstructure:"vm_name" required:"false"`
	VNCBindAddress string     `mapstructure:"vnc_bind_address" required:"false"`
	VNCPortMax     int        `mapstructure:"vnc_port_max"`
	VNCPortMin     int        `mapstructure:"vnc_port_min" required:"false"`
	VNCUsePassword bool       `mapstructure:"vnc_use_password" required:"false"`
	ZPool          string     `mapstructure:"zpool"`

	ctx interpolate.Context
}

func (c *Config) Prepare(raws ...interface{}) ([]string, error) {
	err := config.Decode(c, &config.DecodeOpts{
		PluginType:         BuilderId,
		Interpolate:        true,
		InterpolateContext: &c.ctx,
		InterpolateFilter: &interpolate.RenderFilter{
			Exclude: []string{
				"boot_command",
				"boot_steps",
			},
		},
	}, raws...)
	if err != nil {
		return nil, err
	}

	var errs *packer.MultiError
	warnings := make([]string, 0)

	isoWarnings, isoErrs := c.ISOConfig.Prepare(&c.ctx)
	warnings = append(warnings, isoWarnings...)
	errs = packer.MultiErrorAppend(errs, isoErrs...)
	errs = packer.MultiErrorAppend(errs, c.HTTPConfig.Prepare(&c.ctx)...)
	ccWarn, ccErr := c.CommConfig.Prepare(&c.ctx)
	if len(ccErr) > 0 {
		errs = packer.MultiErrorAppend(errs, ccErr...)
	}
	warnings = append(warnings, ccWarn...)

	if c.VNCBindAddress == "" {
		c.VNCBindAddress = "127.0.0.1"
	}

	if c.VNCPortMin == 0 {
		c.VNCPortMin = 5900
	}

	if c.VNCPortMax == 0 {
		c.VNCPortMax = 6000
	}

	errs = packer.MultiErrorAppend(errs, c.VNCConfig.Prepare(&c.ctx)...)

	if c.VNCPortMin < 5900 {
		errs = packer.MultiErrorAppend(
			errs, fmt.Errorf("vnc_port_min cannot be below 5900"))
	}

	if c.VNCPortMin > 65535 || c.VNCPortMax > 65535 {
		errs = packer.MultiErrorAppend(
			errs, fmt.Errorf("vmc_port_min and vnc_port_max must both be below 65535 to be valid TCP ports"))
	}

	if c.VNCPortMin > c.VNCPortMax {
		errs = packer.MultiErrorAppend(
			errs, fmt.Errorf("vnc_port_min must be less than vnc_port_max"))
	}

	if c.DiskSize == "" {
		errs = packer.MultiErrorAppend(errs, fmt.Errorf("disk_size must be specified"))
	}

	if c.ZPool == "" {
		errs = packer.MultiErrorAppend(errs, fmt.Errorf("zpool must be specified"))
	}

	if errs != nil && len(errs.Errors) > 0 {
		return warnings, errs
	}

	return warnings, nil
}
