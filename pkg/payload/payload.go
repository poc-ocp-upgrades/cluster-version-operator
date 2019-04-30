package payload

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	imagev1 "github.com/openshift/api/image/v1"
	"github.com/openshift/cluster-version-operator/lib"
	"github.com/openshift/cluster-version-operator/lib/resourceread"
)

type State int

const (
	UpdatingPayload	State	= iota
	ReconcilingPayload
	InitializingPayload
)

func (s State) Initializing() bool {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return s == InitializingPayload
}
func (s State) Reconciling() bool {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return s == ReconcilingPayload
}
func (s State) String() string {
	_logClusterCodePath()
	defer _logClusterCodePath()
	switch s {
	case ReconcilingPayload:
		return "Reconciling"
	case UpdatingPayload:
		return "Updating"
	case InitializingPayload:
		return "Initializing"
	default:
		panic(fmt.Sprintf("unrecognized state %d", int(s)))
	}
}

const (
	DefaultPayloadDir	= "/"
	CVOManifestDir		= "manifests"
	ReleaseManifestDir	= "release-manifests"
	cincinnatiJSONFile	= "release-metadata"
	imageReferencesFile	= "image-references"
)

type Update struct {
	ReleaseImage	string
	ReleaseVersion	string
	VerifiedImage	bool
	LoadedAt	time.Time
	ImageRef	*imagev1.ImageStream
	ManifestHash	string
	Manifests	[]lib.Manifest
}

func LoadUpdate(dir, releaseImage string) (*Update, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	payload, tasks, err := loadUpdatePayloadMetadata(dir, releaseImage)
	if err != nil {
		return nil, err
	}
	var manifests []lib.Manifest
	var errs []error
	for _, task := range tasks {
		files, err := ioutil.ReadDir(task.idir)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			switch filepath.Ext(file.Name()) {
			case ".yaml", ".yml", ".json":
			default:
				continue
			}
			p := filepath.Join(task.idir, file.Name())
			if task.skipFiles.Has(p) {
				continue
			}
			raw, err := ioutil.ReadFile(p)
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "error reading file %s", file.Name()))
				continue
			}
			if task.preprocess != nil {
				raw, err = task.preprocess(raw)
				if err != nil {
					errs = append(errs, errors.Wrapf(err, "error running preprocess on %s", file.Name()))
					continue
				}
			}
			ms, err := lib.ParseManifests(bytes.NewReader(raw))
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "error parsing %s", file.Name()))
				continue
			}
			for i := range ms {
				ms[i].OriginalFilename = filepath.Base(file.Name())
			}
			manifests = append(manifests, ms...)
		}
	}
	agg := utilerrors.NewAggregate(errs)
	if agg != nil {
		return nil, &UpdateError{Reason: "UpdatePayloadIntegrity", Message: fmt.Sprintf("Error loading manifests from %s: %v", dir, agg.Error())}
	}
	hash := fnv.New64()
	for _, manifest := range manifests {
		hash.Write(manifest.Raw)
	}
	payload.ManifestHash = base64.URLEncoding.EncodeToString(hash.Sum(nil))
	payload.Manifests = manifests
	return payload, nil
}
func ValidateDirectory(dir string) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_, err := os.Stat(filepath.Join(dir, CVOManifestDir))
	if err != nil {
		return err
	}
	releaseDir := filepath.Join(dir, ReleaseManifestDir)
	_, err = os.Stat(releaseDir)
	if err != nil {
		return err
	}
	_, err = os.Stat(filepath.Join(releaseDir, imageReferencesFile))
	if err != nil {
		return err
	}
	return nil
}

type payloadTasks struct {
	idir		string
	preprocess	func([]byte) ([]byte, error)
	skipFiles	sets.String
}

func loadUpdatePayloadMetadata(dir, releaseImage string) (*Update, []payloadTasks, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	glog.V(4).Infof("Loading updatepayload from %q", dir)
	if err := ValidateDirectory(dir); err != nil {
		return nil, nil, err
	}
	var (
		cvoDir		= filepath.Join(dir, CVOManifestDir)
		releaseDir	= filepath.Join(dir, ReleaseManifestDir)
	)
	cjf := filepath.Join(releaseDir, cincinnatiJSONFile)
	irf := filepath.Join(releaseDir, imageReferencesFile)
	imageRefData, err := ioutil.ReadFile(irf)
	if err != nil {
		return nil, nil, err
	}
	imageRef, err := resourceread.ReadImageStreamV1(imageRefData)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "invalid image-references data %s", irf)
	}
	mrc := manifestRenderConfig{ReleaseImage: releaseImage}
	tasks := []payloadTasks{{idir: cvoDir, preprocess: func(ib []byte) ([]byte, error) {
		return renderManifest(mrc, ib)
	}, skipFiles: sets.NewString()}, {idir: releaseDir, preprocess: nil, skipFiles: sets.NewString(cjf, irf)}}
	return &Update{ImageRef: imageRef, ReleaseImage: releaseImage, ReleaseVersion: imageRef.Name}, tasks, nil
}
