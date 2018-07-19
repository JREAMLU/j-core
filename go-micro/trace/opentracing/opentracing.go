// Package opentracing provides wrappers for OpenTracing
package opentracing

import (
	"context"
	"fmt"
	"reflect"

	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/metadata"
	"github.com/micro/go-micro/server"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
)

type otWrapper struct {
	ot opentracing.Tracer
	client.Client
}

func traceIntoContext(ctx context.Context, tracer opentracing.Tracer, name string) (context.Context, opentracing.Span, error) {
	md, ok := metadata.FromContext(ctx)
	if !ok {
		md = make(map[string]string)
	}

	var span opentracing.Span
	wireContext, err := tracer.Extract(opentracing.TextMap, opentracing.TextMapCarrier(md))
	if err != nil {
		span = tracer.StartSpan(name)
	} else {
		span = tracer.StartSpan(name, opentracing.ChildOf(wireContext))
	}

	if err := span.Tracer().Inject(span.Context(), opentracing.TextMap, opentracing.TextMapCarrier(md)); err != nil {
		return nil, nil, err
	}

	ctx = opentracing.ContextWithSpan(ctx, span)
	ctx = metadata.NewContext(ctx, md)
	return ctx, span, nil
}

func (o *otWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	name := fmt.Sprintf("%s.%s", req.Service(), req.Method())
	ctx, span, err := traceIntoContext(ctx, o.ot, name)
	if err != nil {
		return err
	}
	defer span.Finish()

	r := req.Request()
	t := reflect.TypeOf(r)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	fieldNum := t.NumField()
	result := make([]string, 0, fieldNum)
	for i := 0; i < fieldNum; i++ {
		result = append(result, t.Field(i).Name)
	}
	fmt.Println("++++++++++++: ", result)
	fmt.Println("++++++++++++: ", req.Request())

	span.LogFields(
		log.String("callSrv", req.Service()),
		log.String("method", req.Method()),
		log.String("content-type", req.ContentType()),
		// log.String("params", req.Request().(string)),
	)

	err = o.Client.Call(ctx, req, rsp, opts...)
	if err != nil {
		span.LogEvent(fmt.Sprintf("CALL ERROR: %v", err))
		return err
	}

	return nil
}

func (o *otWrapper) Publish(ctx context.Context, p client.Message, opts ...client.PublishOption) error {
	name := fmt.Sprintf("Pub to %s", p.Topic())
	ctx, span, err := traceIntoContext(ctx, o.ot, name)
	if err != nil {
		return err
	}
	defer span.Finish()

	return o.Client.Publish(ctx, p, opts...)
}

// NewClientWrapper accepts an open tracing Trace and returns a Client Wrapper
func NewClientWrapper(ot opentracing.Tracer) client.Wrapper {
	return func(c client.Client) client.Client {
		return &otWrapper{ot, c}
	}
}

// NewCallWrapper accepts an opentracing Tracer and returns a Call Wrapper
func NewCallWrapper(ot opentracing.Tracer) client.CallWrapper {
	return func(cf client.CallFunc) client.CallFunc {
		return func(ctx context.Context, addr string, req client.Request, rsp interface{}, opts client.CallOptions) error {
			name := fmt.Sprintf("%s.%s", req.Service(), req.Method())
			ctx, span, err := traceIntoContext(ctx, ot, name)
			if err != nil {
				return err
			}
			defer span.Finish()

			return cf(ctx, addr, req, rsp, opts)
		}
	}
}

// NewHandlerWrapper accepts an opentracing Tracer and returns a Handler Wrapper
func NewHandlerWrapper(ot opentracing.Tracer) server.HandlerWrapper {
	return func(h server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			name := fmt.Sprintf("%s.%s", req.Service(), req.Method())
			ctx, span, err := traceIntoContext(ctx, ot, name)
			if err != nil {
				return err
			}
			defer span.Finish()

			return h(ctx, req, rsp)
		}
	}
}

// NewSubscriberWrapper accepts an opentracing Tracer and returns a Subscriber Wrapper
func NewSubscriberWrapper(ot opentracing.Tracer) server.SubscriberWrapper {
	return func(next server.SubscriberFunc) server.SubscriberFunc {
		return func(ctx context.Context, msg server.Message) error {
			name := "Pub to " + msg.Topic()
			ctx, span, err := traceIntoContext(ctx, ot, name)
			if err != nil {
				return err
			}
			defer span.Finish()

			return next(ctx, msg)
		}
	}
}
