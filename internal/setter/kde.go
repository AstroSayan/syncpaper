package setter

import (
	"context"
	"fmt"
	"os/exec"
)

type KDESetter struct{}

func NewKDESetter() *KDESetter {
	return &KDESetter{}
}

func (k *KDESetter) Name() string {
	return "kde"
}

func (k *KDESetter) IsAvailable() bool {
	return hasCommand("qdbus") || hasCommand("gdbus") || hasCommand("kwriteconfig6")
}

func (k *KDESetter) Set(ctx context.Context, imagePath string) error {
	js := fmt.Sprintf(`
var allDesktops = desktops();
for (var i = 0; i < allDesktops.length; i++) {
    var d = allDesktops[i];
    d.wallpaperPlugin = "org.kde.image";
    d.currentConfigGroup = Array("Wallpaper", "org.kde.image", "General");
    d.writeConfig("Image", "file://%s");
}
`, imagePath)

	if hasCommand("qdbus") {
		cmd := exec.CommandContext(ctx, "qdbus", "org.kde.plasmashell", "/PlasmaShell", "org.kde.PlasmaShell.evaluateScript", js)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	if hasCommand("gdbus") {
		cmd := exec.CommandContext(ctx, "gdbus", "call", "--session", "--dest", "org.kde.plasmashell",
			"--object-path", "/PlasmaShell", "--method", "org.kde.PlasmaShell.evaluateScript", js)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	return fmt.Errorf("failed to apply wallpaper to KDE Plasma via DBus")
}
