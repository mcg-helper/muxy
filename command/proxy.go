package command

import (
	"flag"
	"github.com/YangYongZhi/muxy/api/http"
	"github.com/YangYongZhi/muxy/log"
	"strings"
)

// ProxyCommand enables an http proxy for http tampering
type ProxyCommand struct {
	Meta Meta
}

// Run the HTTP Proxy CLI command
func (pc *ProxyCommand) Run(args []string) int {
	cmdFlags := flag.NewFlagSet("proxy", flag.ContinueOnError)
	cmdFlags.Usage = func() { pc.Meta.UI.Output(pc.Help()) }

	cmdFlags.StringVar(&c.ConfigFile, "config", "", "Path to a YAML configuration file")

	// Validate
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	go func() { //Start a http server for api rest interface.
		muxyHttpServer := http.MuxyApiServer{"muxy_api_agent"}
		http.Muxy = Muxy
		log.Debug("Start [%s] for restful apis.", log.Colorize(log.RED, "HTTP Server"))
		muxyHttpServer.Start()
	}()

	go func() {
		heartbeatSender := http.HeartBeatSender{}
		log.Debug("")
		log.Debug("Start [%s] for heartbeat.", log.Colorize(log.RED, "Heartbeat Sender"))
		heartbeatSender.Start()
	}()

	//Start a muxy instance.
	Muxy.Run()

	return 0
}

// Help prints out detailed help for this command
func (pc *ProxyCommand) Help() string {
	helpText := `
Usage: muck proxy [options]

  Run the Muck proxy.

Options:

  --config                    Location of Muxy configuration file
`

	return strings.TrimSpace(helpText)
}

// Synopsis prints out help for this command
func (pc *ProxyCommand) Synopsis() string {
	return "Run the Muxy proxy"
}
