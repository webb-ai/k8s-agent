package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"time"

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
	qps                float32 = 20.0
	burst                      = 30
	resyncPeriod               = time.Second * 10
	collectionInterval         = time.Minute * 5
	metricsAddress             = ":9090"
	dataDir                    = "/app/data/"
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
		return &api.NoOpClient{}
	}
	return client
}

func main() {
	var version bool
	flag.BoolVar(&version, "version", false, "show version")
	flag.StringVar(&dataDir, "data-dir", dataDir, "directory to store staged data")
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
	config.QPS = qps
	config.Burst = burst

	klog.Infof("creating controller manager")
	controllerManager, err := controllerruntime.NewManager(config, controllerruntime.Options{
		LivenessEndpointName:          "/healthz",
		MetricsBindAddress:            metricsAddress,
		LeaderElection:                true,
		LeaderElectionID:              "webb-ai.k8s-resource-collector",
		LeaderElectionNamespace:       os.Getenv("POD_NAMESPACE"),
		LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		klog.Fatal(err)
	}

	klog.Infof("creating resource collector")
	dynamicClient := dynamic.NewForConfigOrDie(config)
	logger := newRotateFileLogger(dataDir, "k8s_resource.log", 100, 28, 10)
	collector := k8s.NewCollector(resyncPeriod, collectionInterval, dynamicClient, logger, NewClient())

	klog.Infof("adding resource collector to controller manager")
	if err := controllerManager.Add(collector); err != nil {
		klog.Fatal(err)
	}

	ctx := apiserver.SetupSignalContext()
	if err := controllerManager.Start(ctx); err != nil {
		klog.Fatal(err)
	}
}
