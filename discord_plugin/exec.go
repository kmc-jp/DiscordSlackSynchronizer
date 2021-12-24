package discord_plugin

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type externalPlugin struct {
	path string
}

func newExternalPlugin(path string) *externalPlugin {
	return &externalPlugin{path}
}

func (e *externalPlugin) GetPrimaryID(discordID string) (string, error) {
	return e.exec(fmt.Sprintf("%s\n%s\n", "GetPrimaryID", discordID))
}
func (e *externalPlugin) GetDiscordID(primaryID string) ([]string, error) {
	output, err := e.exec(fmt.Sprintf("%s\n%s\n", "GetDiscordID", primaryID))
	return strings.Split(output, ","), err
}

func (e *externalPlugin) exec(input string) (output string, err error) {
	var cmd = exec.Command(e.path)

	var stdin = new(bytes.Buffer)
	var stdout = new(bytes.Buffer)
	var stderr = new(bytes.Buffer)

	cmd = &exec.Cmd{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
	}

	stdin.WriteString(input)

	go func() {
		var err error
		var buf = make([]byte, 100)
		var n int

		for ; err == nil; n, err = stdout.Read(buf) {
			fmt.Fprintf(os.Stderr, "%s", buf[:n])
		}
	}()

	err = cmd.Wait()
	if err != nil {
		return
	}

	return stdout.String(), nil
}