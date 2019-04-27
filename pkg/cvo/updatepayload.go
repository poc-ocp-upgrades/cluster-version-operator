package cvo

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	randutil "k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/pointer"
	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/cluster-version-operator/lib/resourcebuilder"
	"github.com/openshift/cluster-version-operator/pkg/payload"
)

func (optr *Operator) defaultPayloadDir() string {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	if len(optr.payloadDir) == 0 {
		return payload.DefaultPayloadDir
	}
	return optr.payloadDir
}
func (optr *Operator) defaultPayloadRetriever() PayloadRetriever {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return &payloadRetriever{kubeClient: optr.kubeClient, operatorName: optr.name, releaseImage: optr.releaseImage, namespace: optr.namespace, nodeName: optr.nodename, payloadDir: optr.defaultPayloadDir(), workingDir: targetUpdatePayloadsDir}
}

const (
	targetUpdatePayloadsDir = "/etc/cvo/updatepayloads"
)

type payloadRetriever struct {
	releaseImage	string
	payloadDir	string
	kubeClient	kubernetes.Interface
	workingDir	string
	namespace	string
	nodeName	string
	operatorName	string
}

func (r *payloadRetriever) RetrievePayload(ctx context.Context, update configv1.Update) (string, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	if r.releaseImage == update.Image {
		return r.payloadDir, nil
	}
	if len(update.Image) == 0 {
		return "", fmt.Errorf("no payload image has been specified and the contents of the payload cannot be retrieved")
	}
	tdir, err := r.targetUpdatePayloadDir(ctx, update)
	if err != nil {
		return "", &payload.UpdateError{Reason: "UpdatePayloadRetrievalFailed", Message: fmt.Sprintf("Unable to download and prepare the update: %v", err)}
	}
	return tdir, nil
}
func (r *payloadRetriever) targetUpdatePayloadDir(ctx context.Context, update configv1.Update) (string, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	hash := md5.New()
	hash.Write([]byte(update.Image))
	payloadHash := base64.RawURLEncoding.EncodeToString(hash.Sum(nil))
	tdir := filepath.Join(r.workingDir, payloadHash)
	err := payload.ValidateDirectory(tdir)
	if os.IsNotExist(err) {
		err = r.fetchUpdatePayloadToDir(ctx, tdir, update)
	}
	if err != nil {
		return "", err
	}
	if err := payload.ValidateDirectory(tdir); err != nil {
		return "", err
	}
	return tdir, nil
}
func (r *payloadRetriever) fetchUpdatePayloadToDir(ctx context.Context, dir string, update configv1.Update) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	var (
		version		= update.Version
		payload		= update.Image
		name		= fmt.Sprintf("%s-%s-%s", r.operatorName, version, randutil.String(5))
		namespace	= r.namespace
		deadline	= pointer.Int64Ptr(2 * 60)
		nodeSelectorKey	= "node-role.kubernetes.io/master"
		nodename	= r.nodeName
		cmd		= []string{"/bin/sh"}
		args		= []string{"-c", copyPayloadCmd(dir)}
	)
	job := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}, Spec: batchv1.JobSpec{ActiveDeadlineSeconds: deadline, Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "payload", Image: payload, Command: cmd, Args: args, VolumeMounts: []corev1.VolumeMount{{MountPath: targetUpdatePayloadsDir, Name: "payloads"}}, SecurityContext: &corev1.SecurityContext{Privileged: pointer.BoolPtr(true)}}}, Volumes: []corev1.Volume{{Name: "payloads", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: targetUpdatePayloadsDir}}}}, NodeName: nodename, NodeSelector: map[string]string{nodeSelectorKey: ""}, Tolerations: []corev1.Toleration{{Key: nodeSelectorKey}}, RestartPolicy: corev1.RestartPolicyOnFailure}}}}
	_, err := r.kubeClient.BatchV1().Jobs(job.Namespace).Create(job)
	if err != nil {
		return err
	}
	return resourcebuilder.WaitForJobCompletion(ctx, r.kubeClient.BatchV1(), job)
}
func copyPayloadCmd(tdir string) string {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	var (
		fromCVOPath	= filepath.Join(payload.DefaultPayloadDir, payload.CVOManifestDir)
		toCVOPath	= filepath.Join(tdir, payload.CVOManifestDir)
		cvoCmd		= fmt.Sprintf("mkdir -p %s && mv %s %s", tdir, fromCVOPath, toCVOPath)
		fromReleasePath	= filepath.Join(payload.DefaultPayloadDir, payload.ReleaseManifestDir)
		toReleasePath	= filepath.Join(tdir, payload.ReleaseManifestDir)
		releaseCmd	= fmt.Sprintf("mkdir -p %s && mv %s %s", tdir, fromReleasePath, toReleasePath)
	)
	return fmt.Sprintf("%s && %s", cvoCmd, releaseCmd)
}
func findUpdateFromConfig(config *configv1.ClusterVersion) (configv1.Update, bool) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	update := config.Spec.DesiredUpdate
	if update == nil {
		return configv1.Update{}, false
	}
	if len(update.Image) == 0 {
		return findUpdateFromConfigVersion(config, update.Version)
	}
	return *update, true
}
func findUpdateFromConfigVersion(config *configv1.ClusterVersion, version string) (configv1.Update, bool) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	for _, update := range config.Status.AvailableUpdates {
		if update.Version == version {
			return update, len(update.Image) > 0
		}
	}
	for _, history := range config.Status.History {
		if history.Version == version {
			return configv1.Update{Image: history.Image, Version: history.Version}, len(history.Image) > 0
		}
	}
	return configv1.Update{}, false
}
