package main

import (
	"context"
	"errors"
	"flag"
	"os"
	"syscall"

	"sigs.k8s.io/sig-storage-lib-external-provisioner/v8/controller"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	klog "k8s.io/klog/v2"
)

const (
	PROVISIONER_NAME          = "linkvt/flashblade-provisioner"
	PROVISIONER_ANNOTATION    = "provisioner"
	SKIP_TLS_VERIFICATION_ENV = "SKIP_TLS_VERIFICATION"
	STORAGE_API_ADDRESS_ENV   = "STORAGE_API_ADDRESS"
	STORAGE_API_TOKEN_ENV     = "STORAGE_API_TOKEN"
	STORAGE_NFS_HOST_ENV      = "STORAGE_NFS_HOST"
	VOLUME_NAME_PREFIX        = "k8s-"
)

type FlashBladeProvisioner struct {
	nfsHost       string
	flashBladeApi *FlashBladeApi
}

func NewFlashBladeProvisioner() controller.Provisioner {
	storageApiAddress := getEnvOrFail(STORAGE_API_ADDRESS_ENV)
	storageApiToken := getEnvOrFail(STORAGE_API_TOKEN_ENV)
	skipTlsVerification := os.Getenv(SKIP_TLS_VERIFICATION_ENV) == "true"

	return &FlashBladeProvisioner{
		nfsHost:       getEnvOrFail(STORAGE_NFS_HOST_ENV),
		flashBladeApi: NewFlashBladeApi(storageApiAddress, storageApiToken, skipTlsVerification),
	}
}

func getEnvOrFail(envName string) string {
	envValue := os.Getenv(envName)
	if envValue == "" {
		klog.Fatalf("env variable %s must be set", envName)
	}
	return envValue
}

func (p *FlashBladeProvisioner) provisionedVolumeName(pvName string) string {
	return VOLUME_NAME_PREFIX + pvName
}

func (p *FlashBladeProvisioner) Provision(ctx context.Context, options controller.ProvisionOptions) (*v1.PersistentVolume, controller.ProvisioningState, error) {
	requestedSize := options.PVC.Spec.Resources.Requests[v1.ResourceName(v1.ResourceStorage)]
	volume, err := p.flashBladeApi.CreateVolume(p.provisionedVolumeName((options.PVName)), requestedSize.Value())
	if err != nil {
		return nil, controller.ProvisioningNoChange, err
	}

	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: options.PVName,
			Annotations: map[string]string{
				PROVISIONER_ANNOTATION: PROVISIONER_NAME,
			},
		},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeReclaimPolicy: *options.StorageClass.ReclaimPolicy,
			AccessModes:                   options.PVC.Spec.AccessModes,
			Capacity: v1.ResourceList{
				v1.ResourceName(v1.ResourceStorage): requestedSize,
			},
			PersistentVolumeSource: v1.PersistentVolumeSource{
				NFS: &v1.NFSVolumeSource{
					Path:   "/" + volume.Name,
					Server: p.nfsHost,
				},
			},
		},
	}

	return pv, controller.ProvisioningFinished, nil
}

func (p *FlashBladeProvisioner) Delete(ctx context.Context, volume *v1.PersistentVolume) error {
	identity, ok := volume.Annotations[PROVISIONER_ANNOTATION]
	if !ok {
		return errors.New("identity annotation not found on PV")
	}
	if identity != PROVISIONER_NAME {
		return &controller.IgnoredError{Reason: "identity annotation on PV does not match ours"}
	}

	err := p.flashBladeApi.DeleteVolume(p.provisionedVolumeName(volume.Name))

	return err
}

func main() {
	syscall.Umask(0)

	flag.Parse()
	flag.Set("logtostderr", "true")

	// Create an InClusterConfig and use it to create a client for the controller
	// to use to communicate with Kubernetes
	config, err := rest.InClusterConfig()
	if err != nil {
		klog.Fatalf("Failed to create config: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Failed to create client: %v", err)
	}

	// Create the provisioner: it implements the Provisioner interface expected by
	// the controller
	hostPathProvisioner := NewFlashBladeProvisioner()

	// Start the provision controller which will dynamically provision hostPath
	// PVs
	pc := controller.NewProvisionController(clientset, PROVISIONER_NAME, hostPathProvisioner)

	// Never stops.
	pc.Run(context.Background())
}
