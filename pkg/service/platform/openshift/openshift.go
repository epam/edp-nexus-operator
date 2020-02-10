package openshift

import (
	"fmt"
	"github.com/epmd-edp/nexus-operator/v2/pkg/apis/edp/v1alpha1"
	nexusDefaultSpec "github.com/epmd-edp/nexus-operator/v2/pkg/service/nexus/spec"
	platformHelper "github.com/epmd-edp/nexus-operator/v2/pkg/service/platform/helper"
	"github.com/epmd-edp/nexus-operator/v2/pkg/service/platform/kubernetes"
	appsV1Api "github.com/openshift/api/apps/v1"
	routeV1Api "github.com/openshift/api/route/v1"
	securityV1Api "github.com/openshift/api/security/v1"
	appsV1client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	routeV1Client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	securityV1Client "github.com/openshift/client-go/security/clientset/versioned/typed/security/v1"
	"github.com/pkg/errors"
	coreV1Api "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("platform")

// OpenshiftService struct for Openshift platform service
type OpenshiftService struct {
	kubernetes.K8SService

	appClient      appsV1client.AppsV1Client
	routeClient    routeV1Client.RouteV1Client
	securityClient securityV1Client.SecurityV1Client
}

// Init initializes OpenshiftService
func (service *OpenshiftService) Init(config *rest.Config, scheme *runtime.Scheme, k8sClient *client.Client) error {
	err := service.K8SService.Init(config, scheme, k8sClient)
	if err != nil {
		return errors.Wrap(err, "failed to initialize Kubernetes service!")
	}

	appClient, err := appsV1client.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "failed to initialize Apps V1 Client")
	}
	service.appClient = *appClient

	routeClient, err := routeV1Client.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "failed to initialize Route V1 Client")
	}
	service.routeClient = *routeClient

	securityClient, err := securityV1Client.NewForConfig(config)
	if err != nil {
		return err
	}
	service.securityClient = *securityClient

	return nil
}

func (service OpenshiftService) CreateSecurityContext(instance v1alpha1.Nexus, p int32) error {
	l := platformHelper.GenerateLabels(instance.Name)
	uid := int64(200)

	sccObject := &securityV1Api.SecurityContextConstraints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			Labels:    l,
		},
		Volumes: []securityV1Api.FSType{
			securityV1Api.FSTypeSecret,
			securityV1Api.FSTypeDownwardAPI,
			securityV1Api.FSTypeEmptyDir,
			securityV1Api.FSTypePersistentVolumeClaim,
			securityV1Api.FSProjected,
			securityV1Api.FSTypeConfigMap,
		},
		AllowHostDirVolumePlugin: false,
		AllowHostIPC:             false,
		AllowHostNetwork:         false,
		AllowHostPID:             false,
		AllowHostPorts:           false,
		AllowPrivilegedContainer: false,
		AllowedCapabilities:      []coreV1Api.Capability{},
		AllowedFlexVolumes:       []securityV1Api.AllowedFlexVolume{},
		DefaultAddCapabilities:   []coreV1Api.Capability{},
		FSGroup: securityV1Api.FSGroupStrategyOptions{
			Type: securityV1Api.FSGroupStrategyMustRunAs,
		},
		Groups:                 []string{},
		Priority:               &p,
		ReadOnlyRootFilesystem: false,
		RunAsUser: securityV1Api.RunAsUserStrategyOptions{
			Type: securityV1Api.RunAsUserStrategyMustRunAs,
			UID:  &uid,
		},
		SELinuxContext: securityV1Api.SELinuxContextStrategyOptions{
			Type:           securityV1Api.SELinuxStrategyMustRunAs,
			SELinuxOptions: nil,
		},
		SupplementalGroups: securityV1Api.SupplementalGroupsStrategyOptions{
			Type:   securityV1Api.SupplementalGroupsStrategyRunAsAny,
			Ranges: nil,
		},
		Users: []string{
			fmt.Sprintf("system:serviceaccount:%s:%s", instance.Namespace, instance.Name),
		},
	}

	scc, err := service.securityClient.SecurityContextConstraints().Get(sccObject.Name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			scc, err = service.securityClient.SecurityContextConstraints().Create(sccObject)
			if err != nil {
				return err
			}
			log.Info(fmt.Sprintf("Security Context Constraint %s has been created", scc.Name))
			return nil
		}
		return err
	}
	if !reflect.DeepEqual(scc.Users, sccObject.Users) {
		scc, err = service.securityClient.SecurityContextConstraints().Update(sccObject)
		if err != nil {
			return err
		}
		log.Info(fmt.Sprintf("Security Context Constraint %s has been updated", scc.Name))
	}

	return nil
}

func (service OpenshiftService) AddKeycloakProxyToDeployConf(instance v1alpha1.Nexus, args []string) error {

	containerSpec := coreV1Api.Container{
		Name:            "keycloak-proxy",
		Image:           nexusDefaultSpec.NexusKeycloakProxyImage,
		ImagePullPolicy: coreV1Api.PullIfNotPresent,
		Ports: []coreV1Api.ContainerPort{
			{
				ContainerPort: nexusDefaultSpec.NexusKeycloakProxyPort,
				Protocol:      coreV1Api.ProtocolTCP,
			},
		},
		TerminationMessagePath:   "/dev/termination-log",
		TerminationMessagePolicy: coreV1Api.TerminationMessageReadFile,
		Args:                     args,
	}

	oldNexusDeploymentConfig, err := service.appClient.DeploymentConfigs(instance.Namespace).Get(instance.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if platformHelper.ContainerInDeployConf(oldNexusDeploymentConfig.Spec.Template.Spec.Containers, containerSpec) {
		log.V(1).Info("Keycloak proxy is present", "Namespace", instance.Namespace, "Name", instance.Name)
		return nil
	}
	oldNexusDeploymentConfig.Spec.Template.Spec.Containers = append(oldNexusDeploymentConfig.Spec.Template.Spec.Containers, containerSpec)

	_, err = service.appClient.DeploymentConfigs(instance.Namespace).Update(oldNexusDeploymentConfig)
	if err != nil {
		return err
	}

	log.Info("Keycloak proxy added.", "Namespace", instance.Namespace, "Name", instance.Name)
	return nil

}

// CreateDeployment performs creating DeploymentConfig in Openshift
func (service OpenshiftService) CreateDeployment(instance v1alpha1.Nexus) error {

	labels := platformHelper.GenerateLabels(instance.Name)
	deploymentConfigObject := &appsV1Api.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Spec: appsV1Api.DeploymentConfigSpec{
			Replicas: 1,
			Triggers: []appsV1Api.DeploymentTriggerPolicy{
				{
					Type: appsV1Api.DeploymentTriggerOnConfigChange,
				},
			},
			Strategy: appsV1Api.DeploymentStrategy{
				Type: appsV1Api.DeploymentStrategyTypeRecreate,
			},
			Selector: labels,
			Template: &coreV1Api.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: coreV1Api.PodSpec{
					Containers: []coreV1Api.Container{
						{
							Name:            instance.Name,
							Image:           instance.Spec.Image + ":" + instance.Spec.Version,
							ImagePullPolicy: coreV1Api.PullAlways,
							Env: []coreV1Api.EnvVar{
								{
									Name:  "CONTEXT_PATH",
									Value: "/",
								},
							},
							Ports: []coreV1Api.ContainerPort{
								{
									ContainerPort: nexusDefaultSpec.NexusPort,
								},
							},
							LivenessProbe: &coreV1Api.Probe{
								FailureThreshold:    5,
								InitialDelaySeconds: 180,
								PeriodSeconds:       20,
								SuccessThreshold:    1,
								Handler: coreV1Api.Handler{
									TCPSocket: &coreV1Api.TCPSocketAction{
										Port: intstr.FromInt(nexusDefaultSpec.NexusPort),
									},
								},
							},
							ReadinessProbe: &coreV1Api.Probe{
								FailureThreshold:    3,
								InitialDelaySeconds: 30,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								Handler: coreV1Api.Handler{
									TCPSocket: &coreV1Api.TCPSocketAction{
										Port: intstr.FromInt(nexusDefaultSpec.NexusPort),
									},
								},
							},
							TerminationMessagePath: "/dev/termination-log",
							Resources: coreV1Api.ResourceRequirements{
								Requests: map[coreV1Api.ResourceName]resource.Quantity{
									coreV1Api.ResourceMemory: resource.MustParse(nexusDefaultSpec.NexusMemoryRequest),
								},
							},
							VolumeMounts: []coreV1Api.VolumeMount{
								{
									MountPath: "/nexus-data",
									Name:      "data",
								},
								{
									MountPath: "/opt/sonatype/nexus/etc/nexus-default.properties",
									Name:      "config",
									SubPath:   "nexus-default.properties",
								},
							},
						},
					},
					ServiceAccountName: instance.Name,
					Volumes: []coreV1Api.Volume{
						{
							Name: "data",
							VolumeSource: coreV1Api.VolumeSource{
								PersistentVolumeClaim: &coreV1Api.PersistentVolumeClaimVolumeSource{
									ClaimName: fmt.Sprintf("%v-data", instance.Name),
								},
							},
						},
						{
							Name: "config",
							VolumeSource: coreV1Api.VolumeSource{
								ConfigMap: &coreV1Api.ConfigMapVolumeSource{
									LocalObjectReference: coreV1Api.LocalObjectReference{Name: fmt.Sprintf("%v-%v", instance.Name, nexusDefaultSpec.NexusDefaultPropertiesConfigMapPrefix)},
								},
							},
						},
					},
				},
			},
		},
	}
	if err := controllerutil.SetControllerReference(&instance, deploymentConfigObject, service.Scheme); err != nil {
		return err
	}

	deploymentConfig, err := service.appClient.DeploymentConfigs(deploymentConfigObject.Namespace).Get(deploymentConfigObject.Name, metav1.GetOptions{})
	if err == nil {
		return err
	}

	if !k8serrors.IsNotFound(err) {
		return err
	}

	deploymentConfig, err = service.appClient.DeploymentConfigs(deploymentConfigObject.Namespace).Create(deploymentConfigObject)
	if err != nil {
		return err
	}

	log.Info("DeploymentConfig has been created", "Namespace", instance.Namespace, "Name", instance.Name, "DeploymentName", deploymentConfig.Name)

	return nil
}

// CreateExternalEndpoint performs creating Route in Openshift
func (service OpenshiftService) CreateExternalEndpoint(instance v1alpha1.Nexus) error {

	labels := platformHelper.GenerateLabels(instance.Name)

	routeObject := &routeV1Api.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Spec: routeV1Api.RouteSpec{
			TLS: &routeV1Api.TLSConfig{
				Termination:                   routeV1Api.TLSTerminationEdge,
				InsecureEdgeTerminationPolicy: routeV1Api.InsecureEdgeTerminationPolicyRedirect,
			},
			To: routeV1Api.RouteTargetReference{
				Name: instance.Name,
				Kind: "Service",
			},
		},
	}

	if err := controllerutil.SetControllerReference(&instance, routeObject, service.Scheme); err != nil {
		return err
	}

	route, err := service.routeClient.Routes(routeObject.Namespace).Get(routeObject.Name, metav1.GetOptions{})
	if err == nil {
		return err
	}

	if !k8serrors.IsNotFound(err) {
		return err
	}

	route, err = service.routeClient.Routes(routeObject.Namespace).Create(routeObject)
	if err != nil {
		return err
	}

	log.Info("Route has been created", "Namespace", instance.Namespace, "Name", instance.Name, "RouteName", route.Name)

	return nil
}

// GetExternalUrl returns Web URL for object and scheme from Openshift Route
func (service OpenshiftService) GetExternalUrl(namespace string, name string) (webURL, scheme string, err error) {
	route, err := service.routeClient.Routes(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("Route not found", "Namespace", namespace, "Name", name, "RouteName", name)
			return "", "", nil
		}
		return "", "", err
	}

	routeScheme := "http"
	if route.Spec.TLS.Termination != "" {
		routeScheme = "https"
	}
	return fmt.Sprintf("%s://%s", routeScheme, route.Spec.Host), routeScheme, nil
}

// IsDeploymentReady verifies that DeploymentConfig is ready in Openshift
func (service OpenshiftService) IsDeploymentReady(instance v1alpha1.Nexus) (res *bool, err error) {
	deploymentConfig, err := service.appClient.DeploymentConfigs(instance.Namespace).Get(instance.Name, metav1.GetOptions{})
	if err != nil {
		return
	}

	t := deploymentConfig.Status.UpdatedReplicas == 1 && deploymentConfig.Status.AvailableReplicas == 1
	res = &t
	return
}

// GetRouteByCr return Route object with instance as a reference owner
func (service OpenshiftService) GetRouteByCr(instance v1alpha1.Nexus) (*routeV1Api.Route, error) {
	routeList, err := service.routeClient.Routes(instance.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "couldn't retrieve services list from the cluster")
	}
	for _, route := range routeList.Items {
		for _, owner := range route.OwnerReferences {
			if owner.UID == instance.UID {
				return &route, nil
			}
		}
	}
	return nil, nil
}

// UpdateExternalTargetPath performs updating route target port
func (service OpenshiftService) UpdateExternalTargetPath(instance v1alpha1.Nexus, targetPort intstr.IntOrString) error {
	instanceRoute, err := service.GetRouteByCr(instance)
	if err != nil || instanceRoute == nil {
		return errors.Wrap(err, "couldn't get route")
	}
	if instanceRoute.Spec.Port != nil && instanceRoute.Spec.Port.TargetPort == targetPort {
		log.V(1).Info("Target Port is already set", "Namespace", instance.Namespace, "Name", instance.Name, "TargetPort", targetPort.StrVal, "Route", instanceRoute.Name)
		return nil
	}
	instanceRoute.Spec.Port = &routeV1Api.RoutePort{TargetPort: targetPort}

	_, err = service.routeClient.Routes(instance.Namespace).Update(instanceRoute)
	return err
}
