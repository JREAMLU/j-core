package util

import (
	"github.com/JREAMLU/j-kit/ext"
	jopentracing "github.com/JREAMLU/j-kit/go-micro/trace/opentracing"

	micro "github.com/micro/go-micro"
	opentracing "github.com/opentracing/opentracing-go"
	zipkin "github.com/openzipkin/zipkin-go-opentracing"
)

var (
	_debug         = true
	_sameSpan      = true
	_traceID128Bit = true
)

// NewTrace new trace
func NewTrace(serviceName, version string, kafkaAddrs []string, kafkaTopic string, hostPort ...string) (opentracing.Tracer, error) {
	collector, err := zipkin.NewKafkaCollector(
		kafkaAddrs,
		zipkin.KafkaTopic(kafkaTopic),
	)
	if err != nil {
		return nil, err
	}

	var ipPort string
	if len(hostPort) == 0 {
		ipPort, err = ext.ExtractIP("")
	} else {
		ipPort, err = ext.ExtractIP(hostPort[0])
	}

	if err != nil {
		return nil, err
	}

	recorder := zipkin.NewRecorder(collector, _debug, ipPort, serviceName)
	tracer, err := zipkin.NewTracer(
		recorder,
		zipkin.ClientServerSameSpan(_sameSpan),
		zipkin.TraceID128Bit(_traceID128Bit),
	)
	if err != nil {
		return nil, err
	}

	return tracer, nil
}

// SetZipkin set zipkin trace
func SetZipkin(service micro.Service, kafkaAddrs []string, kafkaTopic string, hostPort ...string) {
	opts := service.Options()

	setZipkin(
		opts.Server.Options().Name,
		opts.Server.Options().Version,
		kafkaAddrs,
		kafkaTopic,
		hostPort...,
	)(&opts)
}

func setZipkin(serviceName, version string, kafkaAddrs []string, kafkaTopic string, hostPort ...string) micro.Option {
	return func(opt *micro.Options) {
		collector, err := zipkin.NewKafkaCollector(
			kafkaAddrs,
			zipkin.KafkaTopic(kafkaTopic),
		)
		if err != nil {
			panic(err)
		}

		var ipPort string
		if len(hostPort) == 0 {
			ipPort, err = ext.ExtractIP("")
		} else {
			ipPort, err = ext.ExtractIP(hostPort[0])
		}

		if err != nil {
			panic(err)
		}

		recorder := zipkin.NewRecorder(collector, _debug, ipPort, serviceName)
		tracer, err := zipkin.NewTracer(
			recorder,
			zipkin.ClientServerSameSpan(_sameSpan),
			zipkin.TraceID128Bit(_traceID128Bit),
		)
		if err != nil {
			panic(err)
		}

		clientWrap := jopentracing.NewClientWrapper(tracer)
		serverWrap := jopentracing.NewHandlerWrapper(tracer)

		micro.WrapClient(clientWrap)(opt)
		micro.WrapHandler(serverWrap)(opt)
	}
}
