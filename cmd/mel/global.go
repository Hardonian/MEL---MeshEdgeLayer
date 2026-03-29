package main

import (
	"strings"

	"github.com/mel-project/mel/internal/cliout"
	"github.com/mel-project/mel/internal/config"
)

// globalOpts holds flags that apply across mel subcommands (parsed from argv).
type globalOpts struct {
	ConfigPath     string
	ConfigExplicit bool
	Profile        string
	JSON           bool
	Text           bool
	Wide           bool
	Color          bool
	ColorSet       bool
}

var cliGlobal globalOpts

func defaultGlobalOpts() globalOpts {
	return globalOpts{
		ConfigPath: "configs/mel.example.json",
		JSON:       true,
	}
}

// parseGlobalFlags extracts known global flags from args and returns the remainder for the subcommand.
func parseGlobalFlags(args []string) ([]string, globalOpts) {
	g := defaultGlobalOpts()
	var out []string
	for i := 0; i < len(args); {
		a := args[i]
		switch {
		case a == "--json":
			g.JSON = true
			g.Text = false
			i++
			continue
		case a == "--text" || a == "--human":
			g.Text = true
			g.JSON = false
			i++
			continue
		case a == "--wide":
			g.Wide = true
			i++
			continue
		case a == "--color":
			g.Color = true
			g.ColorSet = true
			i++
			continue
		case a == "--no-color":
			g.Color = false
			g.ColorSet = true
			i++
			continue
		case a == "--config" || a == "-config":
			if i+1 >= len(args) {
				out = append(out, a)
				i++
				continue
			}
			g.ConfigPath = args[i+1]
			g.ConfigExplicit = true
			i += 2
			continue
		case strings.HasPrefix(a, "--config="):
			g.ConfigPath = strings.TrimPrefix(a, "--config=")
			g.ConfigExplicit = true
			i++
			continue
		case a == "--profile":
			if i+1 >= len(args) {
				out = append(out, a)
				i++
				continue
			}
			g.Profile = args[i+1]
			i += 2
			continue
		case strings.HasPrefix(a, "--profile="):
			g.Profile = strings.TrimPrefix(a, "--profile=")
			i++
			continue
		default:
			out = append(out, a)
			i++
		}
	}
	if !g.ColorSet {
		g.Color = cliout.DetectTTY()
	}
	return out, g
}

func init() {
	cliGlobal = defaultGlobalOpts()
}


// configFlagDefault is the default for per-command --config when the operator did not pass a global --config.
func configFlagDefault() string {
	if strings.TrimSpace(cliGlobal.ConfigPath) != "" {
		return cliGlobal.ConfigPath
	}
	return "configs/mel.example.json"
}

func loadConfigFile(path string) (config.Config, []byte, error) {
	return config.LoadWithOptions(path, config.LoadOptions{Profile: cliGlobal.Profile})
}

func loadConfigSide(path, profile string) (config.Config, error) {
	c, _, err := config.LoadWithOptions(path, config.LoadOptions{Profile: profile})
	return c, err
}
