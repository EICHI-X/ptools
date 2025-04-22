package wtrace

import (
	"context"
	"encoding/binary"
	"math/rand"
	"sync"

	crand "crypto/rand"

	"go.opentelemetry.io/otel/trace"
)

// IDGenerator allows custom generators for TraceID and SpanID.
type IDGenerator interface {
	// DO NOT CHANGE: any modification will not be backwards compatible and
	// must never be done outside of a new major release.

	// NewIDs returns a new trace and span ID.
	NewIDs(ctx context.Context) (trace.TraceID, trace.SpanID)
	// DO NOT CHANGE: any modification will not be backwards compatible and
	// must never be done outside of a new major release.

	// NewSpanID returns a ID for a new span in the trace with traceID.
	NewSpanID(ctx context.Context, traceID trace.TraceID) trace.SpanID
	// DO NOT CHANGE: any modification will not be backwards compatible and
	// must never be done outside of a new major release.
}

type randomIDGenerator struct {
	sync.Mutex
	randSource *rand.Rand
}

var _ IDGenerator = &randomIDGenerator{}

// NewSpanID returns a non-zero span ID from a randomly-chosen sequence.
func (gen *randomIDGenerator) NewSpanID(ctx context.Context, traceID trace.TraceID) trace.SpanID {
	gen.Lock()
	defer gen.Unlock()
	sid := trace.SpanID{}
	_, _ = gen.randSource.Read(sid[:])
	return sid
}

// NewIDs returns a non-zero trace ID and a non-zero span ID from a
// randomly-chosen sequence.
func (gen *randomIDGenerator) NewIDs(ctx context.Context) (trace.TraceID, trace.SpanID) {
	gen.Lock()
	defer gen.Unlock()
	tid := trace.TraceID{}
	_, _ = gen.randSource.Read(tid[:])
	sid := trace.SpanID{}
	_, _ = gen.randSource.Read(sid[:])
	return tid, sid
}

func DefaultIDGenerator() IDGenerator {
	gen := &randomIDGenerator{}
	var rngSeed int64
	_ = binary.Read(crand.Reader, binary.LittleEndian, &rngSeed)
	gen.randSource = rand.New(rand.NewSource(rngSeed))

	return gen
}
func InitSpanToContext(ctx context.Context, id []byte) (context.Context, trace.SpanContext) {
	var t trace.TraceID
	var s trace.SpanID
	if len(id) > 0 {
		
		// 将 byte 装换为 16进制的字符串
		idStr := []byte(id)
		if len(idStr) >= 16 {
			t = trace.TraceID(idStr[:16])
		}
	}
	tg, sg := DefaultIDGenerator().NewIDs(ctx)
	if !t.IsValid(){
		t = tg
	}
	s = sg
	spin := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: t,
		SpanID:  s,
	})
	ctx = trace.ContextWithSpanContext(ctx, spin)
	return ctx, spin

}
