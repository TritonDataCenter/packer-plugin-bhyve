package bhyve

import (
	"context"
	"time"

	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/multistep/commonsteps"
	"github.com/hashicorp/packer-plugin-sdk/packer"
)

const BuilderId = "bhyve.builder"

type Builder struct {
	config Config
	runner multistep.Runner
}

func (b *Builder) ConfigSpec() hcldec.ObjectSpec { return b.config.FlatMapstructure().HCL2Spec() }

func (b *Builder) Prepare(raws ...interface{}) ([]string, []string, error) {
	warnings, errs := b.config.Prepare(raws...)
	if errs != nil {
		return nil, warnings, errs
	}

	return nil, warnings, nil
}

func (b *Builder) Run(ctx context.Context, ui packer.Ui, hook packer.Hook) (packer.Artifact, error) {
	state := new(multistep.BasicStateBag)
	state.Put("config", &b.config)
	state.Put("debug", b.config.PackerDebug)
	state.Put("hook", hook)
	state.Put("ui", ui)

	driver := &BhyveDriver{
		config: &b.config,
		state:  state,
	}
	state.Put("driver", driver)

	// XXX: Hack until we port SSHTimeout
	tm, _ := time.ParseDuration("1h")

	steps := []multistep.Step{}
	steps = append(steps,
		&commonsteps.StepDownload{
			Checksum:    b.config.ISOChecksum,
			Description: "ISO",
			Extension:   b.config.TargetExtension,
			ResultKey:   "iso_path",
			TargetPath:  b.config.TargetPath,
			Url:         b.config.ISOUrls,
		},
		new(stepHTTPIPDiscover),
		commonsteps.HTTPServerFromHTTPConfig(&b.config.HTTPConfig),
		new(stepCreateZvol),
		new(stepCreateVNIC),
		new(stepConfigureVNC),
		&stepBhyve{
			name: b.config.VMName,
		},
		&stepTypeBootCommand{},
		&stepWaitGuestAddress{
			timeout: tm,
		},
		&communicator.StepConnect{
			Config:    &b.config.CommConfig.Comm,
			Host:      commHost(b.config.CommConfig.Comm.Host()),
			SSHConfig: b.config.CommConfig.Comm.SSHConfigFunc(),
			SSHPort:   commPort,
			WinRMPort: commPort,
		},
		new(commonsteps.StepProvision),
		&stepShutdown{
			ShutdownTimeout: b.config.ShutdownTimeout,
			ShutdownCommand: b.config.ShutdownCommand,
			Comm:            &b.config.CommConfig.Comm,
		},
	)

	// Run!
	b.runner = commonsteps.NewRunnerWithPauseFn(steps, b.config.PackerConfig, ui, state)
	b.runner.Run(ctx, state)

	// If there was an error, return that
	if err, ok := state.GetOk("error"); ok {
		return nil, err.(error)
	}

	artifact := &Artifact{
		StateData: map[string]interface{}{"generated_data": state.Get("generated_data")},
	}
	return artifact, nil
}
