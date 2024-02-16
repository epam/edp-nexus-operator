package main

import (
	"flag"
	"os"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	//+kubebuilder:scaffold:imports
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	buildInfo "github.com/epam/edp-common/pkg/config"
	nexusApiV1Alpha1 "github.com/epam/edp-nexus-operator/api/v1alpha1"
	"github.com/epam/edp-nexus-operator/controllers/blobstore"
	"github.com/epam/edp-nexus-operator/controllers/cleanuppolicy"
	"github.com/epam/edp-nexus-operator/controllers/nexus"
	"github.com/epam/edp-nexus-operator/controllers/repository"
	"github.com/epam/edp-nexus-operator/controllers/role"
	"github.com/epam/edp-nexus-operator/controllers/script"
	"github.com/epam/edp-nexus-operator/controllers/user"
	nexusclient "github.com/epam/edp-nexus-operator/pkg/client/nexus"
	"github.com/epam/edp-nexus-operator/pkg/helper"
	"github.com/epam/edp-nexus-operator/pkg/webhook"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

const nexusOperatorLock = "edp-nexus-operator-lock"

func main() {
	var (
		metricsAddr          string
		enableLeaderElection bool
		probeAddr            string
	)

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", helper.RunningInCluster(),
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	mode, err := helper.GetDebugMode()
	if err != nil {
		setupLog.Error(err, "unable to get debug mode value")
		os.Exit(1)
	}

	opts := zap.Options{
		Development: mode,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(nexusApiV1Alpha1.AddToScheme(scheme))

	v := buildInfo.Get()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	setupLog.Info("Starting the Nexus Operator",
		"version", v.Version,
		"git-commit", v.GitCommit,
		"git-tag", v.GitTag,
		"build-date", v.BuildDate,
		"go-version", v.Go,
		"go-client", v.KubectlVersion,
		"platform", v.Platform,
	)

	ns := helper.GetWatchNamespace()
	cfg := ctrl.GetConfigOrDie()

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		HealthProbeBindAddress: probeAddr,
		Port:                   9443,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       nexusOperatorLock,
		MapperProvider: func(c *rest.Config) (meta.RESTMapper, error) {
			return apiutil.NewDynamicRESTMapper(cfg)
		},
		Namespace: ns,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	apiClientProvider := nexusclient.NewApiClientProvider(mgr.GetClient())

	if err = nexus.NewNexusReconciler(mgr.GetClient(), mgr.GetScheme(), apiClientProvider).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "nexus")
		os.Exit(1)
	}

	if err = role.NewNexusRoleReconciler(mgr.GetClient(), mgr.GetScheme(), apiClientProvider).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "role")
		os.Exit(1)
	}

	if err = user.NewNexusUserReconciler(mgr.GetClient(), mgr.GetScheme(), apiClientProvider).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "user")
		os.Exit(1)
	}

	if err = repository.NewNexusRepositoryReconciler(mgr.GetClient(), apiClientProvider).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "repository")
		os.Exit(1)
	}

	if err = script.NewNexusScriptReconciler(mgr.GetClient(), apiClientProvider).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NexusScript")
		os.Exit(1)
	}

	if err = blobstore.NewNexusBlobStoreReconciler(mgr.GetClient(), apiClientProvider).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NexusBlobStore")
		os.Exit(1)
	}

	if err = cleanuppolicy.NewNexusCleanupPolicyReconciler(mgr.GetClient(), apiClientProvider).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NexusCleanupPolicy")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	ctx := ctrl.SetupSignalHandler()

	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = webhook.RegisterValidationWebHook(ctx, mgr, ns); err != nil {
			setupLog.Error(err, "failed to create webhook", "webhook", "NexusRepositoryß")
			os.Exit(1)
		}
	}

	if err = mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}

	if err = mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")

	if err = mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
