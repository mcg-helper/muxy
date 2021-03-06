package throttler

import (
	"fmt"
	"github.com/YangYongZhi/muxy/log"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

const (
	tcRootQDisc        = `dev %s handle 10: root`
	tcRootExtra        = `default 1`
	tcDefaultClass     = `dev %s parent 10: classid 10:1`
	tcTargetClass      = `dev %s parent 10: classid 10:10`
	tcNetemRule        = `dev %s parent 10:10 handle 100:`
	tcRate             = `rate %vkbit`
	tcDelay            = `delay %vms`
	tcDelayistribution = `distribution %s`
	tcDelayJitter      = `%vms`
	tcDelayCorrelation = `%v%%`
	tcLoss             = `loss %v%% 25%%`
	tcReorder          = `reorder %v%% gap 3`
	tcDuplicate        = `duplicate %v%%`
	tcCorrupt          = `corrupt %v%%`
	tcAddClass         = `sudo tc class add`
	tcDelClass         = `sudo tc class del`
	tcAddQDisc         = `sudo tc qdisc add`
	tcDelQDisc         = `sudo tc qdisc del`
	iptAddTarget       = `sudo %s -A POSTROUTING -t mangle -j CLASSIFY --set-class 10:10`
	iptDelTarget       = `sudo %s -D POSTROUTING -t mangle -j CLASSIFY --set-class 10:10`
	iptDestIP          = `-d %s`
	iptProto           = `-p %s`
	iptDestPorts       = `--match multiport --dports %s`
	iptDestPort        = `--dport %s`
	iptSrcPorts        = `--match multiport --sports %s`
	iptSrcPort         = `--sport %s`
	iptDelSearch       = `class 0010:0010`
	TcList             = `sudo tc qdisc show`
	IptList            = `sudo %s -S -t mangle`
	Ip4Tables          = `iptables`
	Ip6Tables          = `ip6tables`
	iptDel             = `sudo %s -t mangle -D`
	tcExists           = `sudo tc qdisc show | grep "netem"`
	tcCheck            = `sudo tc -s qdisc`

	//tcCbq                     = `cbq avpkt 1000 bandwidth %vkbit`
)

type tcThrottler struct {
	c commander
}

func (t *tcThrottler) setup(cfg *Config) error {
	defer func() {
		if err := recover(); err != nil {
			log.Error("tc setup %s", err)
		}
	}()

	err := addRootQDisc(cfg, t.c) //The root node to append the filters

	if err != nil {
		log.Debug("addRootQDisc error : %s", err)
		return err
	}

	err = addDefaultClass(cfg, t.c) //The default class for all traffic that isn't classified

	if err != nil {
		log.Debug("addDefaultClass error : %s", err)
		return err
	}

	err = addTargetClass(cfg, t.c) //The class that the network emulator rule is assigned

	if err != nil {
		log.Debug("addTargetClass error : %s", err)
		return err
	}

	//err = addRate(cfg, t.c) //The class that the network emulator rule is assigned
	//log.Debug("addRate error : %s", err)
	//if err != nil {
	//	return err
	//}

	err = addNetemRule(cfg, t.c) //The network emulator rule that contains the desired behavior

	if err != nil {
		log.Debug("addNetemRule error : %s", err)
		return err
	}

	return addIptablesRules(cfg, t.c) //The network emulator rule that contains the desired behavior
}

func addRootQDisc(cfg *Config, c commander) error {
	//Add the root QDisc
	root := fmt.Sprintf(tcRootQDisc, cfg.Device)
	strs := []string{tcAddQDisc, root, "htb", tcRootExtra}
	cmd := strings.Join(strs, " ")

	log.Debug("Adding root qdisc :")

	return c.execute(cmd)
}

func addDefaultClass(cfg *Config, c commander) error {
	//Add the default Class
	def := fmt.Sprintf(tcDefaultClass, cfg.Device)
	rate := ""

	if cfg.DefaultBandwidth > 0 {
		rate = fmt.Sprintf(tcRate, cfg.DefaultBandwidth)
	} else {
		rate = fmt.Sprintf(tcRate, 1000000)
	}

	strs := []string{tcAddClass, def, "htb", rate}
	cmd := strings.Join(strs, " ")

	log.Debug("Adding default class :")

	return c.execute(cmd)
}

func addTargetClass(cfg *Config, c commander) error {
	//Add the target Class
	tar := fmt.Sprintf(tcTargetClass, cfg.Device)
	rate := ""

	// tc qdisc add dev eth0 root handle 1:0 tbf rate 256kbit buffer 1600 limit 3000
	if cfg.TargetBandwidth > -1 {
		rate = fmt.Sprintf(tcRate, cfg.TargetBandwidth)
	} else {
		rate = fmt.Sprintf(tcRate, 1000000)
	}

	strs := []string{tcAddClass, tar, "htb", rate}

	// use cbq
	//if cfg.TargetBandwidth > -1 {
	//	rate = fmt.Sprintf(tcCbq, cfg.TargetBandwidth)
	//} else {
	//	rate = fmt.Sprintf(tcCbq, 1000000)
	//}
	//strs := []string{tcAddClass, tar, rate}

	cmd := strings.Join(strs, " ")

	log.Debug("Adding target class :")

	return c.execute(cmd)
}

func addRate(cfg *Config, c commander) error {
	//Add the target Class
	//tar := fmt.Sprintf(tcTargetClass, cfg.Device)
	rate := ""

	if cfg.TargetBandwidth > -1 {
		rate = fmt.Sprintf(tcRate, cfg.TargetBandwidth)
	} else {
		rate = fmt.Sprintf(tcRate, 1000000)
	}

	// tc qdisc add dev eth0 root handle 1:0 tbf rate 256kbit buffer 1600 limit 3000
	strs := []string{"sudo tc qdisc add dev eth0 parent 10: handle 10:10 tbf", rate, "buffer 1600 limit 3000"}

	// use cbq
	//if cfg.TargetBandwidth > -1 {
	//	rate = fmt.Sprintf(tcCbq, cfg.TargetBandwidth)
	//} else {
	//	rate = fmt.Sprintf(tcCbq, 1000000)
	//}
	//strs := []string{tcAddClass, tar, rate}

	cmd := strings.Join(strs, " ")

	log.Debug("addRate : \n")
	log.Error(cmd)

	return c.execute(cmd)
}

func addNetemRule(cfg *Config, c commander) error {
	//Add the Network Emulator rule
	net := fmt.Sprintf(tcNetemRule, cfg.Device)
	strs := []string{tcAddQDisc, net, "netem"}

	if cfg.Latency > 0 {
		strs = append(strs, fmt.Sprintf(tcDelay, cfg.Latency))

		if cfg.LatencyJitter > 0 {
			strs = append(strs, fmt.Sprintf(tcDelayJitter, cfg.LatencyJitter))

			if cfg.LatencyCorrelation > 0 {
				strs = append(strs, fmt.Sprintf(tcDelayCorrelation, cfg.LatencyCorrelation))
			}

			if len(cfg.LatencyDistribution) > 0 {
				strs = append(strs, fmt.Sprintf(tcDelayistribution, cfg.LatencyDistribution))
			}
		}

		if cfg.LatencyReorder > 0 {
			strs = append(strs, fmt.Sprintf(tcReorder, strconv.FormatFloat(cfg.LatencyReorder, 'f', 2, 64))) //"reorder 50% gap 3"
		}

		if cfg.LatencyDuplicate > 0 {
			strs = append(strs, fmt.Sprintf(tcDuplicate, strconv.FormatFloat(cfg.LatencyDuplicate, 'f', 2, 64))) //"duplicate 50%"
		}

		if cfg.LatencyCorrupt > 0 {
			strs = append(strs, fmt.Sprintf(tcCorrupt, strconv.FormatFloat(cfg.LatencyCorrupt, 'f', 2, 64))) //"corrupt 2%"
		}
	}

	log.Debug("TargetBandwidth: %d, but if you used 'rate' in a netem command, you will received an error.", cfg.TargetBandwidth)
	if cfg.TargetBandwidth > -1 {
		// If you used 'rate' in netem, it will has an error.
		//strs = append(strs, fmt.Sprintf(tcRate, cfg.TargetBandwidth))
	}

	if cfg.PacketLoss > 0 {
		strs = append(strs, fmt.Sprintf(tcLoss, strconv.FormatFloat(cfg.PacketLoss, 'f', 2, 64)))
	}

	cmd := strings.Join(strs, " ")

	log.Debug("Adding a netem rule :")
	log.Debug("%s", cmd)
	return c.execute(cmd)
}

func addIptablesRules(cfg *Config, c commander) error {
	var err error
	if err == nil && len(cfg.TargetIps) > 0 {
		err = addIptablesRulesForAddrs(cfg, c, Ip4Tables, cfg.TargetIps)
	}
	if err == nil && len(cfg.TargetIps6) > 0 {
		err = addIptablesRulesForAddrs(cfg, c, Ip6Tables, cfg.TargetIps6)
	}
	log.Error("addIptablesRules error %s", err.Error())
	return err
}

func addIptablesRulesForAddrs(cfg *Config, c commander, command string, addrs []string) error {
	rules := []string{}
	ports := ""

	log.Debug("cfg.TargetPorts : %s", cfg.TargetPorts)
	if len(cfg.TargetPorts) > 0 {
		if len(cfg.TargetPorts) > 1 {
			prts := strings.Join(cfg.TargetPorts, ",")
			ports = fmt.Sprintf(iptSrcPorts, prts)
		} else {
			ports = fmt.Sprintf(iptSrcPort, cfg.TargetPorts[0])
		}
	}

	log.Debug("ports :" + log.Colorize(log.YELLOW, ports))

	addTargetCmd := fmt.Sprintf(iptAddTarget, command)

	if len(cfg.TargetProtos) > 0 {
		for _, ptc := range cfg.TargetProtos {
			proto := fmt.Sprintf(iptProto, ptc)
			rule := addTargetCmd + " " + proto

			if ptc != "icmp" {
				if ports != "" {
					rule += " " + ports
				}
			}

			rules = append(rules, rule)
		}
	} else {
		rules = []string{addTargetCmd}
	}

	if len(addrs) > 0 {
		iprules := []string{}
		for _, ip := range addrs {
			dest := fmt.Sprintf(iptDestIP, ip)
			if len(rules) > 0 {
				for _, rule := range rules {
					r := rule + " " + dest
					iprules = append(iprules, r)
				}
			} else {
				iprules = append(iprules, dest)
			}
		}
		if len(iprules) > 0 {
			rules = iprules
		}
	}

	log.Debug("Final rule for iptables :")
	for _, rule := range rules {
		if err := c.execute(rule); err != nil {
			return err
		}
	}

	return nil
}

func (t *tcThrottler) teardown(cfg *Config) error {
	log.Debug("Start teardown...")

	if err := delIptablesRules(cfg, t.c); err != nil {
		return err
	}

	// The root node to append the filters
	if err := delRootQDisc(cfg, t.c); err != nil {
		return err
	}
	return nil
}

func delIptablesRules(cfg *Config, c commander) error {
	iptablesCommands := []string{Ip4Tables, Ip6Tables}

	for _, iptablesCommand := range iptablesCommands {
		if !c.commandExists(iptablesCommand) {
			continue
		}
		lines, err := c.executeGetLines(fmt.Sprintf(IptList, iptablesCommand))
		if err != nil {
			// ignore exit code 3 from iptables, which might happen if the system
			// has the ip6tables command, but no IPv6 capabilities
			werr, ok := err.(*exec.ExitError)
			if !ok {
				return err
			}
			status, ok := werr.Sys().(syscall.WaitStatus)
			if !ok {
				return err
			}
			if status.ExitStatus() == 3 {
				continue
			}
			return err
		}

		delCmdPrefix := fmt.Sprintf(iptDel, iptablesCommand)

		for _, line := range lines {
			if strings.Contains(line, iptDelSearch) {
				cmd := strings.Replace(line, "-A", delCmdPrefix, 1)
				log.Debug("Deleting Iptables rule :")

				err = c.execute(cmd)

				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func delRootQDisc(cfg *Config, c commander) error {
	//Delete the root QDisc
	root := fmt.Sprintf(tcRootQDisc, cfg.Device)

	strs := []string{tcDelQDisc, root}
	cmd := strings.Join(strs, " ")

	log.Debug("Deleting root qdisc :")

	return c.execute(cmd)
}

func (t *tcThrottler) exists() bool {
	log.Debug("Check if iptables has been exist")
	if dry {
		return false
	}
	err := t.c.execute(tcExists)
	return err == nil
}

func (t *tcThrottler) check() string {
	return tcCheck
}
