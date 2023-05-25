package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"time"

	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	"github.com/webb-ai/k8s-agent/pkg/api"

	"github.com/webb-ai/k8s-agent/pkg/http"

	"github.com/rs/zerolog"
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
	qps                              = 20.0
	burst                            = 30
	resyncPeriod                     = time.Second * 10
	eventCollectionInterval          = time.Minute * 5
	trafficMetricsCollectionInterval = time.Minute * 1
	trafficCollectorPodSelector      = "app=traffic-collector"
	trafficCollectorMetricsPort      = 9095
	trafficCollectorServerPort       = 8897
	metricsAddress                   = ":9090"
	healthProbeAddress               = ":9091"
	dataDir                          = "/app/data/"
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

func NewClient() api.Client {
	client := http.NewWebbaiClient()
	if client == nil {
		klog.Warningf("cannot initialize webb.ai http client. Will not stream data to webb.ai")
		return &api.NoOpClient{}
	}
	return client
}

func main() {
	var version bool
	flag.BoolVar(&version, "version", false, "show version")
	flag.StringVar(&dataDir, "data-dir", dataDir, "directory to store staged data")
	flag.Float64Var(&qps, "kube-api-qps", qps, "max qps from this client to kube api server, default 20")
	flag.IntVar(&burst, "kube-api-burst", burst, "max burst for throttle from this client to kube api server, default 30")
	flag.DurationVar(&eventCollectionInterval, "event-collect-interval", eventCollectionInterval, "interval to collect events")

	flag.DurationVar(&trafficMetricsCollectionInterval, "traffic-metric-collection-interval", trafficMetricsCollectionInterval, "interval to collect traffic metrics")
	flag.StringVar(&trafficCollectorPodSelector, "traffic-collector-pod-selector", trafficCollectorPodSelector, "pod selector for webbai traffic collector")
	flag.IntVar(&trafficCollectorMetricsPort, "traffic-collector-metrics-port", trafficCollectorMetricsPort, "port number to get metrics from traffic collector")
	flag.IntVar(&trafficCollectorServerPort, "traffic-collector-server-port", trafficCollectorServerPort, "port number of traffic collector server")

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
		MetricsBindAddress:            metricsAddress,
		LeaderElection:                true,
		LeaderElectionID:              "webb-ai.k8s-resource-collector",
		LeaderElectionNamespace:       os.Getenv("POD_NAMESPACE"),
		LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		klog.Fatal(err)
	}

	if err := controllerManager.AddHealthzCheck("ping", healthz.Ping); err != nil {
		klog.Fatalf("Failed to add health check endpoint: %v", err)
	}

	klog.Infof("creating resource collector")
	dynamicClient := dynamic.NewForConfigOrDie(config)
	trafficPodSelector, err := labels.Parse(trafficCollectorPodSelector)
	if err != nil {
		klog.Fatal(err)
	}

	resourceLogger := newRotateFileLogger(dataDir, "k8s_resource.log", 100, 28, 10)
	trafficLogger := newRotateFileLogger(dataDir, "k8s_traffic.log", 100, 28, 10)
	collector := k8s.NewCollector(
		resyncPeriod,
		eventCollectionInterval,
		trafficMetricsCollectionInterval,
		trafficPodSelector,
		trafficCollectorServerPort,
		trafficCollectorMetricsPort,
		dynamicClient,
		resourceLogger,
		trafficLogger,
		NewClient(),
	)

	klog.Infof("adding resource collector to controller manager")
	if err := controllerManager.Add(collector); err != nil {
		klog.Fatal(err)
	}

	ctx := apiserver.SetupSignalContext()
	if err := controllerManager.Start(ctx); err != nil {
		klog.Fatal(err)
	}
}
