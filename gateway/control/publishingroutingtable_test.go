// Copyright 2020 Anapaya Systems
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package control_test

import (
	"net"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/scionproto/scion/gateway/control"
	"github.com/scionproto/scion/gateway/control/mock_control"
	"github.com/scionproto/scion/gateway/pktcls"
	"github.com/scionproto/scion/pkg/private/xtest"
)

var (
	routeSourceIPv4 = net.ParseIP("192.0.100.1")
	routeSourceIPv6 = net.ParseIP("2001:db8:1::1")
)

func getRoutingChains(t *testing.T) ([]*control.RoutingChain, control.Route, control.Route) {
	prefixIPv4 := xtest.MustParseCIDR(t, "192.168.100.0/24")
	prefixIPv6 := xtest.MustParseCIDR(t, "2001:db8:2::/48")
	return []*control.RoutingChain{
			{
				Prefixes: []*net.IPNet{
					prefixIPv4,
					prefixIPv6,
				},
				TrafficMatchers: []control.TrafficMatcher{
					{ID: 1, Matcher: pktcls.CondFalse},
					{ID: 2, Matcher: pktcls.CondFalse},
				},
			},
		}, control.Route{
			Prefix:  prefixIPv4,
			Source:  routeSourceIPv4,
			NextHop: net.IP{},
		}, control.Route{
			Prefix:  prefixIPv6,
			Source:  routeSourceIPv6,
			NextHop: net.IP{},
		}
}

func TestNewPublishingRoutingTableLate(t *testing.T) {
	// Test whether adding/removing routes in active state works.

	chains, routeV4, routeV6 := getRoutingChains(t)

	ctrl := gomock.NewController(t)

	publisher := mock_control.NewMockPublisher(ctrl)
	publisher.EXPECT().AddRoute(routeV4)
	publisher.EXPECT().AddRoute(routeV6)
	publisher.EXPECT().DeleteRoute(routeV4)
	publisher.EXPECT().DeleteRoute(routeV6)
	publisher.EXPECT().Close().Times(1)

	routingTable := mock_control.NewMockRoutingTable(ctrl)
	routingTable.EXPECT().SetSession(1, gomock.Any()).Times(1)
	routingTable.EXPECT().ClearSession(1).Times(1)
	routingTable.EXPECT().Close().Times(1)

	rtw := control.NewPublishingRoutingTable(chains, routingTable, publisher, net.IP{},
		routeSourceIPv4, routeSourceIPv6)

	require.NoError(t, rtw.SetSession(1, testPktWriter{}))
	require.NoError(t, rtw.ClearSession(1))
	require.NoError(t, rtw.Close())
}

func TestNewPublishingRoutingTableHealthiness(t *testing.T) {
	// Make sure that one healthy traffic class is sufficient not to retract the routes.

	chains, routeV4, routeV6 := getRoutingChains(t)

	ctrl := gomock.NewController(t)

	publisher := mock_control.NewMockPublisher(ctrl)
	publisher.EXPECT().AddRoute(routeV4)
	publisher.EXPECT().AddRoute(routeV6)

	routingTable := mock_control.NewMockRoutingTable(ctrl)
	routingTable.EXPECT().SetSession(1, gomock.Any()).Times(1)
	routingTable.EXPECT().SetSession(2, gomock.Any()).Times(1)
	routingTable.EXPECT().ClearSession(1).Times(1)

	rtw := control.NewPublishingRoutingTable(chains, routingTable, publisher, net.IP{},
		routeSourceIPv4, routeSourceIPv6)

	require.NoError(t, rtw.SetSession(1, testPktWriter{}))
	require.NoError(t, rtw.SetSession(2, testPktWriter{}))
	require.NoError(t, rtw.ClearSession(1))
}
