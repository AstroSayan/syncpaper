package setter

import (
	"context"
	"fmt"
	"os/exec"
)

type GNOMESetter struct{}

func NewGNOMESetter() *GNOMESetter {
	return &GNOMESetter{}
}

func (g *GNOMESetter) Name() string {
	return "gnome"
}

func (g *GNOMESetter) IsAvailable() bool {
	return hasCommand("gsettings")
}

func (g *GNOMESetter) Set(ctx context.Context, imagePath string) error {
	fileURI := fmt.Sprintf("file://%s", imagePath)

	// Light URI
	_ = exec.CommandContext(ctx, "gsettings", "set", "org.gnome.desktop.background", "picture-uri", fileURI).Run()
	// Dark URI
	_ = exec.CommandContext(ctx, "gsettings", "set", "org.gnome.desktop.background", "picture-uri-dark", fileURI).Run()
	// Picture options
	_ = exec.CommandContext(ctx, "gsettings", "set", "org.gnome.desktop.background", "picture-options", "zoom").Run()

	return nil
}
