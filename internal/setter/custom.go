package setter

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type CustomSetter struct {
	commandTemplate []string
}

func NewCustomSetter(template []string) *CustomSetter {
	return &CustomSetter{commandTemplate: template}
}

func (c *CustomSetter) Name() string {
	return "custom"
}

func (c *CustomSetter) IsAvailable() bool {
	return len(c.commandTemplate) > 0 && hasCommand(c.commandTemplate[0])
}

func (c *CustomSetter) Set(ctx context.Context, imagePath string) error {
	if len(c.commandTemplate) == 0 {
		return fmt.Errorf("custom command is empty")
	}

	cmdArgs := make([]string, len(c.commandTemplate))
	for i, arg := range c.commandTemplate {
		cmdArgs[i] = strings.ReplaceAll(arg, "{path}", imagePath)
	}

	cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("custom setter failed (%v): %s", err, string(out))
	}
	return nil
}
