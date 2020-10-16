package awarent

import (
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/vo"
)


func deRegisgerService(client naming_client.INamingClient, param vo.DeregisterInstanceParam)(bool,error) {
	return client.DeregisterInstance(param)
}

func registerService(client naming_client.INamingClient,param vo.RegisterInstanceParam) (bool,error){
	  return client.RegisterInstance(param vo.RegisterInstanceParam)
	// client.RegisterInstance(vo.RegisterInstanceParam{
	// 	Ip:          "192.168.1.71",
	// 	Port:        8848,
	// 	ServiceName: "nacos",
	// 	Weight:      10,
	// 	Enable:      true,
	// 	Healthy:     true,
	// 	Metadata:    map[string]string{"idc": "beijing", "debug": "true"},
	// })

	// client.RegisterInstance(vo.RegisterInstanceParam{
	// 	Ip:          "192.168.1.71",
	// 	Port:        6379,
	// 	ServiceName: "nacos",
	// 	Weight:      10,
	// 	Enable:      true,
	// 	Healthy:     true,
	// 	Metadata:    map[string]string{"idc": "beijing", "debug": "true"},
	// })

	// client.RegisterInstance(vo.RegisterInstanceParam{
	// 	Ip:          "192.168.1.71",
	// 	Port:        8801,
	// 	ServiceName: "nacos",
	// 	Weight:      10,
	// 	Enable:      true,
	// 	Healthy:     true,
	// 	Metadata:    map[string]string{"idc": "beijing", "debug": "true"},
	// })

	// client.RegisterInstance(vo.RegisterInstanceParam{
	// 	Ip:          "192.168.1.71",
	// 	Port:        9090,
	// 	ServiceName: "test",
	// 	Weight:      10,
	// 	Enable:      true,
	// 	Healthy:     true,
	// 	Metadata:    map[string]string{"idc": "beijing", "debug": "true"},
	// })

	// client.RegisterInstance(vo.RegisterInstanceParam{
	// 	Ip:          "192.168.1.71",
	// 	Port:        3306,
	// 	ServiceName: "mysql",
	// 	Weight:      10,
	// 	Enable:      true,
	// 	Healthy:     true,
	// 	Metadata:    map[string]string{"idc": "beijing", "debug": "true"},
	// })

	// client.DeregisterInstance(vo.DeregisterInstanceParam{
	// 	Ip:          "192.168.1.71",
	// 	Port:        8848,
	// 	ServiceName: "nacos",
	// })
}
