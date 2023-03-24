package bhyve

import (
	"errors"

	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

// Based on qemu's CommConfig with unnecessary sections removed.
type CommConfig struct {
	Comm        communicator.Config `mapstructure:",squash"`
	HostPortMin int                 `mapstructure:"host_port_min" required:"false"`
	HostPortMax int                 `mapstructure:"host_port_max" required:"false"`
}

func (c *CommConfig) Prepare(ctx *interpolate.Context) (warnings []string, errs []error) {
	if c.HostPortMin == 0 {
		c.HostPortMin = 2222
	}

	if c.HostPortMax == 0 {
		c.HostPortMax = 4444
	}

	errs = c.Comm.Prepare(ctx)

	if c.HostPortMin > c.HostPortMax {
		errs = append(errs,
			errors.New("host_port_min must be less than host_port_max"))
	}

	if c.HostPortMin < 0 {
		errs = append(errs, errors.New("host_port_min must be positive"))
	}

	return
}
