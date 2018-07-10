package redis

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/JREAMLU/j-kit/constant"
	"github.com/JREAMLU/j-kit/ext"
	"github.com/gomodule/redigo/redis"
	"github.com/mna/redisc"
)

const (
	_defaultPagesize          = 500
	_defaultMaxidle           = 50
	_defaultIdletimeout       = 240 * time.Second
	_defaultClusterRetryTime  = 3
	_defaultClusterRetryDelay = 100 * time.Millisecond
	// MASTER read & write
	MASTER = true
	// SLAVE read
	SLAVE = false
)

// Structure redis structure
type Structure struct {
	KeyPrefixFmt string
	InstanceName string
	readPool     *redis.Pool
	writePool    *redis.Pool
	clusterPool  *redisc.Cluster
	writeConn    string
	readConn     string
	mutex        sync.Mutex
	MaxIdle      int
	IdleTimeout  time.Duration
}

// NewStructure new structure
func NewStructure(instanceName, keyPrefixFmt string) Structure {
	return Structure{
		KeyPrefixFmt: keyPrefixFmt,
		InstanceName: instanceName,
		MaxIdle:      _defaultMaxidle,
		IdleTimeout:  _defaultIdletimeout,
	}
}

// SetMaxIdle set max idle
func (s *Structure) SetMaxIdle(maxIdle int) {
	s.MaxIdle = maxIdle
}

// SetIdleTimeout set idle timeout
func (s *Structure) SetIdleTimeout(idleTimeout time.Duration) {
	s.IdleTimeout = idleTimeout
}

// InitKey init redis key
func (s *Structure) InitKey(keySuffix string) string {
	if ext.StringEq(keySuffix) {
		return s.KeyPrefixFmt
	}

	return fmt.Sprintf(s.KeyPrefixFmt, keySuffix)
}

// Bool bool base operation
func (s *Structure) Bool(isMaster bool, cmd string, params ...interface{}) (reply bool, err error) {
	conn := s.getConn(isMaster)
	if conn == nil {
		return false, configNotExists(s.InstanceName, isMaster)
	}

	reply, err = redis.Bool(conn.Do(cmd, params...))
	conn.Close()

	return reply, err
}

// String string base operation
func (s *Structure) String(isMaster bool, cmd string, params ...interface{}) (reply string, err error) {
	conn := s.getConn(isMaster)
	if conn == nil {
		return "", configNotExists(s.InstanceName, isMaster)
	}

	reply, err = redis.String(conn.Do(cmd, params...))
	conn.Close()

	return reply, err
}

// Strings strings base operation
func (s *Structure) Strings(isMaster bool, cmd string, params ...interface{}) (reply []string, err error) {
	conn := s.getConn(isMaster)
	if conn == nil {
		return nil, configNotExists(s.InstanceName, isMaster)
	}

	reply, err = redis.Strings(conn.Do(cmd, params...))
	conn.Close()

	return reply, err
}

// Int int base operation
func (s *Structure) Int(isMaster bool, cmd string, params ...interface{}) (reply int, err error) {
	conn := s.getConn(isMaster)
	if conn == nil {
		return constant.ZeroInt, configNotExists(s.InstanceName, isMaster)
	}

	reply, err = redis.Int(conn.Do(cmd, params...))
	conn.Close()

	return reply, err
}

// Ints ints base operation
func (s *Structure) Ints(isMaster bool, cmd string, params ...interface{}) (reply []int, err error) {
	conn := s.getConn(isMaster)
	if conn == nil {
		return nil, configNotExists(s.InstanceName, isMaster)
	}

	reply, err = redis.Ints(conn.Do(cmd, params...))
	conn.Close()

	return reply, err
}

// Int64 int64 base operation
func (s *Structure) Int64(isMaster bool, cmd string, params ...interface{}) (reply int64, err error) {
	conn := s.getConn(isMaster)
	if conn == nil {
		return constant.ZeroInt64, configNotExists(s.InstanceName, isMaster)
	}

	reply, err = redis.Int64(conn.Do(cmd, params...))
	conn.Close()

	return reply, err
}

// ScanAllMap scan all return map
func (s *Structure) ScanAllMap(key, luaBody string) (map[string]string, error) {
	cursor := 0
	conn := s.getConn(SLAVE)
	if conn == nil {
		return nil, configNotExists(s.InstanceName, SLAVE)
	}
	defer conn.Close()
	connStr := s.getConnstr(false)
	if connStr == "" {
		return nil, configNotExists(s.InstanceName, false)
	}
	script := GetScript(connStr, luaBody)
	if script == nil {
		return nil, configNotExists(s.InstanceName, false)
	}
	result := make(map[string]string)
	for {
		results, err := redis.Strings(script.Do(conn, 0, key, cursor, _defaultPagesize))
		if err != nil {
			return nil, err
		}
		for i := 1; i < len(results); i = i + 2 {
			result[results[i]] = results[i+1]
		}
		cursor, err = strconv.Atoi(results[0])
		if err != nil {
			return nil, err
		}
		if cursor == 0 {
			break
		}
	}
	return result, nil
}

func (s *Structure) getConn(isMaster bool) redis.Conn {
	if s.isCluster() {
		return s.getClusterConn()
	}

	return s.getClientConn(isMaster)
}

func (s *Structure) isCluster() bool {
	return isCluster(s.InstanceName)
}

func (s *Structure) getClientConn(isMaster bool) redis.Conn {
	// refresh true, set pool = nil, then get new pool
	if isRefreshPool(s.InstanceName) {
		s.mutex.Lock()
		s.writePool = nil
		s.readPool = nil
		toggleRefreshPool(s.InstanceName, false)
		s.mutex.Unlock()
	}

	if s.writePool == nil {
		s.writePool = s.getPool(s.InstanceName, true)
		s.readPool = s.getPool(s.InstanceName, false)
	}

	if isMaster {
		if s.writePool == nil {
			return nil
		}

		return s.writePool.Get()
	}

	if s.readPool == nil {
		return nil
	}

	return s.readPool.Get()
}

func (s *Structure) getClusterConn() redis.Conn {
	// refresh true, set pool = nil, then get new pool
	if isRefreshPool(s.InstanceName) {
		s.mutex.Lock()
		if s.clusterPool != nil {
			s.clusterPool.Close()
			delete(poolcs, s.InstanceName)
			s.clusterPool = nil
		}
		toggleRefreshPool(s.InstanceName, false)
		s.mutex.Unlock()
	}

	if s.clusterPool == nil {
		if s.clusterPool = s.getClusterPool(s.InstanceName); s.clusterPool == nil {
			return nil
		}
	}

	retryConn, err := redisc.RetryConn(s.clusterPool.Get(), _defaultClusterRetryTime, _defaultClusterRetryDelay)
	if err != nil {
		return nil
	}

	return retryConn
}

func (s *Structure) getPool(instanceName string, isMaster bool) *redis.Pool {
	conn := getConn(instanceName, isMaster)
	if conn == nil {
		return nil
	}

	return GetPool(conn.ConnStr, conn.DB, s.MaxIdle, s.IdleTimeout)
}

func (s *Structure) getClusterPool(instanceName string) *redisc.Cluster {
	return getPoolc(instanceName, s.MaxIdle, s.IdleTimeout)
}

func (s *Structure) getConnstr(isMaster bool) string {
	if isMaster && s.writeConn != "" {
		return s.writeConn
	}

	if !isMaster && s.readConn != "" {
		return s.readConn
	}

	conn := getConn(s.InstanceName, isMaster)
	if conn == nil {
		conn = getConn(s.InstanceName, isMaster)
	}

	if conn == nil {
		return constant.EmptyStr
	}

	if isMaster {
		s.writeConn = conn.ConnStr
	} else {
		s.readConn = conn.ConnStr
	}

	return conn.ConnStr
}
