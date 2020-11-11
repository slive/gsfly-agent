/*
 * Author:slive
 * DATE:2020/9/9
 */
package agent

import (
	"flag"
	"github.com/Slive/gsfly-agent/agent"
	config "github.com/Slive/gsfly-agent/config"
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

func RunDef(extension agent.IExtension) {
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
	Run(extension, cf)
}

func Run(extension agent.IExtension, cfPath string) {
	logx.Info("properties file:", cfPath)
	properties := util.LoadProperties(cfPath)
	serviceConf := config.InitServiceConf(properties)
	service := agent.NewService(serviceConf, extension)
	extension.SetExtConf(properties)

	// 启动
	err := service.Start()
	if err != nil {
		logx.Error("run error:", err)
		service.Stop()
		return
	}
	if extension != nil {
		msgHandlers := extension.GetAgentMsgHandlers()
		if msgHandlers != nil {
			service.GetAgServer().AddMsgHandler(msgHandlers...)
		}
	}

	o := make(chan os.Signal, 1)
	signal.Notify(o)
	select {
	case s := <-o:
		service.Stop()
		logx.Info("stop...., signal:", s)
		break
	}
}
