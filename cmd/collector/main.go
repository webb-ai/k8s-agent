package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/webb-ai/k8s-agent/pkg/server"

	"github.com/webb-ai/k8s-agent/pkg/agentinfo"

	"k8s.io/client-go/dynamic/dynamicinformer"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/webb-ai/k8s-agent/pkg/kafka"

	"k8s.io/client-go/discovery"

	"sigs.k8s.io/controller-runtime/pkg/healthz"

	"github.com/rs/zerolog"
	"github.com/webb-ai/k8s-agent/pkg/api"
	"github.com/webb-ai/k8s-agent/pkg/http"
	"gopkg.in/natefinch/lumberjack.v2"
	"k8s.io/client-go/dynamic"
	klog "k8s.io/klog/v2"

	apiserver "k8s.io/apiserver/pkg/server"

	"github.com/webb-ai/k8s-agent/pkg/k8s"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	BuildVersion = "N/A"
)

var (
	// TODO: make this configurable
	qps                      = 20.0
	burst                    = 30
	resyncPeriod             = time.Second * 10
	eventCollectionInterval  = time.Minute * 5
	backupCollectionInterval = time.Minute * 60
	agentInfoPeriod          = time.Minute * 1
	dataDir                  = "/app/data/"
	metricsAddress           = ":9090"
	healthProbeAddress       = ":9091"
	apiServerProxyAddress    = ":9092"
)

var (
	kafkaBootstrapServers = ""
	kafkaPollingInterval  = time.Minute * 5
)

func newRotateFileLogger(dir, fileName string, maxSizeMb, maxAge, maxBackups int) zerolog.Logger {
	writer := &lumberjack.Logger{
		Filename:   path.Join(dir, fileName),
		MaxSize:    maxSizeMb,
		MaxAge:     maxAge,
		MaxBackups: maxBackups,
		Compress:   true,
	}
	return zerolog.New(writer).With().Timestamp().Logger()
}

func NewClient(agentVersion, kafkaServers string) api.Client {
	client := http.NewWebbaiClient(agentVersion, kafkaServers)
	if client == nil {
		klog.Warningf("cannot initialize webb.ai http client. Will not stream data to webb.ai")
		return &api.NoOpClient{}
	}
	return client
}

func newKafkaCollector(client api.Client) *kafka.Collector {
	if kafkaBootstrapServers == "" {
		klog.Infof("kafka bootstrap server not configured, skipping kafka collector loop")
		return nil
	}
	bootstrapServers := strings.Split(kafkaBootstrapServers, ",")
	collector, err := kafka.NewKafkaCollector(
		bootstrapServers,
		kafkaPollingInterval,
		client,
	)
	if err != nil {
		klog.Errorf("error creating kafka collection: %w", err)
		return nil
	}
	return collector
}

func main() {
	var version bool
	flag.BoolVar(&version, "version", false, "show version")
	flag.StringVar(&dataDir, "data-dir", dataDir, "directory to store staged data")
	flag.Float64Var(&qps, "kube-api-qps", qps, "max qps from this client to kube api server, default 20")
	flag.IntVar(&burst, "kube-api-burst", burst, "max burst for throttle from this client to kube api server, default 30")
	flag.DurationVar(&eventCollectionInterval, "event-collect-interval", eventCollectionInterval, "interval to collect events")
	flag.BoolVar(&api.RedactEnvVar, "redact-env-var", false, "redact env var")

	flag.StringVar(&kafkaBootstrapServers, "kafka-bootstrap-servers", kafkaBootstrapServers, "bootstrap servers for kafka")
	flag.DurationVar(&kafkaPollingInterval, "kafka-polling-interval", kafkaPollingInterval, "polling interval to detect kafka changes")

	flag.Parse()

	if version {
		fmt.Printf("webb.ai k8s agent version %s\n", BuildVersion)
		os.Exit(0)
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// Config precedence:
	//
	// * --kubeconfig flag pointing at a file
	//
	// * KUBECONFIG environment variable pointing at a file
	//
	// * In-cluster config if running in cluster
	//
	// * $HOME/.kube/config if exists.

	config, err := config.GetConfig()

	if err != nil {
		klog.Fatal(err)
	}
	config.QPS = float32(qps)
	config.Burst = burst

	klog.Infof("creating controller manager")
	controllerManager, err := controllerruntime.NewManager(config, controllerruntime.Options{
		HealthProbeBindAddress:        healthProbeAddress,
		Metrics:                       metricsserver.Options{BindAddress: metricsAddress},
		LeaderElection:                true,
		LeaderElectionID:              "webb-ai.k8s-resource-collector",
		LeaderElectionNamespace:       os.Getenv("POD_NAMESPACE"),
		LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		klog.Fatal(err)
	}

	if err := controllerManager.AddHealthzCheck("ping", healthz.Ping); err != nil {
		klog.Fatalf("Failed to add health check endpoint: %w", err)
	}

	klog.Infof("creating resource collector")
	dynamicClient := dynamic.NewForConfigOrDie(config)
	discoveryClient := discovery.NewDiscoveryClientForConfigOrDie(config)
	informerFactory := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, resyncPeriod)
	apiClient := NewClient(BuildVersion, kafkaBootstrapServers)
	collector := k8s.NewChangeCollector(
		eventCollectionInterval,
		backupCollectionInterval,
		informerFactory,
		discoveryClient,
		newRotateFileLogger(dataDir, "k8s_resource.log", 100, 28, 10),
		apiClient,
	)

	klog.Infof("adding resource collector to controller manager")
	if err := controllerManager.Add(collector); err != nil {
		klog.Fatal(err)
	}

	klog.Infof("creating kafka collector")
	if kafkaCollector := newKafkaCollector(apiClient); kafkaCollector != nil {
		if err := controllerManager.Add(kafkaCollector); err != nil {
			klog.Fatal(err)
		}
	}

	klog.Infof("creating agent health controller")
	if agentInfoController := agentinfo.NewController(agentInfoPeriod, apiClient); agentInfoController != nil {
		if err := controllerManager.Add(agentInfoController); err != nil {
			klog.Fatal(err)
		}
	}

	klog.Infof("creating api server proxy")
	apiServerProxy := server.NewApiServerProxy(apiClient, apiServerProxyAddress)
	if err := controllerManager.Add(apiServerProxy); err != nil {
		klog.Fatal(err)
	}

	ctx := apiserver.SetupSignalContext()
	if err := controllerManager.Start(ctx); err != nil {
		klog.Fatal(err)
	}
}
