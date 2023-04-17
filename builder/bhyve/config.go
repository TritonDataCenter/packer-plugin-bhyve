//go:generate packer-sdc mapstructure-to-hcl2 -type Config

package bhyve

import (
	"fmt"
	"os"

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
	DiskName       string     `mapstructure:"disk_name" required:"false"`
	DiskSize       string     `mapstructure:"disk_size" required:"false"`
	DiskUseZVOL    bool       `mapstructure:"disk_use_zvol" required:"false"`
	DiskZPool      string     `mapstructure:"disk_zpool" required:"false"`
	HostNIC        string     `mapstructure:"host_nic"`
	OutputDir      string     `mapstructure:"output_directory" required:"false"`
	VMName         string     `mapstructure:"vm_name" required:"false"`
	VNCBindAddress string     `mapstructure:"vnc_bind_address" required:"false"`
	VNCPortMax     int        `mapstructure:"vnc_port_max"`
	VNCPortMin     int        `mapstructure:"vnc_port_min" required:"false"`
	VNCUsePassword bool       `mapstructure:"vnc_use_password" required:"false"`
	VNICCreate     bool       `mapstructure:"vnic_create" required:"false"`
	VNICName       string     `mapstructure:"vnic_name" required:"false"`
	VNICLink       string     `mapstructure:"vnic_link" required:"false"`

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
	errs = packer.MultiErrorAppend(errs, c.ShutdownConfig.Prepare(&c.ctx)...)
	ccWarn, ccErr := c.CommConfig.Prepare(&c.ctx)
	if len(ccErr) > 0 {
		errs = packer.MultiErrorAppend(errs, ccErr...)
	}
	warnings = append(warnings, ccWarn...)

	if c.DiskName == "" {
		c.DiskName = fmt.Sprintf("disk-%s", c.PackerBuildName)
	}

	if c.DiskSize == "" {
		errs = packer.MultiErrorAppend(errs, fmt.Errorf("disk_size must be specified"))
	}

	if c.DiskZPool == "" {
		c.DiskZPool = "zones"
	}

	if c.OutputDir == "" {
		c.OutputDir = fmt.Sprintf("output-%s", c.PackerBuildName)
	}
	if !c.PackerForce {
		if _, err := os.Stat(c.OutputDir); err == nil {
			errs = packer.MultiErrorAppend(
				errs,
				fmt.Errorf("Output directory '%s' already exists. It must not exist.", c.OutputDir))
		}
	}

	if c.VMName == "" {
		c.VMName = fmt.Sprintf("packer-%s", c.PackerBuildName)
	}

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

	if c.VNICLink == "" {
		c.VNICLink = c.HostNIC
	}

	if c.VNICName == "" {
		c.VNICName = "packer0"
	}

	if errs != nil && len(errs.Errors) > 0 {
		return warnings, errs
	}

	return warnings, nil
}
