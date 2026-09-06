package setter

import (
	"context"
	"fmt"
	"os/exec"
)

type X11Setter struct{}

func NewX11Setter() *X11Setter {
	return &X11Setter{}
}

func (x *X11Setter) Name() string {
	return "x11"
}

func (x *X11Setter) IsAvailable() bool {
	return hasCommand("feh") || hasCommand("nitrogen")
}

func (x *X11Setter) Set(ctx context.Context, imagePath string) error {
	if hasCommand("feh") {
		return exec.CommandContext(ctx, "feh", "--bg-fill", imagePath).Run()
	}
	if hasCommand("nitrogen") {
		return exec.CommandContext(ctx, "nitrogen", "--set-zoom-fill", "--save", imagePath).Run()
	}
	return fmt.Errorf("neither feh nor nitrogen found for X11 wallpaper setting")
}
