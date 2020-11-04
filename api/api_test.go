package api

import (
	"encoding/json"
	"fmt"
	"github.com/ClusterLabs/hawk-apiserver/api"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestGetMethods(t *testing.T) {

	cibFile1 := "test/cib2.xml"
	cibFile2 := "test/bundles_cib.xml"
	crmMonFile1 := "test/crm-mon-1.xml"
	allTestcases := []struct {
		name             string
		api              string
		path             string
		expectedResponse string
	}{
		{
			name: "Test /api/v1/configuration/cluster_property",
			api:  "configuration/cluster_property",
			path: cibFile1,
			expectedResponse: `
{
   "cluster-infrastructure":"corosync",
   "cluster-name":"hawkdev",
   "dc-version":"2.0.0+20190125.788ee2c49-lp150.326.2-2.0.0+20190125.788ee2c49",
   "have-watchdog":"true",
   "placement-strategy":"balanced",
   "stonith-enabled":"true"
}
`,
		},
		{
			name: "Test /api/v1/configuration/rsc_defaults",
			api:  "configuration/rsc_defaults",
			path: cibFile1,
			expectedResponse: `
{
   "migration-threshold":"3",
   "resource-stickiness":"1"
}
`,
		},
		{
			name: "Test /api/v1/configuration/op_defaults",
			api:  "configuration/op_defaults",
			path: cibFile1,
			expectedResponse: `
{
   "record-pending":"true",
   "timeout":"600"
}
`,
		},
		{
			name: "Test /api/v1/configuration/resources",
			api:  "configuration/resources",
			path: cibFile1,
			expectedResponse: `
[
   {
      "id":"base-clone",
      "type":"clone"
   },
   {
      "id":"c-clusterfs",
      "type":"clone"
   },
   {
      "id":"cl-servers",
      "type":"clone"
   },
   {
      "id":"g-proxy",
      "type":"group"
   },
   {
      "id":"ms-DRBD",
      "type":"master"
   },
   {
      "id":"stonith-sbd",
      "type":"primitive"
   }
]
`,
		},
		{
			name: "Test /api/v1/configuration/resources/:id",
			api:  "configuration/resources/stonith-sbd",
			path: cibFile1,
			expectedResponse: `
{
   "id":"stonith-sbd",
   "class":"stonith",
   "type":"external/sbd",
   "param":{
      "pcmk_delay_max":"30s"
   }
}
`,
		},
		{
			name: "Test /api/v1/configuration/primitives",
			api:  "configuration/primitives",
			path: cibFile1,
			expectedResponse: `
[
   {
      "id":"stonith-sbd",
      "class":"stonith",
      "type":"external/sbd",
      "param":{
         "pcmk_delay_max":"30s"
      }
   }
]
`,
		},
		{
			name: "Test /api/v1/configuration/primitives/:id",
			api:  "configuration/primitives/stonith-sbd",
			path: cibFile1,
			expectedResponse: `
{
   "id":"stonith-sbd",
   "class":"stonith",
   "type":"external/sbd",
   "param":{
      "pcmk_delay_max":"30s"
   }
}
`,
		},
		{
			name: "Test /api/v1/configuration/groups",
			api:  "configuration/groups",
			path: cibFile1,
			expectedResponse: `
[
   {
      "id":"g-proxy",
      "primitives":[
         {
            "id":"proxy-vip",
            "class":"ocf",
            "type":"IPaddr2",
            "provider":"heartbeat",
            "param":{
               "ip":"10.13.37.13"
            }
         },
         {
            "id":"proxy",
            "class":"systemd",
            "type":"haproxy",
            "op":[
               {
                  "id":"proxy-monitor-10s",
                  "interval":"10s",
                  "name":"monitor"
               }
            ]
         }
      ]
   }
]
`,
		},
		{
			name: "Test /api/v1/configuration/groups/:id",
			api:  "configuration/groups/g-proxy",
			path: cibFile1,
			expectedResponse: `
{
   "id":"g-proxy",
   "primitives":[
      {
         "id":"proxy-vip",
         "class":"ocf",
         "type":"IPaddr2",
         "provider":"heartbeat",
         "param":{
            "ip":"10.13.37.13"
         }
      },
      {
         "id":"proxy",
         "class":"systemd",
         "type":"haproxy",
         "op":[
            {
               "id":"proxy-monitor-10s",
               "interval":"10s",
               "name":"monitor"
            }
         ]
      }
   ]
}
`,
		},
		{
			name: "Test /api/v1/configuration/groups/:id/:primitiveId",
			api:  "configuration/groups/g-proxy/proxy",
			path: cibFile1,
			expectedResponse: `
{
   "id":"proxy",
   "class":"systemd",
   "type":"haproxy",
   "op":[
      {
         "id":"proxy-monitor-10s",
         "interval":"10s",
         "name":"monitor"
      }
   ]
}
`,
		},
		{
			name: "Test /api/v1/configuration/masters",
			api:  "configuration/masters",
			path: cibFile1,
			expectedResponse: `
[
   {
      "id":"ms-DRBD",
      "meta":{
         "clone-max":"2",
         "clone-node-max":"1",
         "master-max":"1",
         "master-node-max":"1",
         "notify":"true"
      },
      "primitive":{
         "id":"DRBD",
         "class":"ocf",
         "type":"drbd",
         "provider":"linbit",
         "param":{
            "drbd_resource":"r0",
            "drbdconf":"/etc/drbd.conf"
         },
         "op":[
            {
               "id":"DRBD-monitor-29s",
               "interval":"29s",
               "name":"monitor",
               "role":"Master"
            },
            {
               "id":"DRBD-monitor-31s",
               "interval":"31s",
               "name":"monitor",
               "role":"Slave"
            }
         ]
      }
   }
]
`,
		},
		{
			name: "Test /api/v1/configuration/masters/:id",
			api:  "configuration/masters/ms-DRBD",
			path: cibFile1,
			expectedResponse: `
{
   "id":"ms-DRBD",
   "meta":{
      "clone-max":"2",
      "clone-node-max":"1",
      "master-max":"1",
      "master-node-max":"1",
      "notify":"true"
   },
   "primitive":{
      "id":"DRBD",
      "class":"ocf",
      "type":"drbd",
      "provider":"linbit",
      "param":{
         "drbd_resource":"r0",
         "drbdconf":"/etc/drbd.conf"
      },
      "op":[
         {
            "id":"DRBD-monitor-29s",
            "interval":"29s",
            "name":"monitor",
            "role":"Master"
         },
         {
            "id":"DRBD-monitor-31s",
            "interval":"31s",
            "name":"monitor",
            "role":"Slave"
         }
      ]
   }
}
`,
		},
		{
			name: "Test /api/v1/configuration/clones",
			api:  "configuration/clones",
			path: cibFile1,
			expectedResponse: `
[
   {
      "id":"base-clone",
      "meta":{
         "interleave":"true"
      },
      "primitive":{
         "id":"dlm",
         "class":"ocf",
         "type":"controld",
         "provider":"pacemaker",
         "op":[
            {
               "id":"dlm-start-0",
               "interval":"0",
               "name":"start",
               "timeout":"90"
            },
            {
               "id":"dlm-stop-0",
               "interval":"0",
               "name":"stop",
               "timeout":"100"
            },
            {
               "id":"dlm-monitor-60",
               "interval":"60",
               "name":"monitor",
               "timeout":"60"
            }
         ]
      }
   },
   {
      "id":"c-clusterfs",
      "meta":{
         "clone-max":"8",
         "interleave":"true",
         "target-role":"Started"
      },
      "primitive":{
         "id":"clusterfs",
         "class":"ocf",
         "type":"Filesystem",
         "provider":"heartbeat",
         "param":{
            "device":"/dev/vdb2",
            "directory":"/srv/clusterfs",
            "fstype":"ocfs2"
         },
         "op":[
            {
               "id":"clusterfs-monitor-20",
               "interval":"20",
               "name":"monitor",
               "timeout":"40"
            },
            {
               "id":"clusterfs-start-0",
               "interval":"0",
               "name":"start",
               "timeout":"60"
            },
            {
               "id":"clusterfs-stop-0",
               "interval":"0",
               "name":"stop",
               "timeout":"60"
            }
         ]
      }
   },
   {
      "id":"cl-servers",
      "meta":{
         "clone-max":"2",
         "clone-node-max":"1",
         "globally-unique":"false"
      },
      "primitive":{
         "id":"server-instance",
         "class":"",
         "type":""
      }
   }
]
`,
		},
		{
			name: "Test /api/v1/configuration/clones/:id",
			api:  "configuration/clones/base-clone",
			path: cibFile1,
			expectedResponse: `
{
   "id":"base-clone",
   "meta":{
      "interleave":"true"
   },
   "primitive":{
      "id":"dlm",
      "class":"ocf",
      "type":"controld",
      "provider":"pacemaker",
      "op":[
         {
            "id":"dlm-start-0",
            "interval":"0",
            "name":"start",
            "timeout":"90"
         },
         {
            "id":"dlm-stop-0",
            "interval":"0",
            "name":"stop",
            "timeout":"100"
         },
         {
            "id":"dlm-monitor-60",
            "interval":"60",
            "name":"monitor",
            "timeout":"60"
         }
      ]
   }
}
`,
		},
		{
			name: "Test /api/v1/configuration/bundles",
			api:  "configuration/bundles",
			path: cibFile2,
			expectedResponse: `
[
   {
      "id":"httpd-bundle",
      "container":{
         "image":"localhost/pcmktest:http",
         "replicas":"3",
         "type":"podman"
      },
      "network":{
         "host-interface":"eth0",
         "host-netmask":"24",
         "ip-range-start":"192.168.122.131",
         "network":[
            {
               "id":"httpd-port",
               "port":"80"
            }
         ]
      },
      "storage-mapping":[
         {
            "id":"httpd-root",
            "options":"rw",
            "source-dir-root":"/var/local/containers",
            "target-dir":"/var/www/html"
         },
         {
            "id":"httpd-logs",
            "options":"rw",
            "source-dir-root":"/var/log/pacemaker/bundles",
            "target-dir":"/etc/httpd/logs"
         }
      ],
      "primitive":{
         "id":"httpd",
         "class":"ocf",
         "type":"apache",
         "provider":"heartbeat",
         "param":{
            "statusurl":"http://localhost/server-status"
         },
         "op":[
            {
               "id":"httpd-monitor",
               "interval":"30s",
               "name":"monitor"
            }
         ]
      }
   }
]
`,
		},
		{
			name: "Test /api/v1/configuration/bundles/:id",
			api:  "configuration/bundles/httpd-bundle",
			path: cibFile2,
			expectedResponse: `
{
   "id":"httpd-bundle",
   "container":{
      "image":"localhost/pcmktest:http",
      "replicas":"3",
      "type":"podman"
   },
   "network":{
      "host-interface":"eth0",
      "host-netmask":"24",
      "ip-range-start":"192.168.122.131",
      "network":[
         {
            "id":"httpd-port",
            "port":"80"
         }
      ]
   },
   "storage-mapping":[
      {
         "id":"httpd-root",
         "options":"rw",
         "source-dir-root":"/var/local/containers",
         "target-dir":"/var/www/html"
      },
      {
         "id":"httpd-logs",
         "options":"rw",
         "source-dir-root":"/var/log/pacemaker/bundles",
         "target-dir":"/etc/httpd/logs"
      }
   ],
   "primitive":{
      "id":"httpd",
      "class":"ocf",
      "type":"apache",
      "provider":"heartbeat",
      "param":{
         "statusurl":"http://localhost/server-status"
      },
      "op":[
         {
            "id":"httpd-monitor",
            "interval":"30s",
            "name":"monitor"
         }
      ]
   }
}
`,
		},
		{
			name: "Test /api/v1/configuration/nodes",
			api:  "configuration/nodes",
			path: cibFile1,
			expectedResponse: `
[
   {
      "id":"168633610",
      "uname":"webui"
   },
   {
      "id":"168633611",
      "uname":"node1"
   }
]
`,
		},
		{
			name: "Test /api/v1/configuration/nodes/:id",
			api:  "configuration/nodes/168633610",
			path: cibFile1,
			expectedResponse: `
{
   "id":"168633610",
   "uname":"webui"
}
`,
		},
		{
			name: "Test /api/v1/configuration/constraints",
			api:  "configuration/constraints",
			path: cibFile1,
			expectedResponse: `
[
   {
      "id":"base-then-clusterfs",
      "type":"order"
   },
   {
      "id":"clusterfs-then-servers",
      "type":"order"
   },
   {
      "id":"clusterfs-with-base",
      "type":"colocation"
   },
   {
      "id":"l-proxy-on-webui",
      "type":"location"
   },
   {
      "id":"l-web-on-node1",
      "type":"location"
   },
   {
      "id":"l-web-on-node2",
      "type":"location"
   }
]
`,
		},
		{
			name: "Test /api/v1/configuration/constraints/:id",
			api:  "configuration/constraints/l-web-on-node2",
			path: cibFile1,
			expectedResponse: `
{
   "id":"l-web-on-node2",
   "rsc":"cl-servers",
   "score":"200",
   "node":"node2"
}
`,
		},
		{
			name: "Test /api/v1/configuration/locations",
			api:  "configuration/locations",
			path: cibFile1,
			expectedResponse: `
[
   {
      "id":"l-proxy-on-webui",
      "rsc":"g-proxy",
      "score":"200",
      "node":"webui"
   },
   {
      "id":"l-web-on-node1",
      "rsc":"cl-servers",
      "score":"200",
      "node":"node1"
   },
   {
      "id":"l-web-on-node2",
      "rsc":"cl-servers",
      "score":"200",
      "node":"node2"
   }
]
`,
		},
		{
			name: "Test /api/v1/configuration/locations/:id",
			api:  "configuration/locations/l-web-on-node2",
			path: cibFile1,
			expectedResponse: `
{
   "id":"l-web-on-node2",
   "rsc":"cl-servers",
   "score":"200",
   "node":"node2"
}
`,
		},
		{
			name: "Test /api/v1/configuration/colocations",
			api:  "configuration/colocations",
			path: cibFile1,
			expectedResponse: `
[
   {
      "id":"clusterfs-with-base",
      "score":"INFINITY",
      "rsc":"c-clusterfs",
      "with-rsc":"base-clone"
   }
]
`,
		},
		{
			name: "Test /api/v1/configuration/colocations/:id",
			api:  "configuration/colocations/clusterfs-with-base",
			path: cibFile1,
			expectedResponse: `
{
   "id":"clusterfs-with-base",
   "score":"INFINITY",
   "rsc":"c-clusterfs",
   "with-rsc":"base-clone"
}
`,
		},
		{
			name: "Test /api/v1/configuration/orders",
			api:  "configuration/orders",
			path: cibFile1,
			expectedResponse: `
[
   {
      "id":"base-then-clusterfs",
      "score":"INFINITY",
      "first":"base-clone",
      "then":"c-clusterfs"
   },
   {
      "id":"clusterfs-then-servers",
      "kind":"Mandatory",
      "first":"c-clusterfs",
      "then":"cl-servers"
   }
]
`,
		},
		{
			name: "Test /api/v1/configuration/orders/:id",
			api:  "configuration/orders/base-then-clusterfs",
			path: cibFile1,
			expectedResponse: `
{
   "id":"base-then-clusterfs",
   "score":"INFINITY",
   "first":"base-clone",
   "then":"c-clusterfs"
}
`,
		},
		{
			name: "Test /api/v1/status/summary",
			api:  "status/summary",
			path: crmMonFile1,
			expectedResponse: `
{
   "current_dc_id":"168430211",
   "current_dc_name":"Hawk3-1",
   "current_dc_present":"true",
   "current_dc_version":"2.0.0+20181108.62ffcafbc-1.1-2.0.0+20181108.62ffcafbc",
   "current_dc_with_quorum":"true",
   "last_change_client":"cibadmin",
   "last_change_origin":"Hawk3-2",
   "last_change_time":"Tue Jan 15 22:19:59 2019",
   "last_change_user":"root",
   "last_update_time":"Tue Jan 15 22:20:05 2019",
   "nodes_configured_number":"2",
   "resources_configured_blocked":"0",
   "resources_configured_disabled":"0",
   "resources_configured_number":"3",
   "stack_type":"corosync"
}
`,
		},
		{
			name: "Test /api/v1/status/nodes",
			api:  "status/nodes",
			path: crmMonFile1,
			expectedResponse: `
[
   {
      "name":"Hawk3-1",
      "id":"168430211",
      "online":"true",
      "standby":"false",
      "standby_onfail":"false",
      "maintenance":"false",
      "pending":"false",
      "unclean":"false",
      "shutdown":"false",
      "expected_up":"true",
      "is_dc":"true",
      "resources_running":"1",
      "type":"member"
   },
   {
      "name":"Hawk3-2",
      "id":"168430212",
      "online":"true",
      "standby":"false",
      "standby_onfail":"false",
      "maintenance":"false",
      "pending":"false",
      "unclean":"false",
      "shutdown":"false",
      "expected_up":"true",
      "is_dc":"false",
      "resources_running":"1",
      "type":"member"
   }
]
`,
		},
		{
			name: "Test /api/v1/status/nodes/:id",
			api:  "status/nodes/168430212",
			path: crmMonFile1,
			expectedResponse: `
{
   "name":"Hawk3-2",
   "id":"168430212",
   "online":"true",
   "standby":"false",
   "standby_onfail":"false",
   "maintenance":"false",
   "pending":"false",
   "unclean":"false",
   "shutdown":"false",
   "expected_up":"true",
   "is_dc":"false",
   "resources_running":"1",
   "type":"member"
}
`,
		},
		{
			name: "Test /api/v1/status/resources",
			api:  "status/resources",
			path: crmMonFile1,
			expectedResponse: `
[
   {
      "id":"d1",
      "type":"primitive",
      "resource_agent":"ocf::heartbeat:Dummy",
      "role":"Started",
      "on_node":"Hawk3-1"
   },
   {
      "id":"vip1",
      "type":"primitive",
      "resource_agent":"ocf::heartbeat:IPaddr2",
      "role":"Started",
      "on_node":"Hawk3-2"
   }
]
`,
		},
		{
			name: "Test /api/v1/status/resources/:id",
			api:  "status/resources/vip1",
			path: crmMonFile1,
			expectedResponse: `
{
   "id":"vip1",
   "type":"primitive",
   "resource_agent":"ocf::heartbeat:IPaddr2",
   "role":"Started",
   "on_node":"Hawk3-2"
}
`,
		},
		{
			name: "Test /api/v1/status/failures",
			api:  "status/failures",
			path: crmMonFile1,
			expectedResponse: `
[
   {
      "op_key":"ddd_start_0",
      "node":"Hawk3-2",
      "exitstatus":"not installed",
      "exitreason":"Setup problem: couldn't find command: /usr/bin/safe_mysqld",
      "exitcode":"5",
      "call":"16",
      "status":"complete",
      "last-rc-change":"Tue Jan 15 22:19:59 2019",
      "queued":"0",
      "exec":"34",
      "interval":"0",
      "task":"start"
   },
   {
      "op_key":"ddd_start_0",
      "node":"Hawk3-1",
      "exitstatus":"not installed",
      "exitreason":"Setup problem: couldn't find command: /usr/bin/safe_mysqld",
      "exitcode":"5",
      "call":"15",
      "status":"complete",
      "last-rc-change":"Tue Jan 15 22:19:59 2019",
      "queued":"0",
      "exec":"45",
      "interval":"0",
      "task":"start"
   }
]
`,
		},
		{
			name: "Test /api/v1/status/failures/:node",
			api:  "status/failures/Hawk3-1",
			path: crmMonFile1,
			expectedResponse: `
[
   {
      "op_key":"ddd_start_0",
      "node":"Hawk3-1",
      "exitstatus":"not installed",
      "exitreason":"Setup problem: couldn't find command: /usr/bin/safe_mysqld",
      "exitcode":"5",
      "call":"15",
      "status":"complete",
      "last-rc-change":"Tue Jan 15 22:19:59 2019",
      "queued":"0",
      "exec":"45",
      "interval":"0",
      "task":"start"
   }
]
`,
		},
	}

	for _, testcase := range allTestcases {
		t.Run(testcase.name, func(t *testing.T) {
			request := newGetRequest(testcase.api)
			response := httptest.NewRecorder()

			apiType := strings.Split(testcase.api, "/")[0]
			if apiType == "configuration" {
				api.HandleConfiguration(response, request, getCibContents(testcase.path))
			} else if apiType == "status" {
				api.HandleStatus(response, request, getCibContents(testcase.path))
			}

			assertStatus(t, response.Code, http.StatusOK)
			assertContentType(t, response.Result().Header.Get("Content-Type"), "application/json")
			assertResponseBody(t, response.Body.String(), testcase.expectedResponse)
		})
	}
}

func equalJSON(s1, s2 string) (bool, error) {
	var o1 interface{}
	var o2 interface{}

	var err error
	err = json.Unmarshal([]byte(s1), &o1)
	if err != nil {
		return false, fmt.Errorf("Error mashalling string 1 :: %s", err.Error())
	}
	err = json.Unmarshal([]byte(s2), &o2)
	if err != nil {
		return false, fmt.Errorf("Error mashalling string 2 :: %s", err.Error())
	}

	return reflect.DeepEqual(o1, o2), nil
}

func getCibContents(path string) string {
	data, _ := ioutil.ReadFile(path)
	return string(data)
}

func newGetRequest(name string) *http.Request {
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/%s", name), nil)
	return req
}

func assertStatus(t *testing.T, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("did not get correct status, got %d, want %d", got, want)
	}
}

func assertContentType(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("response Content-Type is wrong, got '%s', want '%s'", got, want)
	}
}

func assertResponseBody(t *testing.T, got, want string) {
	t.Helper()
	res, _ := equalJSON(got, want)
	if !res {
		t.Errorf("response body is wrong, got '\n%s', want '\n%s'", got, want)
	}
}
