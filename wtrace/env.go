package wtrace

import (
	"context"
	"strconv"

	"github.com/EICHI-X/ptools/putils"
	"github.com/bytedance/gopkg/cloud/metainfo"
	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/cloudwego/kitex/pkg/remote"
	"github.com/cloudwego/kitex/pkg/remote/transmeta"
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/pkg/rpcinfo/remoteinfo"
	"github.com/cloudwego/kitex/pkg/utils"
	"github.com/cloudwego/kitex/transport"
	"github.com/google/uuid"
)

/*
 * Copyright 2021 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

const (
	framedTransportType   = "framed"
	unframedTransportType = "unframed"

	// for biz error
	bizStatus  = "biz-status"
	bizMessage = "biz-message"
	bizExtra   = "biz-extra"
)

// TTHeader handlers.
var (
	ServerTTHeaderHandler remote.MetaHandler = &serverTTHeaderHandler{}
)

// serverTTHeaderHandler implement remote.MetaHandler
type serverTTHeaderHandler struct{}

// ReadMeta of serverTTHeaderHandler reads headers of TTHeader protocol to transport
func (sh *serverTTHeaderHandler) ReadMeta(ctx context.Context, msg remote.Message) (context.Context, error) {
	if !isTTHeader(msg) {
		return ctx, nil
	}
	ri := msg.RPCInfo()
	transInfo := msg.TransInfo()
	intInfo := transInfo.TransIntInfo()
	klog.CtxInfof(ctx, "service read msg=%v,transInfo=%v", putils.ToJson(msg), putils.ToJson(transInfo))
	ci := rpcinfo.AsMutableEndpointInfo(ri.From())
	if ci != nil {
		if v := intInfo[transmeta.FromService]; v != "" {
			ci.SetServiceName(v)
		}
		if v := intInfo[transmeta.FromMethod]; v != "" {
			ci.SetMethod(v)
		}
	}
	return ctx, nil
}

// WriteMeta of serverTTHeaderHandler writes headers of TTHeader protocol to transport
func (sh *serverTTHeaderHandler) WriteMeta(ctx context.Context, msg remote.Message) (context.Context, error) {
	if !isTTHeader(msg) {
		return ctx, nil
	}
	ri := msg.RPCInfo()
	transInfo := msg.TransInfo()
	intInfo := transInfo.TransIntInfo()
	strInfo := transInfo.TransStrInfo()
	klog.CtxInfof(ctx, "service write msg=%v,transInfo=%v", putils.ToJson(msg), putils.ToJson(transInfo))
	intInfo[transmeta.MsgType] = strconv.Itoa(int(msg.MessageType()))

	if bizErr := ri.Invocation().BizStatusErr(); bizErr != nil {
		strInfo[bizStatus] = strconv.Itoa(int(bizErr.BizStatusCode()))
		strInfo[bizMessage] = bizErr.BizMessage()
		if len(bizErr.BizExtra()) != 0 {
			strInfo[bizExtra], _ = utils.Map2JSONStr(bizErr.BizExtra())
		}
	}

	return ctx, nil
}

func isTTHeader(msg remote.Message) bool {
	transProto := msg.ProtocolInfo().TransProto
	return transProto&transport.TTHeader == transport.TTHeader
}

var ClientTTHeaderHandler remote.MetaHandler = &clientTTHeaderHandler{}

// clientTTHeaderHandler implement remote.MetaHandler
type clientTTHeaderHandler struct{}

// WriteMeta of clientTTHeaderHandler writes headers of TTHeader protocol to transport
func (ch *clientTTHeaderHandler) WriteMeta(ctx context.Context, msg remote.Message) (context.Context, error) {
	ri := msg.RPCInfo()
	transInfo := msg.TransInfo()
	logID := ""
	vlog := ctx.Value(transmeta.LogID)
	klog.CtxInfof(ctx, "client write msg=%v,transInfo=%v", putils.ToJson(msg), putils.ToJson(transInfo))
	if v, ok := vlog.(string); ok {
		logID = v
	}
	if logID == "" {
		lid, _ := uuid.NewUUID()
		logID = lid.String()
		ctx = context.WithValue(ctx, transmeta.LogID, logID)
	}
	hd := map[uint16]string{
		transmeta.FromService: ri.From().ServiceName(),
		transmeta.FromIDC:     ri.From().DefaultTag(transmeta.HeaderTransToIDC, ""),
		transmeta.FromMethod:  ri.From().Method(),
		transmeta.ToService:   ri.To().ServiceName(),
		transmeta.ToMethod:    ri.To().Method(),
		transmeta.MsgType:     strconv.Itoa(int(msg.MessageType())),
		transmeta.LogID:       logID,
	}
	transInfo.PutTransIntInfo(hd)

	if metainfo.HasMetaInfo(ctx) {
		hd := make(map[string]string)
		metainfo.SaveMetaInfoToMap(ctx, hd)
		transInfo.PutTransStrInfo(hd)
	}

	return ctx, nil
}

// ReadMeta of clientTTHeaderHandler reads headers of TTHeader protocol from transport
func (ch *clientTTHeaderHandler) ReadMeta(ctx context.Context, msg remote.Message) (context.Context, error) {
	ri := msg.RPCInfo()
	remote := remoteinfo.AsRemoteInfo(ri.To())
	klog.CtxInfof(ctx, "client read msg=%v,transInfo=%v", putils.ToJson(msg), putils.ToJson(ctx))
	if remote == nil {
		return ctx, nil
	}

	transInfo := msg.TransInfo()
	strInfo := transInfo.TransStrInfo()
	ad := strInfo[transmeta.HeaderTransRemoteAddr]
	if len(ad) > 0 {
		// when proxy case to get the actual remote address
		_ = remote.SetRemoteAddr(utils.NewNetAddr("tcp", ad))
	}
	return ctx, nil
}
