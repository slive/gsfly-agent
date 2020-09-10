/*
 * Author:slive
 * DATE:2020/9/10
 */
package main

import (
	"fmt"
	"github.com/Slive/gsfly-agent/agent"
	"github.com/Slive/gsfly/bootstrap"
	"github.com/Slive/gsfly/channel"
	logx "github.com/Slive/gsfly/logger"
	"math/rand"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	key_prefix_agent        = "agent"
	key_ch_readTimeout      = "channel.readTimeout"
	key_ch_writeTimeout     = "channel.writeTimeout"
	key_ch_readBufSize      = "channel.readBufSize"
	key_ch_writeBufSize     = "channel.writeBufSize"
	key_ch_closeRevFailTime = "channel.closeRevFailTime"

	// agent.channel.readBufSize = 102400
	// ## 写最大限制
	// agent.channel.writeBufSize = 102400
	// ## 接收失败n次后，关闭channel
	// agent.channel.closeRevFailTime = 3
)

func InitServiceConf(config map[string]string) agent.IServiceConf {
	logDirKey := "agent.log.dir"
	logDir := config[logDirKey]
	delete(config, logDirKey)

	logFileKey := "agent.log.file"
	logFile := config[logFileKey]
	delete(config, logFileKey)

	logLevelKey := "agent.log.level"
	logLevel := config[logLevelKey]
	delete(config, logLevelKey)

	initLogConf(logFile, logDir, logLevel)

	readPoolConf := initReadPoolConf(config)
	logx.Info("readPoolConf:", readPoolConf)

	channelConf := initChannelConf(config)
	logx.Info("channelConf:", channelConf)

	channel.InitChannelConfs(readPoolConf, channelConf)

	agentId := config["agent.id"]
	if len(agentId) <= 0 {
		agentId = fmt.Sprintf("agent-%v", rand.Int())
	}

	locations := initLocations(config)

	serverConf := initServerConf(config, agentId)

	if serverConf == nil {
		logx.Panic("serverconf init is nil.")
	}
	logx.Info("serverConf:", serverConf)

	upstreamConfs := initUpstreamConfs(config)
	logx.Info("upstreamConfs:", upstreamConfs)

	agServerConf := agent.NewAgServerConf(agentId, serverConf, locations...)
	serviceConf := agent.NewServiceConf(agentId, agServerConf, upstreamConfs...)
	return serviceConf
}

func initLogConf(logFile string, logDir string, logLevel string) {
	var logConf *logx.LogConf
	if len(logFile) > 0 {
		logConf = &logx.LogConf{
			LogFile:        logFile,
			LogDir:         logDir,
			Level:          logLevel,
			MaxRemainCount: 10,
		}
	} else {
		logConf = logx.NewDefaultLogConf()
	}
	logx.InitLogger(logConf)
	logx.Info("logConf:", logConf)
}

func initReadPoolConf(config map[string]string) *channel.ReadPoolConf {
	readPoolKey := "agent.readpool.maxCpuSize"
	readPoolStr := config[readPoolKey]
	delete(config, readPoolKey)
	readPoolSize := channel.MAX_READ_POOL_EVERY_CPU
	if len(readPoolStr) > 0 {
		retInt, err := strconv.ParseInt(readPoolStr, 10, 32)
		if err == nil {
			readPoolSize = int(retInt)
		} else {
			logx.Panic("invalid readPoolSize.")
		}
	}

	readQueueKey := "agent.readqueue.maxSize"
	readQueueStr := config[readQueueKey]
	delete(config, readQueueKey)
	readQueueSize := channel.MAX_READ_QUEUE_SIZE
	if len(readQueueStr) > 0 {
		retInt, err := strconv.ParseInt(readQueueStr, 10, 32)
		if err == nil {
			readQueueSize = int(retInt)
		} else {
			logx.Panic("invalid readQueueSize.")
		}
	}
	readPoolConf := channel.NewReadPoolConf(runtime.NumCPU()*readPoolSize, readQueueSize)
	return readPoolConf
}

func initUpstreamConfs(config map[string]string) []agent.IUpstreamConf {
	var upstreamConfs []agent.IUpstreamConf
	upstreamMap := make(map[string]string)
	for key, v := range config {
		if strings.Contains(key, "agent.upstream") {
			delete(config, key)
			upstreamMap[key] = v
		}
	}
	if len(upstreamMap) > 0 {
		upsIdKey := "agent.upstream.id"
		upId := upstreamMap[upsIdKey]
		delete(config, upsIdKey)
		if len(upId) > 0 {
			var upsIds []string
			if strings.Index(upId, ";") > 0 {
				upsIds = strings.Split(upId, ";")
			} else if strings.Index(upId, ",") > 0 {
				upsIds = strings.Split(upId, ",")
			}

			if upsIds == nil {
				logx.Panic("invalid upstreamId")
			}

			upstreamConfs = make([]agent.IUpstreamConf, len(upsIds))
			for index, upsId := range upsIds {
				upsTypeKey := "agent.upstream." + upsId + ".type"
				upsType := upstreamMap[upsTypeKey]
				delete(upstreamMap, upsTypeKey)
				upsLbKey := "agent.upstream." + upsId + ".loadBalance"
				loadBalanceStr := upstreamMap[upsLbKey]
				delete(upstreamMap, upsLbKey)

				var dstClientConfs []bootstrap.IClientConf
				dtsKey := "agent.upstream." + upsId + ".dstclient"
				var loadbalance agent.LoadBalanceType
				var err error
				if len(loadBalanceStr) <= 0 {
					loadbalance = agent.LOADBALANCE_DEFAULT
				} else {
					loadbalance, err = agent.GetLoadBalanceType(loadBalanceStr)
					if err != nil {
						logx.Panic(err)
					}
				}

				dstIndex := 0
				for {
					indexKey := fmt.Sprintf((dtsKey + ".%v."), dstIndex)
					// ip不可为空
					dstIpKey := indexKey + "ip"
					dstIp := upstreamMap[dstIpKey]
					delete(upstreamMap, dstIpKey)
					if len(dstIp) <= 0 {
						break
					}

					dstPortKey := indexKey + "port"
					dstPortStr := upstreamMap[dstPortKey]
					delete(upstreamMap, dstPortKey)
					dstPort := 19980
					if len(dstPortStr) > 0 {
						retInt, err := strconv.ParseInt(dstPortStr, 10, 32)
						if err != nil {
							logx.Panic("dstPort is invalid, dstPortKey:" + dstPortKey)
						}
						dstPort = int(retInt)
					}
					dstNetworkKey := indexKey + "network"
					network := upstreamMap[dstNetworkKey]
					delete(upstreamMap, dstNetworkKey)
					var dstClientConf bootstrap.IClientConf
					if network == channel.PROTOCOL_WS.String() {
						dstSchemeKey := indexKey + "scheme"
						dstScheme := upstreamMap[dstSchemeKey]
						logx.Info(dstSchemeKey + "=" + dstScheme)
						delete(upstreamMap, dstSchemeKey)

						dstPathKey := indexKey + "path"
						dstPath := upstreamMap[dstPathKey]
						delete(upstreamMap, dstPathKey)

						dstSubprotocolKey := indexKey + "subprotocol"
						dstSubrotocol := upstreamMap[dstSubprotocolKey]
						delete(upstreamMap, dstSubprotocolKey)
						dstClientConf = bootstrap.NewWsClientConf(dstIp, dstPort, dstScheme, dstPath, dstSubrotocol)
					} else if network == channel.PROTOCOL_KWS00.String() {
						dstPathKey := indexKey + ".path"
						dstPath := upstreamMap[dstPathKey]
						delete(upstreamMap, dstPathKey)
						dstClientConf = bootstrap.NewKws00ClientConf(dstIp, dstPort, dstPath)
					}
					logx.Info("dstClientConf:", dstClientConf)
					if dstClientConf != nil {
						if dstClientConfs == nil {
							dstClientConfs = make([]bootstrap.IClientConf, 1)
							dstClientConfs[0] = dstClientConf
						} else {
							dstClientConfs = append(dstClientConfs, dstClientConf)
						}
					}
					dstIndex++
				}

				var upstreamConf agent.IUpstreamConf
				if len(upsType) <= 0 || upsType == agent.UPSTREAM_PROXY || dstClientConfs != nil {
					upstreamConf = agent.NewProxyConf(upsId, loadbalance, dstClientConfs...)
				} else {
					// TODO...
				}
				logx.Info("upstreamConf:", upstreamConf)
				if upstreamConf != nil {
					upstreamConfs[index] = upstreamConf
				}
			}
		} else {
			logx.Panic("upsteamId is nil.")
		}
	}
	return upstreamConfs
}

func initServerConf(config map[string]string, agentId string) bootstrap.IServerConf {
	serverIp := config["agent.server.ip"]
	if len(serverIp) <= 0 {
		// 本地ip
		serverIp = ""
	}
	portStr := config["agent.server.port"]
	port := 9080
	if len(portStr) > 0 {
		retId, err := strconv.ParseInt(portStr, 10, 32)
		if err != nil {
			logx.Panic("port is error.")
		} else {
			port = int(retId)
		}
	}
	network := config["agent.server.network"]
	if len(network) <= 0 {
		network = channel.PROTOCOL_WS.String()
	}
	maxChannelSizeStr := config["agent.server.maxChannelSize"]
	maxChannelSize := 0
	if len(maxChannelSizeStr) > 0 {
		retId, err := strconv.ParseInt(portStr, 10, 32)
		if err != nil {
			logx.Panic("maxChannelSize is error.")
		} else {
			maxChannelSize = int(retId)
		}
	}
	var serverConf bootstrap.IServerConf
	if network == channel.PROTOCOL_KWS00.String() {
		serverConf = bootstrap.NewKw00ServerConf(serverIp, port)
		serverConf.SetMaxChannelSize(maxChannelSize)
	} else if network == channel.PROTOCOL_WS.String() {
		scheme := config["agent.server.scheme"]
		path := config["agent.server.path"]
		subprotocol := config["agent.server.subprotocol"]
		serverConf = bootstrap.NewWsServerConf(serverIp, port, scheme, path, subprotocol)
		serverConf.SetMaxChannelSize(maxChannelSize)
	}
	return serverConf
}

func initLocations(config map[string]string) []agent.ILocationConf {
	locationMap := make(map[string]string)
	for key, v := range config {
		if strings.Contains(key, "agent.server.location") {
			delete(config, key)
			locationMap[key] = v
		}
	}

	var locationConfs []agent.ILocationConf
	lcSize := len(locationMap)
	if lcSize > 0 {
		index := 0
		for {
			patternKey := fmt.Sprintf("agent.server.location.%v.pattern", index)
			upstreamIdKey := fmt.Sprintf("agent.server.location.%v.upstreamId", index)
			pattern := locationMap[patternKey]
			upstreamId := locationMap[upstreamIdKey]
			if len(pattern) <= 0 || len(upstreamId) <= 0 {
				break
			}
			locationConf := agent.NewLocationConf(pattern, upstreamId, nil)
			logx.Info("locationConf:", locationConf)
			if locationConfs == nil {
				locationConfs = make([]agent.ILocationConf, 1)
				locationConfs[index] = locationConf
			} else {
				locationConfs = append(locationConfs, locationConf)
			}
			index++
		}
	}
	return locationConfs
}

func initChannelConf(config map[string]string) *channel.ChannelConf {
	defChConf := channel.NewDefChannelConf(channel.PROTOCOL_WS)
	readBufSize := config[key_prefix_agent+"."+key_ch_readBufSize]
	if len(readBufSize) > 0 {
		retInt, err := strconv.ParseInt(readBufSize, 10, 32)
		if err == nil {
			defChConf.ReadBufSize = int(retInt)
		} else {
			logx.Info("error:", err)
		}
	}

	writeBufSize := config[key_prefix_agent+"."+key_ch_writeBufSize]
	if len(writeBufSize) > 0 {
		retInt, err := strconv.ParseInt(writeBufSize, 10, 32)
		if err == nil {
			defChConf.WriteBufSize = int(retInt)
		} else {
			logx.Info("error:", err)
		}
	}
	readTimeout := config[key_prefix_agent+"."+key_ch_readTimeout]
	if len(writeBufSize) > 0 {
		retInt, err := strconv.ParseInt(readTimeout, 10, 32)
		if err == nil {
			defChConf.ReadTimeout = time.Duration(retInt)
		} else {
			logx.Info("error:", err)
		}
	}

	writeTimeout := config[key_prefix_agent+"."+key_ch_writeTimeout]
	if len(writeTimeout) > 0 {
		retInt, err := strconv.ParseInt(writeTimeout, 10, 32)
		if err == nil {
			defChConf.WriteTimeout = time.Duration(retInt)
		} else {
			logx.Info("error:", err)
		}
	}
	closeRevFailTime := config[key_prefix_agent+"."+key_ch_closeRevFailTime]
	if len(closeRevFailTime) > 0 {
		retInt, err := strconv.ParseInt(closeRevFailTime, 10, 32)
		if err == nil {
			defChConf.CloseRevFailTime = int(retInt)
		} else {
			logx.Info("error:", err)
		}
	}
	return defChConf
}
