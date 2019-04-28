package cvo

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"
	"github.com/openshift/cluster-version-operator/pkg/verify"
	"github.com/blang/semver"
	"github.com/golang/glog"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	coreclientsetv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	configv1 "github.com/openshift/api/config/v1"
	clientset "github.com/openshift/client-go/config/clientset/versioned"
	configinformersv1 "github.com/openshift/client-go/config/informers/externalversions/config/v1"
	configlistersv1 "github.com/openshift/client-go/config/listers/config/v1"
	"github.com/openshift/cluster-version-operator/lib"
	"github.com/openshift/cluster-version-operator/lib/resourceapply"
	"github.com/openshift/cluster-version-operator/lib/resourcebuilder"
	"github.com/openshift/cluster-version-operator/lib/validation"
	"github.com/openshift/cluster-version-operator/pkg/cvo/internal"
	"github.com/openshift/cluster-version-operator/pkg/cvo/internal/dynamicclient"
	"github.com/openshift/cluster-version-operator/pkg/payload"
)

const (
	maxRetries = 15
)

type Operator struct {
	nodename			string
	namespace, name			string
	releaseImage			string
	releaseVersion			string
	releaseCreated			time.Time
	client				clientset.Interface
	kubeClient			kubernetes.Interface
	eventRecorder			record.EventRecorder
	minimumUpdateCheckInterval	time.Duration
	payloadDir			string
	defaultUpstreamServer		string
	syncBackoff			wait.Backoff
	cvLister			configlistersv1.ClusterVersionLister
	coLister			configlistersv1.ClusterOperatorLister
	cacheSynced			[]cache.InformerSynced
	queue				workqueue.RateLimitingInterface
	availableUpdatesQueue		workqueue.RateLimitingInterface
	statusLock			sync.Mutex
	availableUpdates		*availableUpdates
	verifier			verify.Interface
	configSync			ConfigSyncWorker
	statusInterval			time.Duration
	lastAtLock			sync.Mutex
	lastResourceVersion		int64
}

func New(nodename string, namespace, name string, releaseImage string, overridePayloadDir string, minimumInterval time.Duration, cvInformer configinformersv1.ClusterVersionInformer, coInformer configinformersv1.ClusterOperatorInformer, client clientset.Interface, kubeClient kubernetes.Interface, enableMetrics bool) *Operator {
	_logClusterCodePath()
	defer _logClusterCodePath()
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&coreclientsetv1.EventSinkImpl{Interface: kubeClient.CoreV1().Events(namespace)})
	optr := &Operator{nodename: nodename, namespace: namespace, name: name, releaseImage: releaseImage, statusInterval: 15 * time.Second, minimumUpdateCheckInterval: minimumInterval, payloadDir: overridePayloadDir, defaultUpstreamServer: "https://api.openshift.com/api/upgrades_info/v1/graph", client: client, kubeClient: kubeClient, eventRecorder: eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: namespace}), queue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "clusterversion"), availableUpdatesQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "availableupdates")}
	cvInformer.Informer().AddEventHandler(optr.eventHandler())
	optr.coLister = coInformer.Lister()
	optr.cacheSynced = append(optr.cacheSynced, coInformer.Informer().HasSynced)
	optr.cvLister = cvInformer.Lister()
	optr.cacheSynced = append(optr.cacheSynced, cvInformer.Informer().HasSynced)
	if enableMetrics {
		if err := optr.registerMetrics(coInformer.Informer()); err != nil {
			panic(err)
		}
	}
	return optr
}
func (optr *Operator) InitializeFromPayload(restConfig *rest.Config, burstRestConfig *rest.Config) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	update, err := payload.LoadUpdate(optr.defaultPayloadDir(), optr.releaseImage)
	if err != nil {
		return fmt.Errorf("the local release contents are invalid - no current version can be determined from disk: %v", err)
	}
	if _, err := semver.Parse(update.ImageRef.Name); err != nil {
		return fmt.Errorf("The local release contents name %q is not a valid semantic version - no current version will be reported: %v", update.ImageRef.Name, err)
	}
	optr.releaseCreated = update.ImageRef.CreationTimestamp.Time
	optr.releaseVersion = update.ImageRef.Name
	verifier, err := verify.LoadFromPayload(update)
	if err != nil {
		return err
	}
	if verifier != nil {
		glog.Infof("Verifying release authenticity: %v", verifier)
	} else {
		glog.Warningf("WARNING: No release authenticity verification is configured, all releases are considered unverified")
		verifier = verify.Reject
	}
	optr.verifier = verifier
	optr.configSync = NewSyncWorker(optr.defaultPayloadRetriever(), NewResourceBuilder(restConfig, burstRestConfig, optr.coLister), optr.minimumUpdateCheckInterval, wait.Backoff{Duration: time.Second * 10, Factor: 1.3, Steps: 3})
	return nil
}
func (optr *Operator) Run(ctx context.Context, workers int) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	defer utilruntime.HandleCrash()
	defer optr.queue.ShutDown()
	stopCh := ctx.Done()
	glog.Infof("Starting ClusterVersionOperator with minimum reconcile period %s", optr.minimumUpdateCheckInterval)
	defer glog.Info("Shutting down ClusterVersionOperator")
	if !cache.WaitForCacheSync(stopCh, optr.cacheSynced...) {
		glog.Info("Caches never synchronized")
		return
	}
	optr.queue.Add(optr.queueKey())
	go runThrottledStatusNotifier(stopCh, optr.statusInterval, 2, optr.configSync.StatusCh(), func() {
		optr.queue.Add(optr.queueKey())
	})
	go optr.configSync.Start(ctx, 16)
	go wait.Until(func() {
		optr.worker(optr.queue, optr.sync)
	}, time.Second, stopCh)
	go wait.Until(func() {
		optr.worker(optr.availableUpdatesQueue, optr.availableUpdatesSync)
	}, time.Second, stopCh)
	<-stopCh
}
func (optr *Operator) queueKey() string {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return fmt.Sprintf("%s/%s", optr.namespace, optr.name)
}
func (optr *Operator) eventHandler() cache.ResourceEventHandler {
	_logClusterCodePath()
	defer _logClusterCodePath()
	workQueueKey := optr.queueKey()
	return cache.ResourceEventHandlerFuncs{AddFunc: func(obj interface{}) {
		optr.queue.Add(workQueueKey)
		optr.availableUpdatesQueue.Add(workQueueKey)
	}, UpdateFunc: func(old, new interface{}) {
		optr.queue.Add(workQueueKey)
		optr.availableUpdatesQueue.Add(workQueueKey)
	}, DeleteFunc: func(obj interface{}) {
		optr.queue.Add(workQueueKey)
	}}
}
func (optr *Operator) worker(queue workqueue.RateLimitingInterface, syncHandler func(string) error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	for processNextWorkItem(queue, syncHandler, optr.syncFailingStatus) {
	}
}

type syncFailingStatusFunc func(config *configv1.ClusterVersion, err error) error

func processNextWorkItem(queue workqueue.RateLimitingInterface, syncHandler func(string) error, syncFailingStatus syncFailingStatusFunc) bool {
	_logClusterCodePath()
	defer _logClusterCodePath()
	key, quit := queue.Get()
	if quit {
		return false
	}
	defer queue.Done(key)
	err := syncHandler(key.(string))
	handleErr(queue, err, key, syncFailingStatus)
	return true
}
func handleErr(queue workqueue.RateLimitingInterface, err error, key interface{}, syncFailingStatus syncFailingStatusFunc) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if err == nil {
		queue.Forget(key)
		return
	}
	if queue.NumRequeues(key) < maxRetries {
		glog.V(2).Infof("Error syncing operator %v: %v", key, err)
		queue.AddRateLimited(key)
		return
	}
	err = syncFailingStatus(nil, err)
	utilruntime.HandleError(err)
	glog.V(2).Infof("Dropping operator %q out of the queue %v: %v", key, queue, err)
	queue.Forget(key)
}
func (optr *Operator) sync(key string) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	startTime := time.Now()
	glog.V(4).Infof("Started syncing cluster version %q (%v)", key, startTime)
	defer func() {
		glog.V(4).Infof("Finished syncing cluster version %q (%v)", key, time.Since(startTime))
	}()
	original, changed, err := optr.getOrCreateClusterVersion()
	if err != nil {
		return err
	}
	if changed {
		glog.V(4).Infof("Cluster version changed, waiting for newer event")
		return nil
	}
	errs := validation.ValidateClusterVersion(original)
	config := validation.ClearInvalidFields(original, errs)
	desired, ok := findUpdateFromConfig(config)
	if ok {
		glog.V(4).Infof("Desired version from spec is %#v", desired)
	} else {
		desired = optr.currentVersion()
		glog.V(4).Infof("Desired version from operator is %#v", desired)
	}
	if len(desired.Image) == 0 {
		return optr.syncStatus(original, config, &SyncWorkerStatus{Failure: fmt.Errorf("No configured operator version, unable to update cluster")}, errs)
	}
	var state payload.State
	switch {
	case hasNeverReachedLevel(config):
		state = payload.InitializingPayload
	case hasReachedLevel(config, desired):
		state = payload.ReconcilingPayload
	default:
		state = payload.UpdatingPayload
	}
	status := optr.configSync.Update(config.Generation, desired, config.Spec.Overrides, state)
	return optr.syncStatus(original, config, status, errs)
}
func (optr *Operator) availableUpdatesSync(key string) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	startTime := time.Now()
	glog.V(4).Infof("Started syncing available updates %q (%v)", key, startTime)
	defer func() {
		glog.V(4).Infof("Finished syncing available updates %q (%v)", key, time.Since(startTime))
	}()
	config, err := optr.cvLister.Get(optr.name)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if errs := validation.ValidateClusterVersion(config); len(errs) > 0 {
		return nil
	}
	return optr.syncAvailableUpdates(config)
}
func (optr *Operator) isOlderThanLastUpdate(config *configv1.ClusterVersion) bool {
	_logClusterCodePath()
	defer _logClusterCodePath()
	i, err := strconv.ParseInt(config.ResourceVersion, 10, 64)
	if err != nil {
		return false
	}
	optr.lastAtLock.Lock()
	defer optr.lastAtLock.Unlock()
	return i < optr.lastResourceVersion
}
func (optr *Operator) rememberLastUpdate(config *configv1.ClusterVersion) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if config == nil {
		return
	}
	i, err := strconv.ParseInt(config.ResourceVersion, 10, 64)
	if err != nil {
		return
	}
	optr.lastAtLock.Lock()
	defer optr.lastAtLock.Unlock()
	optr.lastResourceVersion = i
}
func (optr *Operator) getOrCreateClusterVersion() (*configv1.ClusterVersion, bool, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	obj, err := optr.cvLister.Get(optr.name)
	if err == nil {
		if optr.isOlderThanLastUpdate(obj) {
			return nil, true, nil
		}
		return obj, false, nil
	}
	if !apierrors.IsNotFound(err) {
		return nil, false, err
	}
	var upstream configv1.URL
	if len(optr.defaultUpstreamServer) > 0 {
		u := configv1.URL(optr.defaultUpstreamServer)
		upstream = u
	}
	id, _ := uuid.NewRandom()
	config := &configv1.ClusterVersion{ObjectMeta: metav1.ObjectMeta{Name: optr.name}, Spec: configv1.ClusterVersionSpec{Upstream: upstream, Channel: "fast", ClusterID: configv1.ClusterID(id.String())}}
	actual, _, err := resourceapply.ApplyClusterVersionFromCache(optr.cvLister, optr.client.ConfigV1(), config)
	if apierrors.IsAlreadyExists(err) {
		return nil, true, nil
	}
	return actual, true, err
}
func versionString(update configv1.Update) string {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if len(update.Version) > 0 {
		return update.Version
	}
	if len(update.Image) > 0 {
		return update.Image
	}
	return "<unknown>"
}
func (optr *Operator) currentVersion() configv1.Update {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return configv1.Update{Version: optr.releaseVersion, Image: optr.releaseImage}
}
func (optr *Operator) SetSyncWorkerForTesting(worker ConfigSyncWorker) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	optr.configSync = worker
}

type resourceBuilder struct {
	config			*rest.Config
	burstConfig		*rest.Config
	modifier		resourcebuilder.MetaV1ObjectModifierFunc
	clusterOperators	internal.ClusterOperatorsGetter
}

func NewResourceBuilder(config, burstConfig *rest.Config, clusterOperators configlistersv1.ClusterOperatorLister) payload.ResourceBuilder {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return &resourceBuilder{config: config, burstConfig: burstConfig, clusterOperators: clusterOperators}
}
func (b *resourceBuilder) builderFor(m *lib.Manifest, state payload.State) (resourcebuilder.Interface, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	config := b.config
	if state == payload.InitializingPayload {
		config = b.burstConfig
	}
	if b.clusterOperators != nil && m.GVK == configv1.SchemeGroupVersion.WithKind("ClusterOperator") {
		return internal.NewClusterOperatorBuilder(b.clusterOperators, *m), nil
	}
	if resourcebuilder.Mapper.Exists(m.GVK) {
		return resourcebuilder.New(resourcebuilder.Mapper, config, *m)
	}
	client, err := dynamicclient.New(config, m.GVK, m.Object().GetNamespace())
	if err != nil {
		return nil, err
	}
	return internal.NewGenericBuilder(client, *m)
}
func (b *resourceBuilder) Apply(ctx context.Context, m *lib.Manifest, state payload.State) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	builder, err := b.builderFor(m, state)
	if err != nil {
		return err
	}
	if b.modifier != nil {
		builder = builder.WithModifier(b.modifier)
	}
	return builder.WithMode(stateToMode(state)).Do(ctx)
}
func stateToMode(state payload.State) resourcebuilder.Mode {
	_logClusterCodePath()
	defer _logClusterCodePath()
	switch state {
	case payload.InitializingPayload:
		return resourcebuilder.InitializingMode
	case payload.UpdatingPayload:
		return resourcebuilder.UpdatingMode
	case payload.ReconcilingPayload:
		return resourcebuilder.ReconcilingMode
	default:
		panic(fmt.Sprintf("unexpected payload state %d", int(state)))
	}
}
func hasNeverReachedLevel(cv *configv1.ClusterVersion) bool {
	_logClusterCodePath()
	defer _logClusterCodePath()
	for _, version := range cv.Status.History {
		if version.State == configv1.CompletedUpdate {
			return false
		}
	}
	return true
}
func hasReachedLevel(cv *configv1.ClusterVersion, desired configv1.Update) bool {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if len(cv.Status.History) == 0 {
		return false
	}
	if cv.Status.History[0].State != configv1.CompletedUpdate {
		return false
	}
	return desired.Image == cv.Status.History[0].Image
}
