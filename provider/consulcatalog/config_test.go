package consulcatalog

import (
	"testing"
	"text/template"
	"time"

	"github.com/containous/flaeg/parse"
	"github.com/containous/traefik/provider/label"
	"github.com/containous/traefik/types"
	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
)

func TestProviderBuildConfiguration(t *testing.T) {
	p := &Provider{
		Domain:               "localhost",
		Prefix:               "traefik",
		ExposedByDefault:     false,
		FrontEndRule:         "Host:{{.ServiceName}}.{{.Domain}}",
		frontEndRuleTemplate: template.New("consul catalog frontend rule"),
	}

	testCases := []struct {
		desc              string
		nodes             []catalogUpdate
		expectedFrontends map[string]*types.Frontend
		expectedBackends  map[string]*types.Backend
	}{
		{
			desc:              "Should build config of nothing",
			nodes:             []catalogUpdate{},
			expectedFrontends: map[string]*types.Frontend{},
			expectedBackends:  map[string]*types.Backend{},
		},
		{
			desc: "Should build config with no frontend and backend",
			nodes: []catalogUpdate{
				{
					Service: &serviceUpdate{
						ServiceName: "test",
					},
				},
			},
			expectedFrontends: map[string]*types.Frontend{},
			expectedBackends:  map[string]*types.Backend{},
		},
		{
			desc: "Should build config who contains one frontend and one backend",
			nodes: []catalogUpdate{
				{
					Service: &serviceUpdate{
						ServiceName: "test",
						Attributes: []string{
							"random.foo=bar",
							label.TraefikBackendLoadBalancerMethod + "=drr",
							label.TraefikBackendCircuitBreakerExpression + "=NetworkErrorRatio() > 0.5",
							label.TraefikBackendMaxConnAmount + "=1000",
							label.TraefikBackendMaxConnExtractorFunc + "=client.ip",
							label.TraefikFrontendAuthBasicUsers + "=test:$apr1$H6uskkkW$IgXLP6ewTrSuBkTrqE8wj/,test2:$apr1$d9hr9HBB$4HxwgUir3HP4EsggP/QNo0",
						},
					},
					Nodes: []*api.ServiceEntry{
						{
							Service: &api.AgentService{
								Service: "test",
								Address: "127.0.0.1",
								Port:    80,
								Tags: []string{
									"random.foo=bar",
									label.TraefikWeight + "=42",
									label.TraefikFrontendPassHostHeader + "=true",
									label.TraefikProtocol + "=https",
								},
							},
							Node: &api.Node{
								Node:    "localhost",
								Address: "127.0.0.1",
							},
						},
					},
				},
			},
			expectedFrontends: map[string]*types.Frontend{
				"frontend-test": {
					Backend:        "backend-test",
					PassHostHeader: true,
					Routes: map[string]types.Route{
						"route-host-test": {
							Rule: "Host:test.localhost",
						},
					},
					Auth: &types.Auth{
						Basic: &types.Basic{
							Users: []string{"test:$apr1$H6uskkkW$IgXLP6ewTrSuBkTrqE8wj/",
								"test2:$apr1$d9hr9HBB$4HxwgUir3HP4EsggP/QNo0"},
						},
					},
					EntryPoints: []string{},
				},
			},
			expectedBackends: map[string]*types.Backend{
				"backend-test": {
					Servers: map[string]types.Server{
						"test-0-ecTTsmX1vPktQQrl53WhNDy-HEg": {
							URL:    "https://127.0.0.1:80",
							Weight: 42,
						},
					},
					LoadBalancer: &types.LoadBalancer{
						Method: "drr",
					},
					CircuitBreaker: &types.CircuitBreaker{
						Expression: "NetworkErrorRatio() > 0.5",
					},
					MaxConn: &types.MaxConn{
						Amount:        1000,
						ExtractorFunc: "client.ip",
					},
				},
			},
		},
		{
			desc: "Should build config which contains three frontends and one backend",
			nodes: []catalogUpdate{
				{
					Service: &serviceUpdate{
						ServiceName: "test",
						Attributes: []string{
							"random.foo=bar",
							label.Prefix + "frontend.rule=Host:A",
							label.Prefix + "frontends.test1.rule=Host:B",
							label.Prefix + "frontends.test2.rule=Host:C",
						},
					},
					Nodes: []*api.ServiceEntry{
						{
							Service: &api.AgentService{
								Service: "test",
								Address: "127.0.0.1",
								Port:    80,
								Tags: []string{
									"random.foo=bar",
								},
							},
							Node: &api.Node{
								Node:    "localhost",
								Address: "127.0.0.1",
							},
						},
					},
				},
			},
			expectedFrontends: map[string]*types.Frontend{
				"frontend-test": {
					Backend:        "backend-test",
					PassHostHeader: true,
					Routes: map[string]types.Route{
						"route-host-test": {
							Rule: "Host:A",
						},
					},
					EntryPoints: []string{},
				},
				"frontend-test-test1": {
					Backend:        "backend-test",
					PassHostHeader: true,
					Routes: map[string]types.Route{
						"route-host-test-test1": {
							Rule: "Host:B",
						},
					},
					EntryPoints: []string{},
				},
				"frontend-test-test2": {
					Backend:        "backend-test",
					PassHostHeader: true,
					Routes: map[string]types.Route{
						"route-host-test-test2": {
							Rule: "Host:C",
						},
					},
					EntryPoints: []string{},
				},
			},
			expectedBackends: map[string]*types.Backend{
				"backend-test": {
					Servers: map[string]types.Server{
						"test-0-O0Tnh-SwzY69M6SurTKP3wNKkzI": {
							URL:    "http://127.0.0.1:80",
							Weight: 1,
						},
					},
				},
			},
		},
		{
			desc: "Should build config with a basic auth with a backward compatibility",
			nodes: []catalogUpdate{
				{
					Service: &serviceUpdate{
						ServiceName: "test",
						Attributes: []string{
							"random.foo=bar",
							label.TraefikFrontendAuthBasicUsers + "=test:$apr1$H6uskkkW$IgXLP6ewTrSuBkTrqE8wj/,test2:$apr1$d9hr9HBB$4HxwgUir3HP4EsggP/QNo0",
						},
					},
					Nodes: []*api.ServiceEntry{
						{
							Service: &api.AgentService{
								Service: "test",
								Address: "127.0.0.1",
								Port:    80,
								Tags: []string{
									"random.foo=bar",
									label.TraefikWeight + "=42",
									label.TraefikFrontendPassHostHeader + "=true",
									label.TraefikProtocol + "=https",
								},
							},
							Node: &api.Node{
								Node:    "localhost",
								Address: "127.0.0.1",
							},
						},
					},
				},
			},
			expectedFrontends: map[string]*types.Frontend{
				"frontend-test": {
					Backend:        "backend-test",
					PassHostHeader: true,
					Routes: map[string]types.Route{
						"route-host-test": {
							Rule: "Host:test.localhost",
						},
					},
					Auth: &types.Auth{
						Basic: &types.Basic{
							Users: []string{"test:$apr1$H6uskkkW$IgXLP6ewTrSuBkTrqE8wj/",
								"test2:$apr1$d9hr9HBB$4HxwgUir3HP4EsggP/QNo0"},
						},
					},
					EntryPoints: []string{},
				},
			},
			expectedBackends: map[string]*types.Backend{
				"backend-test": {
					Servers: map[string]types.Server{
						"test-0-ecTTsmX1vPktQQrl53WhNDy-HEg": {
							URL:    "https://127.0.0.1:80",
							Weight: 42,
						},
					},
				},
			},
		},
		{
			desc: "Should build config with a digest auth",
			nodes: []catalogUpdate{
				{
					Service: &serviceUpdate{
						ServiceName: "test",
						Attributes: []string{
							"random.foo=bar",
							label.TraefikFrontendAuthDigestRemoveHeader + "=true",
							label.TraefikFrontendAuthDigestUsers + "=test:$apr1$H6uskkkW$IgXLP6ewTrSuBkTrqE8wj/,test2:$apr1$d9hr9HBB$4HxwgUir3HP4EsggP/QNo0",
							label.TraefikFrontendAuthDigestUsersFile + "=.htpasswd",
						},
					},
					Nodes: []*api.ServiceEntry{
						{
							Service: &api.AgentService{
								Service: "test",
								Address: "127.0.0.1",
								Port:    80,
								Tags: []string{
									"random.foo=bar",
									label.TraefikWeight + "=42",
									label.TraefikFrontendPassHostHeader + "=true",
									label.TraefikProtocol + "=https",
								},
							},
							Node: &api.Node{
								Node:    "localhost",
								Address: "127.0.0.1",
							},
						},
					},
				},
			},
			expectedFrontends: map[string]*types.Frontend{
				"frontend-test": {
					Backend:        "backend-test",
					PassHostHeader: true,
					Routes: map[string]types.Route{
						"route-host-test": {
							Rule: "Host:test.localhost",
						},
					},
					Auth: &types.Auth{
						Digest: &types.Digest{
							RemoveHeader: true,
							Users: []string{"test:$apr1$H6uskkkW$IgXLP6ewTrSuBkTrqE8wj/",
								"test2:$apr1$d9hr9HBB$4HxwgUir3HP4EsggP/QNo0"},
							UsersFile: ".htpasswd",
						},
					},
					EntryPoints: []string{},
				},
			},
			expectedBackends: map[string]*types.Backend{
				"backend-test": {
					Servers: map[string]types.Server{
						"test-0-ecTTsmX1vPktQQrl53WhNDy-HEg": {
							URL:    "https://127.0.0.1:80",
							Weight: 42,
						},
					},
				},
			},
		},
		{
			desc: "Should build config with a forward auth",
			nodes: []catalogUpdate{
				{
					Service: &serviceUpdate{
						ServiceName: "test",
						Attributes: []string{
							"random.foo=bar",
							label.TraefikFrontendAuthForwardAddress + "=auth.server",
							label.TraefikFrontendAuthForwardTrustForwardHeader + "=true",
							label.TraefikFrontendAuthForwardTLSCa + "=ca.crt",
							label.TraefikFrontendAuthForwardTLSCaOptional + "=true",
							label.TraefikFrontendAuthForwardTLSCert + "=server.crt",
							label.TraefikFrontendAuthForwardTLSKey + "=server.key",
							label.TraefikFrontendAuthForwardTLSInsecureSkipVerify + "=true",
							label.TraefikFrontendAuthHeaderField + "=X-WebAuth-User",
						},
					},
					Nodes: []*api.ServiceEntry{
						{
							Service: &api.AgentService{
								Service: "test",
								Address: "127.0.0.1",
								Port:    80,
								Tags: []string{
									"random.foo=bar",
									label.TraefikWeight + "=42",
									label.TraefikFrontendPassHostHeader + "=true",
									label.TraefikProtocol + "=https",
								},
							},
							Node: &api.Node{
								Node:    "localhost",
								Address: "127.0.0.1",
							},
						},
					},
				},
			},
			expectedFrontends: map[string]*types.Frontend{
				"frontend-test": {
					Backend:        "backend-test",
					PassHostHeader: true,
					Routes: map[string]types.Route{
						"route-host-test": {
							Rule: "Host:test.localhost",
						},
					},
					Auth: &types.Auth{
						HeaderField: "X-WebAuth-User",
						Forward: &types.Forward{
							Address:            "auth.server",
							TrustForwardHeader: true,
							TLS: &types.ClientTLS{
								CA:                 "ca.crt",
								CAOptional:         true,
								InsecureSkipVerify: true,
								Cert:               "server.crt",
								Key:                "server.key",
							},
						},
					},
					EntryPoints: []string{},
				},
			},
			expectedBackends: map[string]*types.Backend{
				"backend-test": {
					Servers: map[string]types.Server{
						"test-0-ecTTsmX1vPktQQrl53WhNDy-HEg": {
							URL:    "https://127.0.0.1:80",
							Weight: 42,
						},
					},
				},
			},
		},
		{
			desc: "when all labels are set",
			nodes: []catalogUpdate{
				{
					Service: &serviceUpdate{
						ServiceName: "test",
						Attributes: []string{
							label.TraefikBackend + "=foobar",

							label.TraefikBackendCircuitBreakerExpression + "=NetworkErrorRatio() > 0.5",
							label.TraefikBackendHealthCheckPath + "=/health",
							label.TraefikBackendHealthCheckScheme + "=http",
							label.TraefikBackendHealthCheckPort + "=880",
							label.TraefikBackendHealthCheckInterval + "=6",
							label.TraefikBackendHealthCheckTimeout + "=3",
							label.TraefikBackendHealthCheckHostname + "=foo.com",
							label.TraefikBackendHealthCheckHeaders + "=Foo:bar || Bar:foo",
							label.TraefikBackendLoadBalancerMethod + "=drr",
							label.TraefikBackendLoadBalancerStickiness + "=true",
							label.TraefikBackendLoadBalancerStickinessCookieName + "=chocolate",
							label.TraefikBackendMaxConnAmount + "=666",
							label.TraefikBackendMaxConnExtractorFunc + "=client.ip",
							label.TraefikBackendBufferingMaxResponseBodyBytes + "=10485760",
							label.TraefikBackendBufferingMemResponseBodyBytes + "=2097152",
							label.TraefikBackendBufferingMaxRequestBodyBytes + "=10485760",
							label.TraefikBackendBufferingMemRequestBodyBytes + "=2097152",
							label.TraefikBackendBufferingRetryExpression + "=IsNetworkError() && Attempts() <= 2",

							label.TraefikFrontendPassTLSClientCertPem + "=true",
							label.TraefikFrontendPassTLSClientCertInfosNotBefore + "=true",
							label.TraefikFrontendPassTLSClientCertInfosNotAfter + "=true",
							label.TraefikFrontendPassTLSClientCertInfosSans + "=true",
							label.TraefikFrontendPassTLSClientCertInfosSubjectCommonName + "=true",
							label.TraefikFrontendPassTLSClientCertInfosSubjectCountry + "=true",
							label.TraefikFrontendPassTLSClientCertInfosSubjectLocality + "=true",
							label.TraefikFrontendPassTLSClientCertInfosSubjectOrganization + "=true",
							label.TraefikFrontendPassTLSClientCertInfosSubjectProvince + "=true",
							label.TraefikFrontendPassTLSClientCertInfosSubjectSerialNumber + "=true",

							label.TraefikFrontendAuthBasic + "=test:$apr1$H6uskkkW$IgXLP6ewTrSuBkTrqE8wj/,test2:$apr1$d9hr9HBB$4HxwgUir3HP4EsggP/QNo0",
							label.TraefikFrontendAuthBasicRemoveHeader + "=true",
							label.TraefikFrontendAuthBasicUsers + "=test:$apr1$H6uskkkW$IgXLP6ewTrSuBkTrqE8wj/,test2:$apr1$d9hr9HBB$4HxwgUir3HP4EsggP/QNo0",
							label.TraefikFrontendAuthBasicUsersFile + "=.htpasswd",
							label.TraefikFrontendAuthDigestRemoveHeader + "=true",
							label.TraefikFrontendAuthDigestUsers + "=test:$apr1$H6uskkkW$IgXLP6ewTrSuBkTrqE8wj/,test2:$apr1$d9hr9HBB$4HxwgUir3HP4EsggP/QNo0",
							label.TraefikFrontendAuthDigestUsersFile + "=.htpasswd",
							label.TraefikFrontendAuthForwardAddress + "=auth.server",
							label.TraefikFrontendAuthForwardTrustForwardHeader + "=true",
							label.TraefikFrontendAuthForwardTLSCa + "=ca.crt",
							label.TraefikFrontendAuthForwardTLSCaOptional + "=true",
							label.TraefikFrontendAuthForwardTLSCert + "=server.crt",
							label.TraefikFrontendAuthForwardTLSKey + "=server.key",
							label.TraefikFrontendAuthForwardTLSInsecureSkipVerify + "=true",
							label.TraefikFrontendAuthHeaderField + "=X-WebAuth-User",

							label.TraefikFrontendEntryPoints + "=http,https",
							label.TraefikFrontendPassHostHeader + "=true",
							label.TraefikFrontendPassTLSCert + "=true",
							label.TraefikFrontendPriority + "=666",
							label.TraefikFrontendRedirectEntryPoint + "=https",
							label.TraefikFrontendRedirectRegex + "=nope",
							label.TraefikFrontendRedirectReplacement + "=nope",
							label.TraefikFrontendRedirectPermanent + "=true",
							label.TraefikFrontendRule + "=Host:traefik.io",
							label.TraefikFrontendWhiteListSourceRange + "=10.10.10.10",
							label.TraefikFrontendWhiteListIPStrategyExcludedIPS + "=10.10.10.10,10.10.10.11",
							label.TraefikFrontendWhiteListIPStrategyDepth + "=5",

							label.TraefikFrontendRequestHeaders + "=Access-Control-Allow-Methods:POST,GET,OPTIONS || Content-type: application/json; charset=utf-8",
							label.TraefikFrontendResponseHeaders + "=Access-Control-Allow-Methods:POST,GET,OPTIONS || Content-type: application/json; charset=utf-8",
							label.TraefikFrontendSSLProxyHeaders + "=Access-Control-Allow-Methods:POST,GET,OPTIONS || Content-type: application/json; charset=utf-8",
							label.TraefikFrontendAllowedHosts + "=foo,bar,bor",
							label.TraefikFrontendHostsProxyHeaders + "=foo,bar,bor",
							label.TraefikFrontendSSLHost + "=foo",
							label.TraefikFrontendCustomFrameOptionsValue + "=foo",
							label.TraefikFrontendContentSecurityPolicy + "=foo",
							label.TraefikFrontendPublicKey + "=foo",
							label.TraefikFrontendReferrerPolicy + "=foo",
							label.TraefikFrontendCustomBrowserXSSValue + "=foo",
							label.TraefikFrontendSTSSeconds + "=666",
							label.TraefikFrontendSSLForceHost + "=true",
							label.TraefikFrontendSSLRedirect + "=true",
							label.TraefikFrontendSSLTemporaryRedirect + "=true",
							label.TraefikFrontendSTSIncludeSubdomains + "=true",
							label.TraefikFrontendSTSPreload + "=true",
							label.TraefikFrontendForceSTSHeader + "=true",
							label.TraefikFrontendFrameDeny + "=true",
							label.TraefikFrontendContentTypeNosniff + "=true",
							label.TraefikFrontendBrowserXSSFilter + "=true",
							label.TraefikFrontendIsDevelopment + "=true",

							label.Prefix + label.BaseFrontendErrorPage + "foo." + label.SuffixErrorPageStatus + "=404",
							label.Prefix + label.BaseFrontendErrorPage + "foo." + label.SuffixErrorPageBackend + "=foobar",
							label.Prefix + label.BaseFrontendErrorPage + "foo." + label.SuffixErrorPageQuery + "=foo_query",
							label.Prefix + label.BaseFrontendErrorPage + "bar." + label.SuffixErrorPageStatus + "=500,600",
							label.Prefix + label.BaseFrontendErrorPage + "bar." + label.SuffixErrorPageBackend + "=foobar",
							label.Prefix + label.BaseFrontendErrorPage + "bar." + label.SuffixErrorPageQuery + "=bar_query",

							label.TraefikFrontendRateLimitExtractorFunc + "=client.ip",
							label.Prefix + label.BaseFrontendRateLimit + "foo." + label.SuffixRateLimitPeriod + "=6",
							label.Prefix + label.BaseFrontendRateLimit + "foo." + label.SuffixRateLimitAverage + "=12",
							label.Prefix + label.BaseFrontendRateLimit + "foo." + label.SuffixRateLimitBurst + "=18",
							label.Prefix + label.BaseFrontendRateLimit + "bar." + label.SuffixRateLimitPeriod + "=3",
							label.Prefix + label.BaseFrontendRateLimit + "bar." + label.SuffixRateLimitAverage + "=6",
							label.Prefix + label.BaseFrontendRateLimit + "bar." + label.SuffixRateLimitBurst + "=9",
						},
					},
					Nodes: []*api.ServiceEntry{
						{
							Service: &api.AgentService{
								Service: "test",
								Address: "10.0.0.1",
								Port:    80,
								Tags: []string{
									label.TraefikProtocol + "=https",
									label.TraefikWeight + "=12",
								},
							},
							Node: &api.Node{
								Node:    "localhost",
								Address: "127.0.0.1",
							},
						},
						{
							Service: &api.AgentService{
								Service: "test",
								Address: "10.0.0.2",
								Port:    80,
								Tags: []string{
									label.TraefikProtocol + "=https",
									label.TraefikWeight + "=12",
								},
							},
							Node: &api.Node{
								Node:    "localhost",
								Address: "127.0.0.1",
							},
						},
					},
				},
			},
			expectedFrontends: map[string]*types.Frontend{
				"frontend-test": {
					EntryPoints: []string{
						"http",
						"https",
					},
					Backend: "backend-test",
					Routes: map[string]types.Route{
						"route-host-test": {
							Rule: "Host:traefik.io",
						},
					},
					PassHostHeader: true,
					PassTLSCert:    true,
					Priority:       666,
					PassTLSClientCert: &types.TLSClientHeaders{
						PEM: true,
						Infos: &types.TLSClientCertificateInfos{
							NotBefore: true,
							Sans:      true,
							NotAfter:  true,
							Subject: &types.TLSCLientCertificateSubjectInfos{
								CommonName:   true,
								Country:      true,
								Locality:     true,
								Organization: true,
								Province:     true,
								SerialNumber: true,
							},
						},
					},
					Auth: &types.Auth{
						HeaderField: "X-WebAuth-User",
						Basic: &types.Basic{
							RemoveHeader: true,
							Users: []string{"test:$apr1$H6uskkkW$IgXLP6ewTrSuBkTrqE8wj/",
								"test2:$apr1$d9hr9HBB$4HxwgUir3HP4EsggP/QNo0"},
							UsersFile: ".htpasswd",
						},
					},
					WhiteList: &types.WhiteList{
						SourceRange: []string{
							"10.10.10.10",
						},
						IPStrategy: &types.IPStrategy{
							Depth:       5,
							ExcludedIPs: []string{"10.10.10.10", "10.10.10.11"},
						},
					},
					Headers: &types.Headers{
						CustomRequestHeaders: map[string]string{
							"Access-Control-Allow-Methods": "POST,GET,OPTIONS",
							"Content-Type":                 "application/json; charset=utf-8",
						},
						CustomResponseHeaders: map[string]string{
							"Access-Control-Allow-Methods": "POST,GET,OPTIONS",
							"Content-Type":                 "application/json; charset=utf-8",
						},
						AllowedHosts: []string{
							"foo",
							"bar",
							"bor",
						},
						HostsProxyHeaders: []string{
							"foo",
							"bar",
							"bor",
						},
						SSLRedirect:          true,
						SSLTemporaryRedirect: true,
						SSLForceHost:         true,
						SSLHost:              "foo",
						SSLProxyHeaders: map[string]string{
							"Access-Control-Allow-Methods": "POST,GET,OPTIONS",
							"Content-Type":                 "application/json; charset=utf-8",
						},
						STSSeconds:              666,
						STSIncludeSubdomains:    true,
						STSPreload:              true,
						ForceSTSHeader:          true,
						FrameDeny:               true,
						CustomFrameOptionsValue: "foo",
						ContentTypeNosniff:      true,
						BrowserXSSFilter:        true,
						CustomBrowserXSSValue:   "foo",
						ContentSecurityPolicy:   "foo",
						PublicKey:               "foo",
						ReferrerPolicy:          "foo",
						IsDevelopment:           true,
					},
					Errors: map[string]*types.ErrorPage{
						"foo": {
							Status:  []string{"404"},
							Query:   "foo_query",
							Backend: "backend-foobar",
						},
						"bar": {
							Status:  []string{"500", "600"},
							Query:   "bar_query",
							Backend: "backend-foobar",
						},
					},
					RateLimit: &types.RateLimit{
						ExtractorFunc: "client.ip",
						RateSet: map[string]*types.Rate{
							"foo": {
								Period:  parse.Duration(6 * time.Second),
								Average: 12,
								Burst:   18,
							},
							"bar": {
								Period:  parse.Duration(3 * time.Second),
								Average: 6,
								Burst:   9,
							},
						},
					},
					Redirect: &types.Redirect{
						EntryPoint:  "https",
						Regex:       "",
						Replacement: "",
						Permanent:   true,
					},
				},
			},
			expectedBackends: map[string]*types.Backend{
				"backend-test": {
					Servers: map[string]types.Server{
						"test-0-N753CZ-JEP1SmRf5Wfe6S3-RuM": {
							URL:    "https://10.0.0.1:80",
							Weight: 12,
						},
						"test-1-u4RAIw2K4-PDJh41dqqB4kM2wy0": {
							URL:    "https://10.0.0.2:80",
							Weight: 12,
						},
					},
					CircuitBreaker: &types.CircuitBreaker{
						Expression: "NetworkErrorRatio() > 0.5",
					},
					LoadBalancer: &types.LoadBalancer{
						Method: "drr",
						Stickiness: &types.Stickiness{
							CookieName: "chocolate",
						},
					},
					MaxConn: &types.MaxConn{
						Amount:        666,
						ExtractorFunc: "client.ip",
					},
					HealthCheck: &types.HealthCheck{
						Scheme:   "http",
						Path:     "/health",
						Port:     880,
						Interval: "6",
						Timeout:  "3",
						Hostname: "foo.com",
						Headers: map[string]string{
							"Foo": "bar",
							"Bar": "foo",
						},
					},
					Buffering: &types.Buffering{
						MaxResponseBodyBytes: 10485760,
						MemResponseBodyBytes: 2097152,
						MaxRequestBodyBytes:  10485760,
						MemRequestBodyBytes:  2097152,
						RetryExpression:      "IsNetworkError() && Attempts() <= 2",
					},
				},
			},
		},
		{
			desc: "Should build config containing one frontend, one IPv4 and one IPv6 backend",
			nodes: []catalogUpdate{
				{
					Service: &serviceUpdate{
						ServiceName: "test",
						Attributes: []string{
							"random.foo=bar",
							label.TraefikBackendLoadBalancerMethod + "=drr",
							label.TraefikBackendCircuitBreakerExpression + "=NetworkErrorRatio() > 0.5",
							label.TraefikBackendMaxConnAmount + "=1000",
							label.TraefikBackendMaxConnExtractorFunc + "=client.ip",
							label.TraefikFrontendAuthBasicUsers + "=test:$apr1$H6uskkkW$IgXLP6ewTrSuBkTrqE8wj/,test2:$apr1$d9hr9HBB$4HxwgUir3HP4EsggP/QNo0",
						},
					},
					Nodes: []*api.ServiceEntry{
						{
							Service: &api.AgentService{
								Service: "test",
								Address: "127.0.0.1",
								Port:    80,
								Tags: []string{
									"random.foo=bar",
									label.TraefikWeight + "=42",
									label.TraefikFrontendPassHostHeader + "=true",
									label.TraefikProtocol + "=https",
								},
							},
							Node: &api.Node{
								Node:    "localhost",
								Address: "127.0.0.1",
							},
						},
						{
							Service: &api.AgentService{
								Service: "test",
								Address: "::1",
								Port:    80,
								Tags: []string{
									"random.foo=bar",
									label.TraefikWeight + "=42",
									label.TraefikFrontendPassHostHeader + "=true",
									label.TraefikProtocol + "=https",
								},
							},
							Node: &api.Node{
								Node:    "localhost",
								Address: "::1",
							},
						},
					},
				},
			},
			expectedFrontends: map[string]*types.Frontend{
				"frontend-test": {
					Backend:        "backend-test",
					PassHostHeader: true,
					Routes: map[string]types.Route{
						"route-host-test": {
							Rule: "Host:test.localhost",
						},
					},
					Auth: &types.Auth{
						Basic: &types.Basic{
							Users: []string{"test:$apr1$H6uskkkW$IgXLP6ewTrSuBkTrqE8wj/",
								"test2:$apr1$d9hr9HBB$4HxwgUir3HP4EsggP/QNo0"},
						},
					},
					EntryPoints: []string{},
				},
			},
			expectedBackends: map[string]*types.Backend{
				"backend-test": {
					Servers: map[string]types.Server{
						"test-0-ecTTsmX1vPktQQrl53WhNDy-HEg": {
							URL:    "https://127.0.0.1:80",
							Weight: 42,
						},
						"test-1-9tI2Ud3Vkl4T4B6bAIWV0vFjEIg": {
							URL:    "https://[::1]:80",
							Weight: 42,
						},
					},
					LoadBalancer: &types.LoadBalancer{
						Method: "drr",
					},
					CircuitBreaker: &types.CircuitBreaker{
						Expression: "NetworkErrorRatio() > 0.5",
					},
					MaxConn: &types.MaxConn{
						Amount:        1000,
						ExtractorFunc: "client.ip",
					},
				},
			},
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			nodes := fakeLoadTraefikLabelsSlice(test.nodes, p.Prefix)

			actualConfig := p.buildConfiguration(nodes)
			assert.NotNil(t, actualConfig)
			assert.Equal(t, test.expectedBackends, actualConfig.Backends)
			assert.Equal(t, test.expectedFrontends, actualConfig.Frontends)
		})
	}
}

func TestGetTag(t *testing.T) {
	testCases := []struct {
		desc         string
		tags         []string
		key          string
		defaultValue string
		expected     string
	}{
		{
			desc: "Should return value of foo.bar key",
			tags: []string{
				"foo.bar=random",
				"traefik.backend.weight=42",
				"management",
			},
			key:          "foo.bar",
			defaultValue: "0",
			expected:     "random",
		},
		{
			desc: "Should return default value when nonexistent key",
			tags: []string{
				"foo.bar.foo.bar=random",
				"traefik.backend.weight=42",
				"management",
			},
			key:          "foo.bar",
			defaultValue: "0",
			expected:     "0",
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			actual := getTag(test.key, test.tags, test.defaultValue)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestHasTag(t *testing.T) {
	testCases := []struct {
		desc     string
		name     string
		tags     []string
		expected bool
	}{
		{
			desc:     "tag without value",
			name:     "foo",
			tags:     []string{"foo"},
			expected: true,
		},
		{
			desc:     "tag with value",
			name:     "foo",
			tags:     []string{"foo=true"},
			expected: true,
		},
		{
			desc:     "missing tag",
			name:     "foo",
			tags:     []string{"foobar=true"},
			expected: false,
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			actual := hasTag(test.name, test.tags)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestProviderGetPrefixedName(t *testing.T) {
	testCases := []struct {
		desc     string
		name     string
		prefix   string
		expected string
	}{
		{
			desc:     "empty name with prefix",
			name:     "",
			prefix:   "foo",
			expected: "",
		},
		{
			desc:     "empty name without prefix",
			name:     "",
			prefix:   "",
			expected: "",
		},
		{
			desc:     "with prefix",
			name:     "bar",
			prefix:   "foo",
			expected: "foo.bar",
		},
		{
			desc:     "without prefix",
			name:     "bar",
			prefix:   "",
			expected: "bar",
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			p := &Provider{Prefix: test.prefix}

			actual := p.getPrefixedName(test.name)
			assert.Equal(t, test.expected, actual)
		})
	}

}

func TestProviderGetAttribute(t *testing.T) {
	testCases := []struct {
		desc         string
		tags         []string
		key          string
		defaultValue string
		prefix       string
		expected     string
	}{
		{
			desc:   "Should return tag value 42",
			prefix: "traefik",
			tags: []string{
				"foo.bar=ramdom",
				"traefik.backend.weight=42",
			},
			key:          "backend.weight",
			defaultValue: "0",
			expected:     "42",
		},
		{
			desc:   "Should return tag default value 0",
			prefix: "traefik",
			tags: []string{
				"foo.bar=ramdom",
				"traefik.backend.wei=42",
			},
			key:          "backend.weight",
			defaultValue: "0",
			expected:     "0",
		},
		{
			desc: "Should return tag value 42 when empty prefix",
			tags: []string{
				"foo.bar=ramdom",
				"backend.weight=42",
			},
			key:          "backend.weight",
			defaultValue: "0",
			expected:     "42",
		},
		{
			desc: "Should return default value 0 when empty prefix",
			tags: []string{
				"foo.bar=ramdom",
				"backend.wei=42",
			},
			key:          "backend.weight",
			defaultValue: "0",
			expected:     "0",
		},
		{
			desc: "Should return for.bar key value random when empty prefix",
			tags: []string{
				"foo.bar=ramdom",
				"backend.wei=42",
			},
			key:          "foo.bar",
			defaultValue: "random",
			expected:     "ramdom",
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			p := &Provider{
				Domain: "localhost",
				Prefix: test.prefix,
			}

			actual := p.getAttribute(test.key, test.tags, test.defaultValue)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestProviderGetFrontendRule(t *testing.T) {
	testCases := []struct {
		desc     string
		service  serviceUpdate
		expected string
	}{
		{
			desc: "Should return default host foo.localhost",
			service: serviceUpdate{
				ServiceName: "foo",
				Attributes:  []string{},
			},
			expected: "Host:foo.localhost",
		},
		{
			desc: "Should return host *.example.com",
			service: serviceUpdate{
				ServiceName: "foo",
				Attributes: []string{
					"traefik.frontend.rule=Host:*.example.com",
				},
			},
			expected: "Host:*.example.com",
		},
		{
			desc: "Should return host foo.example.com",
			service: serviceUpdate{
				ServiceName: "foo",
				Attributes: []string{
					"traefik.frontend.rule=Host:{{.ServiceName}}.example.com",
				},
			},
			expected: "Host:foo.example.com",
		},
		{
			desc: "Should return path prefix /bar",
			service: serviceUpdate{
				ServiceName: "foo",
				Attributes: []string{
					"traefik.frontend.rule=PathPrefix:{{getTag \"contextPath\" .Attributes \"/\"}}",
					"contextPath=/bar",
				},
			},
			expected: "PathPrefix:/bar",
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			p := &Provider{
				Domain:               "localhost",
				Prefix:               "traefik",
				FrontEndRule:         "Host:{{.ServiceName}}.{{.Domain}}",
				frontEndRuleTemplate: template.New("consul catalog frontend rule"),
			}
			p.setupFrontEndRuleTemplate()

			labels := tagsToNeutralLabels(test.service.Attributes, p.Prefix)
			test.service.TraefikLabels = labels

			actual := p.getFrontendRule(test.service)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestGetBackendAddress(t *testing.T) {
	testCases := []struct {
		desc     string
		node     *api.ServiceEntry
		expected string
	}{
		{
			desc: "Should return the address of the service",
			node: &api.ServiceEntry{
				Node: &api.Node{
					Address: "10.1.0.1",
				},
				Service: &api.AgentService{
					Address: "10.2.0.1",
				},
			},
			expected: "10.2.0.1",
		},
		{
			desc: "Should return the address of the node",
			node: &api.ServiceEntry{
				Node: &api.Node{
					Address: "10.1.0.1",
				},
				Service: &api.AgentService{
					Address: "",
				},
			},
			expected: "10.1.0.1",
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			actual := getBackendAddress(test.node)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestGetServerName(t *testing.T) {
	testCases := []struct {
		desc     string
		node     *api.ServiceEntry
		expected string
	}{
		{
			desc: "Should create backend name without tags",
			node: &api.ServiceEntry{
				Service: &api.AgentService{
					Service: "api",
					Address: "10.0.0.1",
					Port:    80,
					Tags:    []string{},
				},
			},
			expected: "api-0-eUSiqD6uNvvh6zxsY-OeRi8ZbaE",
		},
		{
			desc: "Should create backend name with multiple tags",
			node: &api.ServiceEntry{
				Service: &api.AgentService{
					Service: "api",
					Address: "10.0.0.1",
					Port:    80,
					Tags:    []string{"traefik.weight=42", "traefik.enable=true"},
				},
			},
			expected: "api-1-eJ8MR2JxjXyZgs1bhurVa0-9OI8",
		},
		{
			desc: "Should create backend name with one tag",
			node: &api.ServiceEntry{
				Service: &api.AgentService{
					Service: "api",
					Address: "10.0.0.1",
					Port:    80,
					Tags:    []string{"a funny looking tag"},
				},
			},
			expected: "api-2-lMCDCsG7sh0SCXOHo4oBOQB-9D4",
		},
	}

	for i, test := range testCases {
		test := test
		i := i
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			actual := getServerName(test.node, i)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func fakeLoadTraefikLabelsSlice(nodes []catalogUpdate, prefix string) []catalogUpdate {
	var result []catalogUpdate

	for _, node := range nodes {
		labels := tagsToNeutralLabels(node.Service.Attributes, prefix)
		node.Service.TraefikLabels = labels
		result = append(result, node)
	}

	return result
}
