package mysql

import (
	"fmt"
	"path"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/JREAMLU/j-core/consul"
	"github.com/JREAMLU/j-core/ext"
	"github.com/jinzhu/gorm"
	// mysql driver
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

const (
	// Driver mysql driver
	Driver = "mysql"
	// Conn conn
	Conn = "%s:%s@tcp(%s:%s)/%s?charset=%s&loc=%s&parseTime=True"
	// Local Sets the location for time
	Local = "Asia%2FShanghai"
	// Charset charset
	Charset = "utf8"
	// READONLY readonly
	READONLY = "ReadOnly"
)

var lock sync.Mutex

// Config mysql config in consul
type Config struct {
	InstanceName string
	DBName       string
	ReadWrite    readwrite
	ReadOnly     readonly
}

type readwrite struct {
	Server   string
	Password string
	Port     string
	UserID   string
	CharSet  string
}

type readonly struct {
	Server   string
	Password string
	Port     string
	UserID   string
	CharSet  string
}

// Load load mysql
func Load(consulAddr string, names ...string) (map[string]*gorm.DB, error) {
	return LoadConfig(consulAddr, names...)
}

// LoadConfig load config
func LoadConfig(consulAddr string, names ...string) (map[string]*gorm.DB, error) {
	client, err := consul.NewClient(consul.SetAddress(consulAddr))
	if err != nil {
		return nil, err
	}

	if len(names) == 0 {
		return loadAll(client)
	}

	return loadByNames(client, names)
}

// GetReadOnly get readonly
func GetReadOnly(name string) string {
	return ext.StringSplice(name, READONLY)
}

func loadByNames(client *consul.Client, names []string) (map[string]*gorm.DB, error) {
	for i := range names {
		names[i] = path.Join(consul.MYSQL, names[i])
	}

	return loadConfig(client, names)
}

func loadAll(client *consul.Client) (map[string]*gorm.DB, error) {
	keys, err := client.GetChildKeys(consul.MYSQL)
	if err != nil {
		return nil, err
	}

	return loadConfig(client, keys)
}

func loadConfig(client *consul.Client, keys []string) (map[string]*gorm.DB, error) {
	var dbs = make(map[string]*gorm.DB, len(keys)*2)
	for _, key := range keys {
		instanceName := path.Base(key)
		buf, err := client.Get(key)
		if err != nil {
			return nil, err
		}
		var config Config
		if _, err = toml.Decode(buf, &config); err != nil {
			return nil, err
		}

		// read and write
		rwdb, err := registerDatabase(instanceName, config, true)
		if err != nil {
			return nil, err
		}
		lock.Lock()
		dbs[instanceName] = rwdb
		lock.Unlock()

		// readonly
		rdb, err := registerDatabase(instanceName, config, false)
		if err != nil {
			return nil, err
		}
		lock.Lock()
		dbs[GetReadOnly(instanceName)] = rdb
		lock.Unlock()
	}

	return dbs, nil
}

func registerDatabase(name string, config Config, isWrite bool) (*gorm.DB, error) {
	if isWrite {
		var charset string
		if config.ReadWrite.CharSet == "" {
			charset = "utf8"
		} else {
			charset = config.ReadWrite.CharSet
		}
		conn := fmt.Sprintf(Conn, config.ReadWrite.UserID, config.ReadWrite.Password,
			config.ReadWrite.Server, config.ReadWrite.Port, config.DBName, charset, Local)
		db, err := gorm.Open(Driver, conn)

		return db, err
	}

	var charset string
	if config.ReadOnly.CharSet == "" {
		charset = "utf8"
	} else {
		charset = config.ReadOnly.CharSet
	}
	conn := fmt.Sprintf(Conn, config.ReadOnly.UserID, config.ReadOnly.Password,
		config.ReadOnly.Server, config.ReadOnly.Port, config.DBName, charset, Local)
	db, err := gorm.Open(Driver, conn)

	return db, err
}