package nacoscli

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/stretchr/testify/assert"

	"github.com/go-dev-frame/sponge/pkg/utils"
)

var (
	ipAddr      = "192.168.3.37"
	port        = 8848
	namespaceID = "3454d2b5-2455-4d0e-bf6d-e033b086bb4c"
)

func TestParse(t *testing.T) {
	//conf := new(map[string]interface{})
	params := &Params{
		IPAddr:      ipAddr,
		Port:        uint64(port),
		NamespaceID: namespaceID,
		Group:       "dev",
		DataID:      "serverNameExample.yml",
		Format:      "yaml",
	}

	utils.SafeRunWithTimeout(time.Second*2, func(cancel context.CancelFunc) {
		format, data, err := GetConfig(params)
		t.Log(err, format, data)
	})

	//conf = new(map[string]interface{})
	params = &Params{
		Group:  "dev",
		DataID: "serverNameExample.yml",
		Format: "yaml",
	}
	clientConfig := &constant.ClientConfig{
		NamespaceId:         namespaceID,
		TimeoutMs:           1000,
		NotLoadCacheAtStart: true,
		LogDir:              os.TempDir() + "/nacos/log",
		CacheDir:            os.TempDir() + "/nacos/cache",
	}
	serverConfigs := []constant.ServerConfig{
		{
			IpAddr: ipAddr,
			Port:   uint64(port),
		},
	}
	utils.SafeRunWithTimeout(time.Second*2, func(cancel context.CancelFunc) {
		format, data, err := GetConfig(params,
			WithClientConfig(clientConfig),
			WithServerConfigs(serverConfigs),
			WithAuth("foo", "bar"),
		)
		t.Log(err, format, data)
	})
}

func TestNewNamingClient(t *testing.T) {
	utils.SafeRunWithTimeout(time.Second*2, func(cancel context.CancelFunc) {
		namingClient, err := NewNamingClient(ipAddr, port, namespaceID)
		t.Log(err, namingClient)
	})
}

func TestError(t *testing.T) {
	p := &Params{}
	p.Group = ""
	err := p.valid()
	assert.Error(t, err)

	p.Group = "group"
	p.DataID = ""
	err = p.valid()
	assert.Error(t, err)

	p.Group = "group"
	p.DataID = "id"
	p.Format = ""
	err = p.valid()
	assert.Error(t, err)

	p.Group = "group"
	p.DataID = "id"
	p.Format = "yml"
	err = p.valid()
	assert.NoError(t, err)

	p.Group = "group"
	p.DataID = "id"
	p.Format = "unknown"
	err = p.valid()
	assert.Error(t, err)

	_, _, err = GetConfig(&Params{})
	assert.Error(t, err)
}
