/*
 * Author:slive
 * DATE:2020/9/9
 */
package main

import (
	"flag"
	"github.com/Slive/gsfly-agent/agent"
	logx "github.com/Slive/gsfly/logger"
	"github.com/Slive/gsfly/util"
	"os"
	"os/signal"
	"strings"
)

const (
	FILE_TYPE_PROPERTIES = ".properties"
	FILE_TYPE_YAML       = ".yaml"
	FILE_TYPE_YML        = ".yml"
	FILE_TYPE_JSON       = ".json"
)

var (
	cf        string
	ft        string
	agentAddr string
	dstAddrs  string
	pattern   string
)

func init() {
	flag.StringVar(&cf, "cf", "", "config file path, as'/home/agent.properties', support filetype see ft(fileType)")
}

func main() {
	flag.Parse()
	cf = strings.TrimSpace(cf)
	if len(cf) > 0 {
		config := util.LoadProperties(cf)
		serviceConf := InitServiceConf(config)
		service := agent.NewService(serviceConf)
		o := make(chan os.Signal, 1)
		signal.Notify(o, os.Interrupt)
		service.Start()
		select {
		case <-o:
			service.Stop()
			logx.Info("stop....")
			break
		}
	} else {
		panic("please input config file path.")
	}
}
