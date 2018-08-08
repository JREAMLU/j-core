package util

import (
	"log"
	"os"
	"time"

	"github.com/JREAMLU/j-kit/go-micro/trace/opentracing"
	"github.com/juju/ratelimit"

	"github.com/hashicorp/consul/api"
	micro "github.com/micro/go-micro"
	microClient "github.com/micro/go-micro/client"
	"github.com/micro/go-micro/registry"
	microServer "github.com/micro/go-micro/server"
	client "github.com/micro/go-plugins/client/grpc"
	register "github.com/micro/go-plugins/registry/consul"
	server "github.com/micro/go-plugins/server/grpc"
	transport "github.com/micro/go-plugins/transport/grpc"
	microGobreaker "github.com/micro/go-plugins/wrapper/breaker/gobreaker"
	microRatelimit "github.com/micro/go-plugins/wrapper/ratelimiter/ratelimit"
	"github.com/sony/gobreaker"
	// brokerKafka "github.com/micro/go-plugins/broker/kafka"
)

// NewMicroService new micro service
func NewMicroService(config *Config) micro.Service {
	t, err := NewTrace(
		config.Service.Name,
		config.Service.Version,
		config.Kafka.ZipkinBroker,
		config.Kafka.ZipkinTopic,
	)
	if err != nil {
		panic(err)
	}

	clientBucket := ratelimit.NewBucketWithRate(config.RateLimit.ClientRate, config.RateLimit.ClientCapacity)
	serverBucket := ratelimit.NewBucketWithRate(config.RateLimit.ServerRate, config.RateLimit.ServerCapacity)

	service := micro.NewService(
		micro.Client(client.NewClient(
			microClient.Wrap(microGobreaker.NewClientWrapper(
				gobreaker.NewCircuitBreaker(gobreaker.Settings{
					Name:        config.Service.Name,
					MaxRequests: config.CircuitBreaker.MaxRequests,
					Interval:    time.Duration(config.CircuitBreaker.Interval) * time.Second,
					Timeout:     time.Duration(config.CircuitBreaker.Timeout) * time.Second,
					ReadyToTrip: func(counts gobreaker.Counts) bool {
						failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
						return counts.Requests >= config.CircuitBreaker.CountsRequests && failureRatio >= config.CircuitBreaker.FailureRatio
					},
					OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
					},
				}),
			)),
			microClient.Wrap(microRatelimit.NewClientWrapper(clientBucket, config.RateLimit.ClientWait)),
		)),
		micro.Server(server.NewServer(
			microServer.WrapHandler(microRatelimit.NewHandlerWrapper(serverBucket, config.RateLimit.ServerWait)),
		)),
		micro.Registry(register.NewRegistry(
			registry.Option(func(opts *registry.Options) {
				if len(config.Consul.RegistryAddrs) > 0 {
					opts.Addrs = config.Consul.RegistryAddrs
					log.Printf("Registry Consul Addrs: %v\n", config.Consul.RegistryAddrs)
				} else {
					log.Printf("Registry Consul Addr: %v\n", os.Getenv(api.HTTPAddrEnvName))
				}
			}),
		)),
		micro.Transport(transport.NewTransport()),
		micro.Name(config.Service.Name),
		micro.Version(config.Service.Version),
		micro.WrapClient(opentracing.NewClientWrapper(t)),
		micro.WrapHandler(opentracing.NewHandlerWrapper(t)),
		// micro.Broker(brokerKafka.NewBroker(
		// 	broker.Option(func(opt *broker.Options) {
		// 		opt.Addrs = []string{"10.200.119.128:9092"}
		// 	}),
		// )),
	)

	service.Init(
		micro.RegisterTTL(time.Duration(config.Service.RegisterTTL)*time.Second),
		micro.RegisterInterval(time.Duration(config.Service.RegisterInterval)*time.Second),
	)

	return service
}