package state

import (
	"sort"
	"strings"
	"testing"

	"github.com/haproxytech/haproxy-consul-connect/consul"
	"github.com/haproxytech/models/v2"
	"github.com/stretchr/testify/require"
)

func GetTestConsulConfig() consul.Config {
	return consul.Config{
		Downstream: consul.Downstream{
			LocalBindAddress:  "127.0.0.2",
			LocalBindPort:     9999,
			TargetAddress:     "128.0.0.5",
			TargetPort:        8888,
			AppNameHeaderName: "X-App",
			ConnectTimeout:    consul.DefaultConnectTimeout,
			ReadTimeout:       consul.DefaultReadTimeout,
		},
		Upstreams: []consul.Upstream{
			consul.Upstream{
				Name:             "service_1",
				LocalBindAddress: "127.0.0.1",
				LocalBindPort:    10000,
				ConnectTimeout:   consul.DefaultConnectTimeout,
				ReadTimeout:      consul.DefaultReadTimeout,
				Nodes: []consul.UpstreamNode{
					consul.UpstreamNode{
						Host:   "1.2.3.4",
						Port:   8080,
						Weight: 5,
					},
					consul.UpstreamNode{
						Host:   "1.2.3.5",
						Port:   8081,
						Weight: 8,
					},
				},
			},
		},
	}
}

func GetTestHAConfig(baseCfg string, certVersion string) State {
	s := State{
		Frontends: []Frontend{

			// downstream front
			Frontend{
				Frontend: models.Frontend{
					Name:           "front_downstream",
					DefaultBackend: "back_downstream",
					ClientTimeout:  int64p(int(consul.DefaultReadTimeout.Milliseconds())),
					Mode:           models.FrontendModeHTTP,
					Httplog:        true,
				},
				Bind: models.Bind{
					Name:           "front_downstream_bind",
					Address:        "127.0.0.2",
					Port:           int64p(9999),
					Ssl:            true,
					SslCafile:      baseCfg + "/ca" + certVersion,
					SslCertificate: baseCfg + "/cert" + certVersion,
					Verify:         models.BindVerifyRequired,
				},
				LogTarget: &models.LogTarget{
					Index:    int64p(0),
					Address:  baseCfg + "/logs.sock",
					Facility: models.LogTargetFacilityLocal0,
					Format:   models.LogTargetFormatRfc5424,
				},
				Filter: &FrontendFilter{
					Filter: models.Filter{
						Index:      int64p(0),
						Type:       models.FilterTypeSpoe,
						SpoeEngine: "intentions",
						SpoeConfig: baseCfg + "/spoe",
					},
					Rule: models.TCPRequestRule{
						Index:    int64p(0),
						Action:   models.TCPRequestRuleActionReject,
						Cond:     models.TCPRequestRuleCondUnless,
						CondTest: "{ var(sess.connect.auth) -m int eq 1 }",
						Type:     models.TCPRequestRuleTypeContent,
					},
				},
			},

			// upstream front
			Frontend{
				Frontend: models.Frontend{
					Name:           "front_service_1",
					DefaultBackend: "back_service_1",
					ClientTimeout:  int64p(int(consul.DefaultReadTimeout.Milliseconds())),
					Mode:           models.FrontendModeHTTP,
					Httplog:        true,
				},
				Bind: models.Bind{
					Name:    "front_service_1_bind",
					Address: "127.0.0.1",
					Port:    int64p(10000),
				},
				LogTarget: &models.LogTarget{
					Index:    int64p(0),
					Address:  baseCfg + "/logs.sock",
					Facility: models.LogTargetFacilityLocal0,
					Format:   models.LogTargetFormatRfc5424,
				},
			},
		},

		Backends: []Backend{

			// downstream backend
			Backend{
				Backend: models.Backend{
					Name:           "back_downstream",
					ServerTimeout:  int64p(int(consul.DefaultReadTimeout.Milliseconds())),
					ConnectTimeout: int64p(int(consul.DefaultConnectTimeout.Milliseconds())),
					Mode:           models.BackendModeHTTP,
				},
				Servers: []models.Server{
					models.Server{
						Name:    "downstream_node",
						Address: "128.0.0.5",
						Port:    int64p(8888),
					},
				},
				LogTarget: &models.LogTarget{
					Index:    int64p(0),
					Address:  baseCfg + "/logs.sock",
					Facility: models.LogTargetFacilityLocal0,
					Format:   models.LogTargetFormatRfc5424,
				},
				HTTPRequestRules: []models.HTTPRequestRule{
					{
						Index:     int64p(0),
						Type:      models.HTTPRequestRuleTypeAddHeader,
						HdrName:   "X-App",
						HdrFormat: "%[var(sess.connect.source_app)]",
					},
				},
			},

			// upstream backend
			Backend{
				Backend: models.Backend{
					Name:           "back_service_1",
					ServerTimeout:  int64p(int(consul.DefaultReadTimeout.Milliseconds())),
					ConnectTimeout: int64p(int(consul.DefaultConnectTimeout.Milliseconds())),
					Mode:           models.BackendModeHTTP,
					Balance: &models.Balance{
						Algorithm: stringp(models.BalanceAlgorithmLeastconn),
					},
				},
				Servers: []models.Server{
					models.Server{
						Name:           "srv_0",
						Address:        "1.2.3.4",
						Port:           int64p(8080),
						Weight:         int64p(5),
						Ssl:            models.ServerSslEnabled,
						SslCafile:      baseCfg + "/ca" + certVersion,
						SslCertificate: baseCfg + "/cert" + certVersion,
						Verify:         models.BindVerifyRequired,
						Maintenance:    models.ServerMaintenanceDisabled,
					},
					models.Server{
						Name:           "srv_1",
						Address:        "1.2.3.5",
						Port:           int64p(8081),
						Weight:         int64p(8),
						Ssl:            models.ServerSslEnabled,
						SslCafile:      baseCfg + "/ca" + certVersion,
						SslCertificate: baseCfg + "/cert" + certVersion,
						Verify:         models.BindVerifyRequired,
						Maintenance:    models.ServerMaintenanceDisabled,
					},
				},
				LogTarget: &models.LogTarget{
					Index:    int64p(0),
					Address:  baseCfg + "/logs.sock",
					Facility: models.LogTargetFacilityLocal0,
					Format:   models.LogTargetFormatRfc5424,
				},
			},

			// spoe backend
			Backend{
				Backend: models.Backend{
					Name:           "spoe_back",
					ServerTimeout:  int64p(int(spoeTimeout.Milliseconds())),
					ConnectTimeout: int64p(int(spoeTimeout.Milliseconds())),
					Mode:           models.BackendModeTCP,
				},
				Servers: []models.Server{
					models.Server{
						Name:    "haproxy_connect",
						Address: "unix@",
					},
				},
			},
		},
	}

	sort.Slice(s.Frontends, func(i, j int) bool {
		return strings.Compare(s.Frontends[i].Frontend.Name, s.Frontends[j].Frontend.Name) < 0
	})

	sort.Slice(s.Backends, func(i, j int) bool {
		return strings.Compare(s.Backends[i].Backend.Name, s.Backends[j].Backend.Name) < 0
	})

	return s
}

var TestOpts = Options{
	EnableIntentions: true,
	LogRequests:      true,
	LogSocket:        "//logs.sock",
	SPOEConfigPath:   "//spoe",
}

var TestCertStore = fakeCertStore{}

func TestSnapshotDownstream(t *testing.T) {
	generated, err := Generate(TestOpts, TestCertStore, State{}, GetTestConsulConfig())
	require.Nil(t, err)

	require.Equal(t, GetTestHAConfig("/", ""), generated)
}

func TestServerUpdate(t *testing.T) {
	consulCfg := GetTestConsulConfig()
	consulCfg.Upstreams[0].Nodes = consulCfg.Upstreams[0].Nodes[1:]

	oldState := GetTestHAConfig("/", "")

	// remove first server
	expectedNewState := GetTestHAConfig("/", "")
	expectedNewState.Backends[1].Servers[0].Maintenance = models.ServerMaintenanceEnabled
	expectedNewState.Backends[1].Servers[0].Address = "127.0.0.1"
	expectedNewState.Backends[1].Servers[0].Port = int64p(1)
	expectedNewState.Backends[1].Servers[0].Weight = int64p(1)

	generated, err := Generate(TestOpts, TestCertStore, oldState, consulCfg)
	require.Nil(t, err)
	require.Equal(t, expectedNewState, generated)

	// re-add first server
	generated, err = Generate(TestOpts, TestCertStore, generated, GetTestConsulConfig())
	require.Nil(t, err)
	require.Equal(t, GetTestHAConfig("/", ""), generated)

	// add another one
	consulCfg = GetTestConsulConfig()
	consulCfg.Upstreams[0].Nodes = append(consulCfg.Upstreams[0].Nodes, consul.UpstreamNode{
		Host:   "1.2.3.6",
		Port:   8082,
		Weight: 10,
	})

	expectedNewState = GetTestHAConfig("/", "")
	expectedNewState.Backends[1].Servers = append(expectedNewState.Backends[1].Servers,
		models.Server{
			Name:           "srv_2",
			Address:        "1.2.3.6",
			Port:           int64p(8082),
			Weight:         int64p(10),
			Ssl:            models.ServerSslEnabled,
			SslCafile:      "//ca",
			SslCertificate: "//cert",
			Verify:         models.BindVerifyRequired,
			Maintenance:    models.ServerMaintenanceDisabled,
		},
		models.Server{
			Name:           "srv_3",
			Address:        "127.0.0.1",
			Port:           int64p(1),
			Weight:         int64p(1),
			Ssl:            models.ServerSslEnabled,
			SslCafile:      "//ca",
			SslCertificate: "//cert",
			Verify:         models.BindVerifyRequired,
			Maintenance:    models.ServerMaintenanceEnabled,
		},
	)

	generated, err = Generate(TestOpts, TestCertStore, generated, consulCfg)
	require.Nil(t, err)
	require.Equal(t, expectedNewState, generated)
}

func TestCertificateUpgrade(t *testing.T) {
	generated, err := Generate(TestOpts, fakeCertStore{"1"}, State{}, GetTestConsulConfig())
	require.Nil(t, err)

	generated, err = Generate(TestOpts, fakeCertStore{"2"}, generated, GetTestConsulConfig())
	require.Nil(t, err)

	haCfg := GetTestHAConfig("/", "2")

	require.Equal(t, haCfg, generated)
}

type fakeCertStore struct {
	suffix string
}

func (s fakeCertStore) CertsPath(t consul.TLS) (string, string, error) {
	return "//ca" + s.suffix, "//cert" + s.suffix, nil
}
