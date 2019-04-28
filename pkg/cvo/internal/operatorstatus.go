package internal

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	configv1 "github.com/openshift/api/config/v1"
	configclientv1 "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	"github.com/openshift/cluster-version-operator/lib"
	"github.com/openshift/cluster-version-operator/lib/resourcebuilder"
	"github.com/openshift/cluster-version-operator/pkg/payload"
)

var (
	osScheme	= runtime.NewScheme()
	osCodecs	= serializer.NewCodecFactory(osScheme)
	osMapper	= resourcebuilder.NewResourceMapper()
)

func init() {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if err := configv1.AddToScheme(osScheme); err != nil {
		panic(err)
	}
	osMapper.RegisterGVK(configv1.SchemeGroupVersion.WithKind("ClusterOperator"), newClusterOperatorBuilder)
	osMapper.AddToMap(resourcebuilder.Mapper)
}
func readClusterOperatorV1OrDie(objBytes []byte) *configv1.ClusterOperator {
	_logClusterCodePath()
	defer _logClusterCodePath()
	requiredObj, err := runtime.Decode(osCodecs.UniversalDecoder(configv1.SchemeGroupVersion), objBytes)
	if err != nil {
		panic(err)
	}
	return requiredObj.(*configv1.ClusterOperator)
}

type clusterOperatorBuilder struct {
	client		ClusterOperatorsGetter
	raw		[]byte
	modifier	resourcebuilder.MetaV1ObjectModifierFunc
	mode		resourcebuilder.Mode
}

func newClusterOperatorBuilder(config *rest.Config, m lib.Manifest) resourcebuilder.Interface {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return NewClusterOperatorBuilder(clientClusterOperatorsGetter{getter: configclientv1.NewForConfigOrDie(config).ClusterOperators()}, m)
}

type ClusterOperatorsGetter interface {
	Get(name string) (*configv1.ClusterOperator, error)
}
type clientClusterOperatorsGetter struct {
	getter configclientv1.ClusterOperatorInterface
}

func (g clientClusterOperatorsGetter) Get(name string) (*configv1.ClusterOperator, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return g.getter.Get(name, metav1.GetOptions{})
}
func NewClusterOperatorBuilder(client ClusterOperatorsGetter, m lib.Manifest) resourcebuilder.Interface {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return &clusterOperatorBuilder{client: client, raw: m.Raw}
}
func (b *clusterOperatorBuilder) WithMode(m resourcebuilder.Mode) resourcebuilder.Interface {
	_logClusterCodePath()
	defer _logClusterCodePath()
	b.mode = m
	return b
}
func (b *clusterOperatorBuilder) WithModifier(f resourcebuilder.MetaV1ObjectModifierFunc) resourcebuilder.Interface {
	_logClusterCodePath()
	defer _logClusterCodePath()
	b.modifier = f
	return b
}
func (b *clusterOperatorBuilder) Do(ctx context.Context) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	os := readClusterOperatorV1OrDie(b.raw)
	if b.modifier != nil {
		b.modifier(os)
	}
	return waitForOperatorStatusToBeDone(ctx, 1*time.Second, b.client, os, b.mode)
}
func waitForOperatorStatusToBeDone(ctx context.Context, interval time.Duration, client ClusterOperatorsGetter, expected *configv1.ClusterOperator, mode resourcebuilder.Mode) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	var lastErr error
	err := wait.PollImmediateUntil(interval, func() (bool, error) {
		actual, err := client.Get(expected.Name)
		if err != nil {
			lastErr = &payload.UpdateError{Nested: err, Reason: "ClusterOperatorNotAvailable", Message: fmt.Sprintf("Cluster operator %s has not yet reported success", expected.Name), Name: expected.Name}
			return false, nil
		}
		undone := map[string][]string{}
		for _, expOp := range expected.Status.Versions {
			undone[expOp.Name] = []string{expOp.Version}
			for _, actOp := range actual.Status.Versions {
				if actOp.Name == expOp.Name {
					undone[expOp.Name] = append(undone[expOp.Name], actOp.Version)
					if actOp.Version == expOp.Version {
						delete(undone, expOp.Name)
					}
					break
				}
			}
		}
		if len(undone) > 0 {
			var keys []string
			for k := range undone {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			var reports []string
			for _, op := range keys {
				if op == "operator" {
					continue
				}
				ver := undone[op]
				if len(ver) == 1 {
					reports = append(reports, fmt.Sprintf("missing version information for %s", op))
					continue
				}
				reports = append(reports, fmt.Sprintf("upgrading %s from %s to %s", op, ver[1], ver[0]))
			}
			message := fmt.Sprintf("Cluster operator %s is still updating", actual.Name)
			if len(reports) > 0 {
				message = fmt.Sprintf("Cluster operator %s is still updating: %s", actual.Name, strings.Join(reports, ", "))
			}
			lastErr = &payload.UpdateError{Nested: errors.New(lowerFirst(message)), Reason: "ClusterOperatorNotAvailable", Message: message, Name: actual.Name}
			return false, nil
		}
		available := false
		progressing := true
		failing := true
		var failingCondition *configv1.ClusterOperatorStatusCondition
		degradedValue := true
		var degradedCondition *configv1.ClusterOperatorStatusCondition
		for i := range actual.Status.Conditions {
			condition := &actual.Status.Conditions[i]
			switch {
			case condition.Type == configv1.OperatorAvailable && condition.Status == configv1.ConditionTrue:
				available = true
			case condition.Type == configv1.OperatorProgressing && condition.Status == configv1.ConditionFalse:
				progressing = false
			case condition.Type == configv1.ClusterStatusConditionType("Failing"):
				if condition.Status == configv1.ConditionFalse {
					failing = false
				}
				failingCondition = condition
			case condition.Type == configv1.OperatorDegraded:
				if condition.Status == configv1.ConditionFalse {
					degradedValue = false
				}
				degradedCondition = condition
			}
		}
		degraded := failing
		if degradedCondition != nil {
			degraded = degradedValue
		}
		switch mode {
		case resourcebuilder.InitializingMode:
			if available && (!progressing || len(expected.Status.Versions) > 0) {
				return true, nil
			}
		default:
			if available && (!progressing || len(expected.Status.Versions) > 0) && !degraded {
				return true, nil
			}
		}
		condition := failingCondition
		if degradedCondition != nil {
			condition = degradedCondition
		}
		if condition != nil && condition.Status == configv1.ConditionTrue {
			message := fmt.Sprintf("Cluster operator %s is reporting a failure", actual.Name)
			if len(condition.Message) > 0 {
				message = fmt.Sprintf("Cluster operator %s is reporting a failure: %s", actual.Name, condition.Message)
			}
			lastErr = &payload.UpdateError{Nested: errors.New(lowerFirst(message)), Reason: "ClusterOperatorDegraded", Message: message, Name: actual.Name}
			return false, nil
		}
		lastErr = &payload.UpdateError{Nested: fmt.Errorf("cluster operator %s is not done; it is available=%v, progressing=%v, degraded=%v", actual.Name, available, progressing, degraded), Reason: "ClusterOperatorNotAvailable", Message: fmt.Sprintf("Cluster operator %s has not yet reported success", actual.Name), Name: actual.Name}
		return false, nil
	}, ctx.Done())
	if err != nil {
		if err == wait.ErrWaitTimeout && lastErr != nil {
			return lastErr
		}
		return err
	}
	return nil
}
func lowerFirst(str string) string {
	_logClusterCodePath()
	defer _logClusterCodePath()
	for i, v := range str {
		return string(unicode.ToLower(v)) + str[i+1:]
	}
	return ""
}
