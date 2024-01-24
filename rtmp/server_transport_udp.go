//
// Copyright [2024] [https://github.com/gnolizuh]
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package rtmp

import (
	"context"
	"github.com/panjf2000/ants"
	"net"
	"trpc.group/trpc-go/trpc-go/transport"
)

func (s *serverTransport) serveUDP(ctx context.Context, rwc *net.UDPConn, pool *ants.PoolWithFunc,
	opts *transport.ListenServeOptions) error {
	return nil
}
