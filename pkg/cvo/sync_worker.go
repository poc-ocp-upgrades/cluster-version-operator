package cvo

import (
	"context"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/time/rate"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/cluster-version-operator/lib"
	"github.com/openshift/cluster-version-operator/lib/resourcebuilder"
	"github.com/openshift/cluster-version-operator/pkg/payload"
)

type ConfigSyncWorker interface {
	Start(ctx context.Context, maxWorkers int)
	Update(generation int64, desired configv1.Update, overrides []configv1.ComponentOverride, state payload.State) *SyncWorkerStatus
	StatusCh() <-chan SyncWorkerStatus
}
type PayloadInfo struct {
	Directory		string
	Local			bool
	Verified		bool
	VerificationError	error
}
type PayloadRetriever interface {
	RetrievePayload(ctx context.Context, desired configv1.Update) (PayloadInfo, error)
}
type StatusReporter interface{ Report(status SyncWorkerStatus) }
type SyncWork struct {
	Generation	int64
	Desired		configv1.Update
	Overrides	[]configv1.ComponentOverride
	State		payload.State
	Completed	int
	Attempt		int
}

func (w SyncWork) Empty() bool {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return len(w.Desired.Image) == 0
}

type SyncWorkerStatus struct {
	Generation	int64
	Step		string
	Failure		error
	Fraction	float32
	Completed	int
	Reconciling	bool
	Initial		bool
	VersionHash	string
	LastProgress	time.Time
	Actual		configv1.Update
	Verified	bool
}

func (w SyncWorkerStatus) DeepCopy() *SyncWorkerStatus {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return &w
}

type SyncWorker struct {
	backoff				wait.Backoff
	retriever			PayloadRetriever
	builder				payload.ResourceBuilder
	reconciling			bool
	minimumReconcileInterval	time.Duration
	notify				chan struct{}
	report				chan SyncWorkerStatus
	lock				sync.Mutex
	work				*SyncWork
	cancelFn			func()
	status				SyncWorkerStatus
	payload				*payload.Update
}

func NewSyncWorker(retriever PayloadRetriever, builder payload.ResourceBuilder, reconcileInterval time.Duration, backoff wait.Backoff) ConfigSyncWorker {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return &SyncWorker{retriever: retriever, builder: builder, backoff: backoff, minimumReconcileInterval: reconcileInterval, notify: make(chan struct{}, 1), report: make(chan SyncWorkerStatus, 500)}
}
func (w *SyncWorker) StatusCh() <-chan SyncWorkerStatus {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return w.report
}
func (w *SyncWorker) Update(generation int64, desired configv1.Update, overrides []configv1.ComponentOverride, state payload.State) *SyncWorkerStatus {
	_logClusterCodePath()
	defer _logClusterCodePath()
	w.lock.Lock()
	defer w.lock.Unlock()
	work := &SyncWork{Generation: generation, Desired: desired, Overrides: overrides}
	if w.work != nil {
		w.work.Generation = generation
	}
	if work.Empty() || equalSyncWork(w.work, work) {
		return w.status.DeepCopy()
	}
	if w.work == nil {
		work.State = state
		w.status = SyncWorkerStatus{Generation: generation, Reconciling: state.Reconciling(), Actual: work.Desired}
	}
	w.work = work
	if w.cancelFn != nil {
		w.cancelFn()
		w.cancelFn = nil
	}
	select {
	case w.notify <- struct{}{}:
	default:
	}
	return w.status.DeepCopy()
}
func (w *SyncWorker) Start(ctx context.Context, maxWorkers int) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	glog.V(5).Infof("Starting sync worker")
	work := &SyncWork{}
	wait.Until(func() {
		consecutiveErrors := 0
		errorInterval := w.minimumReconcileInterval / 16
		var next <-chan time.Time
		for {
			waitingToReconcile := work.State == payload.ReconcilingPayload
			select {
			case <-ctx.Done():
				glog.V(5).Infof("Stopped worker")
				return
			case <-next:
				waitingToReconcile = false
				glog.V(5).Infof("Wait finished")
			case <-w.notify:
				glog.V(5).Infof("Work updated")
			}
			changed := w.calculateNext(work)
			if !changed && waitingToReconcile {
				glog.V(5).Infof("No change, waiting")
				continue
			}
			if work.Empty() {
				next = time.After(w.minimumReconcileInterval)
				glog.V(5).Infof("No work, waiting")
				continue
			}
			err := func() error {
				var syncTimeout time.Duration
				switch work.State {
				case payload.InitializingPayload:
					syncTimeout = w.minimumReconcileInterval
				case payload.UpdatingPayload:
					syncTimeout = w.minimumReconcileInterval * 2
				default:
					syncTimeout = w.minimumReconcileInterval * 2
				}
				ctx, cancelFn := context.WithTimeout(ctx, syncTimeout)
				w.lock.Lock()
				w.cancelFn = cancelFn
				w.lock.Unlock()
				defer cancelFn()
				reporter := &statusWrapper{w: w, previousStatus: w.Status()}
				glog.V(5).Infof("Previous sync status: %#v", reporter.previousStatus)
				return w.syncOnce(ctx, work, maxWorkers, reporter)
			}()
			if err != nil {
				consecutiveErrors++
				interval := w.minimumReconcileInterval
				if consecutiveErrors < 4 {
					interval = errorInterval
					for i := 0; i < consecutiveErrors; i++ {
						interval *= 2
					}
				}
				next = time.After(wait.Jitter(interval, 0.2))
				utilruntime.HandleError(fmt.Errorf("unable to synchronize image (waiting %s): %v", interval, err))
				continue
			}
			glog.V(5).Infof("Sync succeeded, reconciling")
			work.Completed++
			work.State = payload.ReconcilingPayload
			next = time.After(w.minimumReconcileInterval)
		}
	}, 10*time.Millisecond, ctx.Done())
	glog.V(5).Infof("Worker shut down")
}

type statusWrapper struct {
	w		*SyncWorker
	previousStatus	*SyncWorkerStatus
}

func (w *statusWrapper) Report(status SyncWorkerStatus) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	p := w.previousStatus
	if p.Failure != nil && status.Failure == nil {
		if p.Actual == status.Actual {
			if status.Fraction < p.Fraction {
				glog.V(5).Infof("Dropping status report from earlier in sync loop")
				return
			}
		}
	}
	if status.Fraction > p.Fraction || status.Completed > p.Completed || (status.Failure == nil && status.Actual != p.Actual) {
		status.LastProgress = time.Now()
	}
	if status.Generation == 0 {
		status.Generation = p.Generation
	} else if status.Generation < p.Generation {
		glog.Warningf("Received a Generation(%d) lower than previously known Generation(%d), this is most probably an internal error", status.Generation, p.Generation)
	}
	w.w.updateStatus(status)
}
func (w *SyncWorker) calculateNext(work *SyncWork) bool {
	_logClusterCodePath()
	defer _logClusterCodePath()
	w.lock.Lock()
	defer w.lock.Unlock()
	changed := !equalSyncWork(w.work, work)
	if work.Empty() {
		work.State = w.work.State
	} else {
		if changed {
			work.State = payload.UpdatingPayload
		}
	}
	if work.State != payload.ReconcilingPayload {
		work.Completed = 0
	}
	if changed || w.work.State != work.State {
		work.Attempt = 0
	} else {
		work.Attempt++
	}
	if w.work != nil {
		work.Desired = w.work.Desired
		work.Overrides = w.work.Overrides
	}
	work.Generation = w.work.Generation
	return changed
}
func equalUpdate(a, b configv1.Update) bool {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if a.Force != b.Force {
		return false
	}
	return a.Image == b.Image
}
func equalSyncWork(a, b *SyncWork) bool {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if a == b {
		return true
	}
	if (a == nil && b != nil) || (a != nil && b == nil) {
		return false
	}
	return equalUpdate(a.Desired, b.Desired) && reflect.DeepEqual(a.Overrides, b.Overrides)
}
func (w *SyncWorker) updateStatus(update SyncWorkerStatus) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	w.lock.Lock()
	defer w.lock.Unlock()
	glog.V(5).Infof("Status change %#v", update)
	w.status = update
	select {
	case w.report <- update:
	default:
		if glog.V(5) {
			glog.Infof("Status report channel was full %#v", update)
		}
	}
}
func (w *SyncWorker) Desired() configv1.Update {
	_logClusterCodePath()
	defer _logClusterCodePath()
	w.lock.Lock()
	defer w.lock.Unlock()
	if w.work == nil {
		return configv1.Update{}
	}
	return w.work.Desired
}
func (w *SyncWorker) Status() *SyncWorkerStatus {
	_logClusterCodePath()
	defer _logClusterCodePath()
	w.lock.Lock()
	defer w.lock.Unlock()
	return w.status.DeepCopy()
}
func (w *SyncWorker) syncOnce(ctx context.Context, work *SyncWork, maxWorkers int, reporter StatusReporter) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	glog.V(4).Infof("Running sync %s (force=%t) on generation %d in state %s at attempt %d", versionString(work.Desired), work.Desired.Force, work.Generation, work.State, work.Attempt)
	update := work.Desired
	validPayload := w.payload
	if validPayload == nil || !equalUpdate(configv1.Update{Image: validPayload.ReleaseImage}, update) {
		glog.V(4).Infof("Loading payload")
		reporter.Report(SyncWorkerStatus{Generation: work.Generation, Step: "RetrievePayload", Initial: work.State.Initializing(), Reconciling: work.State.Reconciling(), Actual: update})
		info, err := w.retriever.RetrievePayload(ctx, update)
		if err != nil {
			reporter.Report(SyncWorkerStatus{Generation: work.Generation, Failure: err, Step: "RetrievePayload", Initial: work.State.Initializing(), Reconciling: work.State.Reconciling(), Actual: update})
			return err
		}
		payloadUpdate, err := payload.LoadUpdate(info.Directory, update.Image)
		if err != nil {
			reporter.Report(SyncWorkerStatus{Generation: work.Generation, Failure: err, Step: "VerifyPayload", Initial: work.State.Initializing(), Reconciling: work.State.Reconciling(), Actual: update, Verified: info.Verified})
			return err
		}
		payloadUpdate.VerifiedImage = info.Verified
		payloadUpdate.LoadedAt = time.Now()
		w.payload = payloadUpdate
		glog.V(4).Infof("Payload loaded from %s with hash %s", payloadUpdate.ReleaseImage, payloadUpdate.ManifestHash)
	}
	return w.apply(ctx, w.payload, work, maxWorkers, reporter)
}
func (w *SyncWorker) apply(ctx context.Context, payloadUpdate *payload.Update, work *SyncWork, maxWorkers int, reporter StatusReporter) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	update := configv1.Update{Version: payloadUpdate.ReleaseVersion, Image: payloadUpdate.ReleaseImage, Force: work.Desired.Force}
	version := payloadUpdate.ReleaseVersion
	total := len(payloadUpdate.Manifests)
	cr := &consistentReporter{status: SyncWorkerStatus{Generation: work.Generation, Initial: work.State.Initializing(), Reconciling: work.State.Reconciling(), VersionHash: payloadUpdate.ManifestHash, Actual: update, Verified: payloadUpdate.VerifiedImage}, completed: work.Completed, version: version, total: total, reporter: reporter}
	var tasks []*payload.Task
	backoff := w.backoff
	if backoff.Steps > 1 && work.State == payload.InitializingPayload {
		backoff = wait.Backoff{Steps: 4, Factor: 2, Duration: time.Second}
	}
	for i := range payloadUpdate.Manifests {
		tasks = append(tasks, &payload.Task{Index: i + 1, Total: total, Manifest: &payloadUpdate.Manifests[i], Backoff: backoff})
	}
	graph := payload.NewTaskGraph(tasks)
	graph.Split(payload.SplitOnJobs)
	switch work.State {
	case payload.InitializingPayload:
		graph.Parallelize(payload.FlattenByNumberAndComponent)
		maxWorkers = len(graph.Nodes)
	case payload.ReconcilingPayload:
		r := rand.New(rand.NewSource(int64(work.Attempt)))
		graph.Parallelize(payload.PermuteOrder(payload.FlattenByNumberAndComponent, r))
		maxWorkers = 2
	default:
		graph.Parallelize(payload.ByNumberAndComponent)
	}
	errs := payload.RunGraph(ctx, graph, maxWorkers, func(ctx context.Context, tasks []*payload.Task) error {
		for _, task := range tasks {
			if contextIsCancelled(ctx) {
				return cr.CancelError()
			}
			cr.Update()
			glog.V(4).Infof("Running sync for %s", task)
			glog.V(5).Infof("Manifest: %s", string(task.Manifest.Raw))
			ov, ok := getOverrideForManifest(work.Overrides, task.Manifest)
			if ok && ov.Unmanaged {
				glog.V(4).Infof("Skipping %s as unmanaged", task)
				continue
			}
			if err := task.Run(ctx, version, w.builder, work.State); err != nil {
				return err
			}
			cr.Inc()
			glog.V(4).Infof("Done syncing for %s", task)
		}
		return nil
	})
	if len(errs) > 0 {
		err := cr.Errors(errs)
		return err
	}
	cr.Complete()
	return nil
}

var (
	metricPayload = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "cluster_version_payload", Help: "Report the number of entries in the payload."}, []string{"version", "type"})
)

func init() {
	_logClusterCodePath()
	defer _logClusterCodePath()
	prometheus.MustRegister(metricPayload)
}

type errCanceled struct{ err error }

func (e errCanceled) Error() string {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return e.err.Error()
}

type consistentReporter struct {
	lock		sync.Mutex
	status		SyncWorkerStatus
	version		string
	completed	int
	total		int
	done		int
	reporter	StatusReporter
}

func (r *consistentReporter) Inc() {
	_logClusterCodePath()
	defer _logClusterCodePath()
	r.lock.Lock()
	defer r.lock.Unlock()
	r.done++
}
func (r *consistentReporter) Update() {
	_logClusterCodePath()
	defer _logClusterCodePath()
	r.lock.Lock()
	defer r.lock.Unlock()
	metricPayload.WithLabelValues(r.version, "pending").Set(float64(r.total - r.done))
	metricPayload.WithLabelValues(r.version, "applied").Set(float64(r.done))
	copied := r.status
	copied.Step = "ApplyResources"
	copied.Fraction = float32(r.done) / float32(r.total)
	r.reporter.Report(copied)
}
func (r *consistentReporter) Error(err error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	r.lock.Lock()
	defer r.lock.Unlock()
	copied := r.status
	copied.Step = "ApplyResources"
	copied.Fraction = float32(r.done) / float32(r.total)
	if !isCancelledError(err) {
		copied.Failure = err
	}
	r.reporter.Report(copied)
}
func (r *consistentReporter) Errors(errs []error) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	err := summarizeTaskGraphErrors(errs)
	r.lock.Lock()
	defer r.lock.Unlock()
	copied := r.status
	copied.Step = "ApplyResources"
	copied.Fraction = float32(r.done) / float32(r.total)
	if err != nil {
		copied.Failure = err
	}
	r.reporter.Report(copied)
	return err
}
func (r *consistentReporter) CancelError() error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	r.lock.Lock()
	defer r.lock.Unlock()
	return errCanceled{fmt.Errorf("update was cancelled at %d/%d", r.done, r.total)}
}
func (r *consistentReporter) Complete() {
	_logClusterCodePath()
	defer _logClusterCodePath()
	r.lock.Lock()
	defer r.lock.Unlock()
	metricPayload.WithLabelValues(r.version, "pending").Set(float64(r.total - r.done))
	metricPayload.WithLabelValues(r.version, "applied").Set(float64(r.done))
	copied := r.status
	copied.Completed = r.completed + 1
	copied.Initial = false
	copied.Reconciling = true
	copied.Fraction = 1
	r.reporter.Report(copied)
}
func isCancelledError(err error) bool {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if err == nil {
		return false
	}
	_, ok := err.(errCanceled)
	return ok
}
func isImageVerificationError(err error) bool {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if err == nil {
		return false
	}
	updateErr, ok := err.(*payload.UpdateError)
	if !ok {
		return false
	}
	return updateErr.Reason == "ImageVerificationFailed"
}
func summarizeTaskGraphErrors(errs []error) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	err := errors.FilterOut(errors.NewAggregate(errs), isCancelledError)
	if err == nil {
		glog.V(4).Infof("All errors were cancellation errors: %v", errs)
		return nil
	}
	agg, ok := err.(errors.Aggregate)
	if !ok {
		errs = []error{err}
	} else {
		errs = agg.Errors()
	}
	if glog.V(4) {
		glog.Infof("Summarizing %d errors", len(errs))
		for _, err := range errs {
			if uErr, ok := err.(*payload.UpdateError); ok {
				if uErr.Task != nil {
					glog.Infof("Update error %d/%d: %s %s (%T: %v)", uErr.Task.Index, uErr.Task.Total, uErr.Reason, uErr.Message, uErr.Nested, uErr.Nested)
				} else {
					glog.Infof("Update error: %s %s (%T: %v)", uErr.Reason, uErr.Message, uErr.Nested, uErr.Nested)
				}
			} else {
				glog.Infof("Update error: %T: %v", err, err)
			}
		}
	}
	if len(errs) == 1 {
		return errs[0]
	}
	if err := newClusterOperatorsNotAvailable(errs); err != nil {
		return err
	}
	return newMultipleError(errs)
}
func newClusterOperatorsNotAvailable(errs []error) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	names := make([]string, 0, len(errs))
	for _, err := range errs {
		uErr, ok := err.(*payload.UpdateError)
		if !ok || uErr.Reason != "ClusterOperatorNotAvailable" {
			return nil
		}
		if len(uErr.Name) > 0 {
			names = append(names, uErr.Name)
		}
	}
	if len(names) == 0 {
		return nil
	}
	nested := make([]error, 0, len(errs))
	for _, err := range errs {
		nested = append(nested, err)
	}
	sort.Strings(names)
	name := strings.Join(names, ", ")
	return &payload.UpdateError{Nested: errors.NewAggregate(errs), Reason: "ClusterOperatorsNotAvailable", Message: fmt.Sprintf("Some cluster operators are still updating: %s", name), Name: name}
}
func uniqueStrings(arr []string) []string {
	_logClusterCodePath()
	defer _logClusterCodePath()
	var last int
	for i := 1; i < len(arr); i++ {
		if arr[i] == arr[last] {
			continue
		}
		last++
		if last != i {
			arr[last] = arr[i]
		}
	}
	if last < len(arr) {
		last++
	}
	return arr[:last]
}
func newMultipleError(errs []error) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}
	messages := make([]string, 0, len(errs))
	for _, err := range errs {
		messages = append(messages, err.Error())
	}
	sort.Strings(messages)
	messages = uniqueStrings(messages)
	if len(messages) == 0 {
		return errs[0]
	}
	return &payload.UpdateError{Nested: errors.NewAggregate(errs), Reason: "MultipleErrors", Message: fmt.Sprintf("Multiple errors are preventing progress:\n* %s", strings.Join(messages, "\n* "))}
}
func getOverrideForManifest(overrides []configv1.ComponentOverride, manifest *lib.Manifest) (configv1.ComponentOverride, bool) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	for idx, ov := range overrides {
		kind, namespace, name := manifest.GVK.Kind, manifest.Object().GetNamespace(), manifest.Object().GetName()
		if ov.Kind == kind && (namespace == "" || ov.Namespace == namespace) && ov.Name == name {
			return overrides[idx], true
		}
	}
	return configv1.ComponentOverride{}, false
}

var ownerKind = configv1.SchemeGroupVersion.WithKind("ClusterVersion")

func ownerRefModifier(config *configv1.ClusterVersion) resourcebuilder.MetaV1ObjectModifierFunc {
	_logClusterCodePath()
	defer _logClusterCodePath()
	oref := metav1.NewControllerRef(config, ownerKind)
	return func(obj metav1.Object) {
		obj.SetOwnerReferences([]metav1.OwnerReference{*oref})
	}
}
func contextIsCancelled(ctx context.Context) bool {
	_logClusterCodePath()
	defer _logClusterCodePath()
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
func runThrottledStatusNotifier(stopCh <-chan struct{}, interval time.Duration, bucket int, ch <-chan SyncWorkerStatus, fn func()) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	throttle := rate.NewLimiter(rate.Every(interval), bucket)
	wait.Until(func() {
		ctx := context.Background()
		var last SyncWorkerStatus
		for {
			select {
			case <-stopCh:
				return
			case next := <-ch:
				if next.Generation == last.Generation && next.Actual == last.Actual && next.Reconciling == last.Reconciling && (next.Failure != nil) == (last.Failure != nil) {
					if err := throttle.Wait(ctx); err != nil {
						utilruntime.HandleError(fmt.Errorf("unable to throttle status notification: %v", err))
					}
				}
				last = next
				fn()
			}
		}
	}, 1*time.Second, stopCh)
}
