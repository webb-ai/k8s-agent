package main

import (
	"path"
	"time"

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
	// TODO: make this configurable
	qps            float32 = 20.0
	burst                  = 30
	resyncPeriod           = time.Second * 10
	metricsAddress         = ":9090"
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

func main() {
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
		LivenessEndpointName: "/healthz",
		MetricsBindAddress:   metricsAddress,
	})
	if err != nil {
		klog.Fatal(err)
	}

	klog.Infof("creating resource collector")
	dynamicClient := dynamic.NewForConfigOrDie(config)
	logger := newRotateFileLogger("/var/log/webb-ai", "k8s_resource.log", 100, 28, 3)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	collector := k8s.NewCollector(resyncPeriod, dynamicClient, logger)

	klog.Infof("adding resource collector to controller manager")
	if err := controllerManager.Add(collector); err != nil {
		klog.Fatal(err)
	}

	ctx := apiserver.SetupSignalContext()
	if err := controllerManager.Start(ctx); err != nil {
		klog.Fatal(err)
	}
}
