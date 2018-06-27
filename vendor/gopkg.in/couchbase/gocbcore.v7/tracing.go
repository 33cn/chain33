package gocbcore

import (
	"fmt"
	"github.com/opentracing/opentracing-go"
)

type opTracer struct {
	parentContext opentracing.SpanContext
	opSpan        opentracing.Span
}

func (tracer *opTracer) Finish() {
	if tracer.opSpan != nil {
		tracer.opSpan.Finish()
	}
}

func (tracer *opTracer) RootContext() opentracing.SpanContext {
	if tracer.opSpan != nil {
		return tracer.opSpan.Context()
	}

	return tracer.parentContext
}

func (agent *Agent) createOpTrace(operationName string, parentContext opentracing.SpanContext) *opTracer {
	if agent.noRootTraceSpans {
		return &opTracer{
			parentContext: parentContext,
			opSpan:        nil,
		}
	}

	opSpan := agent.tracer.StartSpan(operationName,
		opentracing.ChildOf(parentContext),
		opentracing.Tag{Key: "component", Value: "couchbase-go-sdk"},
		opentracing.Tag{Key: "db.instance", Value: agent.bucket},
		opentracing.Tag{Key: "db.type", Value: "couchbase"},
		opentracing.Tag{Key: "span.kind", Value: "client"})

	return &opTracer{
		parentContext: parentContext,
		opSpan:        opSpan,
	}
}

func (agent *Agent) startCmdTrace(req *memdQRequest) {
	if req.cmdTraceSpan != nil {
		logWarnf("Attempted to start tracing on traced request")
		return
	}

	if req.RootTraceContext == nil {
		return
	}

	req.cmdTraceSpan = agent.tracer.StartSpan(
		getCommandName(req.memdPacket.Opcode),
		opentracing.ChildOf(req.RootTraceContext),
		opentracing.Tag{Key: "retry", Value: req.retryCount})
}

func (agent *Agent) stopCmdTrace(req *memdQRequest) {
	if req.RootTraceContext == nil {
		return
	}

	if req.cmdTraceSpan == nil {
		logWarnf("Attempted to stop tracing on untraced request")
		return
	}

	req.cmdTraceSpan.Finish()
	req.cmdTraceSpan = nil
}

func (agent *Agent) startNetTrace(req *memdQRequest) {
	if req.cmdTraceSpan == nil {
		return
	}

	if req.netTraceSpan != nil {
		logWarnf("Attempted to start net tracing on traced request")
		return
	}

	req.netTraceSpan = agent.tracer.StartSpan(
		"rpc",
		opentracing.ChildOf(req.cmdTraceSpan.Context()),
		opentracing.Tag{Key: "span.kind", Value: "client"})
}

func (agent *Agent) stopNetTrace(req *memdQRequest, resp *memdQResponse, client *memdClient) {
	if req.cmdTraceSpan == nil {
		return
	}

	if req.netTraceSpan == nil {
		logWarnf("Attempted to stop net tracing on an untraced request")
		return
	}

	req.netTraceSpan.SetTag("couchbase.operation_id", resp.sourceConnId+"/"+fmt.Sprintf("%08x", resp.Opaque))
	req.netTraceSpan.SetTag("local.address", client.conn.LocalAddr())
	req.netTraceSpan.SetTag("peer.address", client.conn.RemoteAddr())
	if resp.FrameExtras != nil && resp.FrameExtras.HasSrvDuration {
		req.netTraceSpan.SetTag("server_duration", resp.FrameExtras.SrvDuration)
	}

	req.netTraceSpan.Finish()
	req.netTraceSpan = nil
}

func (agent *Agent) cancelReqTrace(req *memdQRequest, err error) {
	if req.cmdTraceSpan != nil {
		if req.netTraceSpan != nil {
			req.netTraceSpan.Finish()
		}

		req.cmdTraceSpan.Finish()
	}
}
