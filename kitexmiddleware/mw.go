// Copyright 2021 CloudWeGo Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package kitemiddleware

import (
	"context"
	"fmt"

	"github.com/EICHI-X/ptools/env"
	"github.com/EICHI-X/ptools/logs"
	"github.com/EICHI-X/ptools/pmodel"
	"github.com/EICHI-X/ptools/wtrace"
	"github.com/bytedance/gopkg/cloud/metainfo"
	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/pkg/endpoint"
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	xds2 "github.com/cloudwego/kitex/pkg/xds"
	"github.com/cloudwego/kitex/transport"
	"github.com/google/uuid"
	"github.com/kitex-contrib/obs-opentelemetry/tracing"
	"github.com/kitex-contrib/xds/xdssuite"
	"go.opentelemetry.io/otel/trace"
)

var _ endpoint.Middleware = CommonMiddleware

type args interface {
	GetFirstArgument() interface{}
}

type result interface {
	GetResult() interface{}
}

func CommonMiddleware(next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, req, resp interface{}) (err error) {
		ri := rpcinfo.GetRPCInfo(ctx)
		// get real request
		logs.CtxInfof(ctx, "real request: %+v\n", req.(args).GetFirstArgument())
		// get local service information
		logs.CtxInfof(ctx, "local service name: %v\n", ri.From().ServiceName())
		// get remote service information

		logs.CtxInfof(ctx, "remote service name: %v, remote method: %v\n", ri.To().ServiceName(), ri.To().Method())
		if err := next(ctx, req, resp); err != nil {
			return err
		}
		// get real response
		logs.CtxInfof(ctx, "real response: %+v\n", resp.(result).GetResult())
		return nil
	}
}

func ClientMiddleware(next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, req, resp interface{}) (err error) {
		ri := rpcinfo.GetRPCInfo(ctx)
		// get timeout information
		traceIDKey := string(pmodel.TraceIdLogKey)
		// ctx = metainfo.WithValue(ctx, "temp", "temp-value")       // only present in next service
		traceID, ok := metainfo.GetPersistentValue(ctx, traceIDKey)
		if !ok || traceID == "" {
			traceIdUid, _ := uuid.NewUUID()
			traceID = traceIdUid.String()

		}
		ctx = metainfo.WithPersistentValue(ctx, traceIDKey, traceID) // will present in the next service and its successors

		logs.CtxInfof(ctx, "rpc timeout: %v, readwrite timeout: %v\n", ri.Config().RPCTimeout(), ri.Config().ConnectTimeout())
		if err := next(ctx, req, resp); err != nil {
			return err
		}
		// get server information
		logs.CtxInfof(ctx, "server address: %v\n", ri.To().Address())
		return nil
	}
}

func ServerMiddleware(next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, req, resp interface{}) (err error) {
		ri := rpcinfo.GetRPCInfo(ctx)
		// get client information
		// traceIDKey := string(pmodel.TraceIdLogKey)
		// ctx = metainfo.WithValue(ctx, "temp", "temp-value")       // only present in next service
		// traceID, ok := metainfo.GetPersistentValue(ctx, traceIDKey)
		// if !ok || traceID == "" {
		// 	traceIdUid, _ := uuid.NewUUID()
		// 	traceID = traceIdUid.String()
		// 	ctx = metainfo.WithPersistentValue(ctx, traceIDKey, traceID) // will present in the next service and its successors

		// }
		spanCtx := trace.SpanContextFromContext(ctx)
		if !spanCtx.IsValid() || !spanCtx.HasTraceID() {
			ctx, spanCtx = wtrace.InitSpanToContext(ctx, []byte{})
			// traceID,_ := wtrace.DefaultIDGenerator().NewIDs(c)
			// traceIdUid, _ := uuid.NewUUID()
			// reqId = traceIdUid.String()

			// c = context.WithValue(c, pmodel.TraceIDKey, spanCtx.TraceID())
		}
		logs.CtxInfof(ctx, "client address: %v\n", ri.From().Address())
		if err := next(ctx, req, resp); err != nil {
			return err
		}
		return nil
	}
}

func PassHeaderFunc(ctx context.Context) map[string]string {
	r := make(map[string]string)
	for idx := range pmodel.PassHeaderKeys {
		key := pmodel.PassHeaderKeys[idx]
		if v, ok := metainfo.GetPersistentValue(ctx, key); ok {
			r[key] = v
			// logs.CtxInfof(ctx, "PassHeaderKeys key:%v v:%v", key, v)
		}
	}

	return r
}
func ClientDefaultOptions() []client.Option {

	r := []client.Option{
		client.WithXDSSuite(xds2.ClientSuite{
			RouterMiddleware: xdssuite.NewXDSRouterMiddleware(
				xdssuite.WithRouterMetaExtractor(PassHeaderFunc),
			),
			Resolver: xdssuite.NewXDSResolver(),
		}),
		client.WithTransportProtocol(transport.TTHeader),
		client.WithSuite(tracing.NewClientSuite()),
		// Please keep the same as provider.WithServiceName
		client.WithClientBasicInfo(&rpcinfo.EndpointBasicInfo{ServiceName: env.Instance().ServiceName}),
		// client.WithRPCTimeout(1*time.Second),
	}
	return r
}

const (
	Namespace = "prod"

	Suffix            = "svc.cluster.local"
	ClientServicePort = "8888"
	ServerServicePort = "80"

	POD_NAMESPACE_KEY = "POD_NAMESPACE"
)

func ServiceName(ServerSvc string) string {
	return fmt.Sprintf("%s.%s.%s:%s", ServerSvc, Namespace, Suffix, ClientServicePort)
}
