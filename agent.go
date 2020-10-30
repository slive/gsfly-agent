/*
 * Author:slive
 * DATE:2020/9/9
 */
package agent

import (
	"flag"
	"github.com/Slive/gsfly-agent/agent"
	"github.com/Slive/gsfly-agent/conf"
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
	flag.StringVar(&cf, "cf", "", "config file path, as'/home/agent.properties'")
}

func RunDef(extension agent.IExtension, agentMsgHandle ...agent.IMsgHandler) {
	defer func() {
		ret := recover()
		if ret != nil {
			logx.Panic("run agent error:", ret)
		}
	}()

	flag.Parse()
	cf = strings.TrimSpace(cf)
	if len(cf) <= 0 {
		// 当前目录下或者当前目录的conf下
		cf = util.GetPwd() + "/agent.properties"
		if !util.CheckFileExist(cf) {
			cf = util.GetPwd() + "/conf/agent.properties"
		}
	}
	Run(extension, cf, agentMsgHandle...)
}

func Run(extension agent.IExtension, cfPath string, agentMsgHandle ...agent.IMsgHandler) {
	logx.Info("properties file:", cfPath)
	config := util.LoadProperties(cfPath)
	serviceConf := conf.InitServiceConf(config)
	service := agent.NewService(serviceConf, extension)
	o := make(chan os.Signal, 1)
	signal.Notify(o)
	err := service.Start()
	if err != nil {
		logx.Error("run error:", err)
		service.Stop()
		return
	}
	service.GetAgServer().AddMsgHandler(agentMsgHandle...)
	select {
	case <-o:
		service.Stop()
		logx.Info("stop....")
		break
	}
}
