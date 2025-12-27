package main

import (
	"github.com/Lupino/go-periodic"
	"github.com/Lupino/go-periodic/cmd/periodic/subcmd"
	"github.com/Lupino/go-periodic/protocol"
	"github.com/urfave/cli"
	"log"
	"os"
	"runtime"
	"time"
)

func getRSAParam(c *cli.Context) protocol.RSAConnParam {
	return protocol.RSAConnParam{
		PrivateKeyPath:      c.GlobalString("rsa-private-key-path"),
		ServerPublicKeyPath: c.GlobalString("rsa-public-key-path"),
		Mode:                c.GlobalInt("rsa-mode"),
	}
}

func main() {
	app := cli.NewApp()
	app.Name = "periodic"
	app.Usage = "Periodic task system client"
	app.Version = periodic.Version
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "H",
			Value:  "unix:///tmp/periodic.sock",
			Usage:  "Socket path eg: tcp://127.0.0.1:5000",
			EnvVar: "PERIODIC_PORT",
		},
		cli.StringFlag{
			Name:   "rsa-public-key-path",
			Value:  "",
			Usage:  "RSA Transport server public key file path",
			EnvVar: "RSA_PUBLIC_KEY_PATH",
		},
		cli.StringFlag{
			Name:   "rsa-private-key-path",
			Value:  "",
			Usage:  "RSA Transport private key file path",
			EnvVar: "RSA_PRIVATE_KEY_PATH",
		},
		cli.IntFlag{
			Name:   "rsa-mode",
			Value:  0,
			Usage:  "RSA Transport mode",
			EnvVar: "RSA_MODE",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:  "status",
			Usage: "Show status",
			Action: func(c *cli.Context) error {
				subcmd.ShowStatus(c.GlobalString("H"), getRSAParam(c))
				return nil
			},
		},
		{
			Name:  "submit",
			Usage: "Submit job",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "f",
					Value: "",
					Usage: "function name",
				},
				cli.StringFlag{
					Name:  "n",
					Value: "",
					Usage: "job name",
				},
				cli.StringFlag{
					Name:  "args",
					Value: "",
					Usage: "job workload",
				},
				cli.IntFlag{
					Name:  "sched_later",
					Value: 0,
					Usage: "job sched_later",
				},
			},
			Action: func(c *cli.Context) error {
				var name = c.String("n")
				var funcName = c.String("f")
				var opts = map[string]interface{}{
					"args": c.String("args"),
				}
				if len(name) == 0 || len(funcName) == 0 {
					cli.ShowCommandHelp(c, "submit")
					log.Fatal("Job name and func is require")
				}
				delay := c.Int("sched_later")
				var now = time.Now()
				var schedAt = int64(now.Unix()) + int64(delay)
				opts["schedat"] = schedAt
				subcmd.SubmitJob(c.GlobalString("H"), getRSAParam(c), funcName, name, opts)
				return nil
			},
		},
		{
			Name:  "remove",
			Usage: "Remove job",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "f",
					Value: "",
					Usage: "function name",
				},
				cli.StringFlag{
					Name:  "n",
					Value: "",
					Usage: "job name",
				},
			},
			Action: func(c *cli.Context) error {
				var name = c.String("n")
				var funcName = c.String("f")
				if len(name) == 0 || len(funcName) == 0 {
					cli.ShowCommandHelp(c, "remove")
					log.Fatal("Job name and func is require")
				}
				subcmd.RemoveJob(c.GlobalString("H"), getRSAParam(c), funcName, name)
				return nil
			},
		},
		{
			Name:  "drop",
			Usage: "Drop func",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "f",
					Value: "",
					Usage: "function name",
				},
			},
			Action: func(c *cli.Context) error {
				Func := c.String("f")
				if len(Func) == 0 {
					cli.ShowCommandHelp(c, "drop")
					log.Fatal("function name is required")
				}
				subcmd.DropFunc(c.GlobalString("H"), getRSAParam(c), Func)
				return nil
			},
		},
		{
			Name:  "run",
			Usage: "Run func",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "f",
					Value: "",
					Usage: "function name required",
				},
				cli.StringFlag{
					Name:  "exec",
					Value: "",
					Usage: "command required",
				},
				cli.IntFlag{
					Name:  "n",
					Value: runtime.NumCPU() * 2,
					Usage: "the size of goroutines. (optional)",
				},
			},
			Action: func(c *cli.Context) error {
				Func := c.String("f")
				exec := c.String("exec")
				n := c.Int("n")
				if len(Func) == 0 {
					cli.ShowCommandHelp(c, "run")
					log.Fatal("function name is required")
				}
				if len(exec) == 0 {
					cli.ShowCommandHelp(c, "run")
					log.Fatal("command is required")
				}
				subcmd.Run(c.GlobalString("H"), getRSAParam(c), Func, exec, n)
				return nil
			},
		},
	}
	app.Action = func(c *cli.Context) error {
		cli.ShowAppHelp(c)
		return nil
	}

	app.Run(os.Args)
}
