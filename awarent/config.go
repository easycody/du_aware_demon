package awarent

import (
	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

func getConfig(configClient config_client.IConfigClient, vo vo.ConfigParam) (string, error) {
	return configClient.GetConfig(vo)
}

func listenConfig(configClient config_client.IConfigClient, vo vo.ConfigParam) {
	return configClient.ListenConfig(vo)
}
