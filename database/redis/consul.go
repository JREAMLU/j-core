package redis

import (
	"fmt"
	"path"

	"github.com/JREAMLU/j-core/consul"
)

// MasterSlave master slave
type MasterSlave bool

// Configs redis configs in consul
type Configs struct {
	InstanceName string
	Master       []master
	Slave        []slave
}

type master struct {
	DB        string
	IP        string
	Port      string
	PoolSize  int
	IsCluster bool
	IsMaster  bool
}

type slave struct {
	DB        string
	IP        string
	Port      string
	PoolSize  int
	IsCluster bool
	IsMaster  bool
}

func (masterSlave MasterSlave) String() string {
	if masterSlave {
		return "master"
	}
	return "slave"
}

// Load load redis
func Load(consulAddr string, isWatching bool, names ...string) error {
	return LoadConfig(consulAddr, isWatching, names...)
}

// LoadConfig load config
func LoadConfig(consulAddr string, isWatching bool, names ...string) error {
	client, err := consul.NewClient(consul.SetAddress(consulAddr))
	if err != nil {
		return err
	}

	if err = loadAll(client); err != nil {
		return err
	}

	return nil
}

func loadAll(client *consul.Client) error {
	// 获取所有实例的key
	keys, err := client.GetChildKeys(consul.Redis)
	if err != nil {
		return err
	}

	return loadConfig(client, keys)
}

func loadConfig(client *consul.Client, prefixKeys []string) error {
	loadMasterOrSlave := func(key, instanceName string, isMaster MasterSlave) error {
		keys, err := client.GetChildKeys(key)
		if err != nil {
			return err
		}

		if len(keys) == 0 && isMaster {
			return fmt.Errorf(consul.DirNotExist, key)
		}

		for i := range keys {
			if err = loadNode(client, isMaster, keys[i], instanceName); err != nil {
				return err
			}
		}

		return nil
	}

	for _, key := range prefixKeys {
		instanceName := path.Base(key)
		if err := loadMasterOrSlave(key, instanceName, true); err != nil {
			return err
		}
		if err := loadMasterOrSlave(key, instanceName, false); err != nil {
			return err
		}
	}

	return nil
}

func loadNode(client *consul.Client, isMaster MasterSlave, keyPre, instanceName string) error {
	return nil
}
