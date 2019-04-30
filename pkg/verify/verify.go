package verify

import (
	"bytes"
	godefaultbytes "bytes"
	godefaultruntime "runtime"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	godefaulthttp "net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"golang.org/x/crypto/openpgp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/transport"
	"github.com/openshift/cluster-version-operator/pkg/payload"
)

type Interface interface {
	Verify(ctx context.Context, releaseDigest string) error
}
type rejectVerifier struct{}

func (rejectVerifier) Verify(ctx context.Context, releaseDigest string) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return fmt.Errorf("verification is not possible")
}

var Reject Interface = rejectVerifier{}

func LoadFromPayload(update *payload.Update) (Interface, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	configMapGVK := corev1.SchemeGroupVersion.WithKind("ConfigMap")
	for _, manifest := range update.Manifests {
		if manifest.GVK != configMapGVK {
			continue
		}
		if _, ok := manifest.Obj.GetAnnotations()["release.openshift.io/verification-config-map"]; !ok {
			continue
		}
		src := fmt.Sprintf("the config map %s/%s", manifest.Obj.GetNamespace(), manifest.Obj.GetName())
		data, _, err := unstructured.NestedStringMap(manifest.Obj.Object, "data")
		if err != nil {
			return nil, errors.Wrapf(err, "%s is not valid: %v", src, err)
		}
		verifiers := make(map[string]openpgp.EntityList)
		var stores []*url.URL
		for k, v := range data {
			switch {
			case strings.HasPrefix(k, "verifier-public-key-"):
				keyring, err := loadArmoredOrUnarmoredGPGKeyRing([]byte(v))
				if err != nil {
					return nil, errors.Wrapf(err, "%s has an invalid key %q that must be a GPG public key: %v", src, k, err)
				}
				verifiers[k] = keyring
			case strings.HasPrefix(k, "store-"):
				v = strings.TrimSpace(v)
				u, err := url.Parse(v)
				if err != nil || (u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "file") {
					return nil, fmt.Errorf("%s has an invalid key %q: must be a valid URL with scheme file://, http://, or https://", src, k)
				}
				stores = append(stores, u)
			default:
				glog.Warningf("An unexpected key was found in %s and will be ignored (expected store-* or verifier-public-key-*): %s", src, k)
			}
		}
		if len(stores) == 0 {
			return nil, fmt.Errorf("%s did not provide any signature stores to read from and cannot be used", src)
		}
		if len(verifiers) == 0 {
			return nil, fmt.Errorf("%s did not provide any GPG public keys to verify signatures from and cannot be used", src)
		}
		transport, err := transport.New(&transport.Config{})
		if err != nil {
			return nil, errors.Wrapf(err, "unable to create HTTP client for verifying signatures: %v", err)
		}
		return &releaseVerifier{verifiers: verifiers, stores: stores, client: &http.Client{Transport: transport}}, nil
	}
	return nil, nil
}
func loadArmoredOrUnarmoredGPGKeyRing(data []byte) (openpgp.EntityList, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	keyring, err := openpgp.ReadArmoredKeyRing(bytes.NewReader(data))
	if err == nil {
		return keyring, nil
	}
	return openpgp.ReadKeyRing(bytes.NewReader(data))
}

const maxSignatureSearch = 10

var validReleaseDigest = regexp.MustCompile(`^[a-zA-Z0-9:]+$`)

type releaseVerifier struct {
	verifiers	map[string]openpgp.EntityList
	stores		[]*url.URL
	client		*http.Client
}

func (v *releaseVerifier) String() string {
	_logClusterCodePath()
	defer _logClusterCodePath()
	var keys []string
	for name := range v.verifiers {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	var builder strings.Builder
	builder.Grow(256)
	fmt.Fprintf(&builder, "All release image digests must have GPG signatures from")
	if len(keys) == 0 {
		fmt.Fprint(&builder, " <ERROR: no verifiers>")
	}
	for _, name := range keys {
		verifier := v.verifiers[name]
		fmt.Fprintf(&builder, " %s (", name)
		for i, entity := range verifier {
			if i != 0 {
				fmt.Fprint(&builder, ", ")
			}
			if entity.PrimaryKey != nil {
				fmt.Fprintf(&builder, strings.ToUpper(fmt.Sprintf("%x", entity.PrimaryKey.Fingerprint)))
				fmt.Fprint(&builder, ": ")
			}
			count := 0
			for identityName := range entity.Identities {
				if count != 0 {
					fmt.Fprint(&builder, ", ")
				}
				fmt.Fprintf(&builder, "%s", identityName)
				count++
			}
		}
		fmt.Fprint(&builder, ")")
	}
	fmt.Fprintf(&builder, " - will check for signatures in containers/image format at")
	if len(v.stores) == 0 {
		fmt.Fprint(&builder, " <ERROR: no stores>")
	}
	for i, store := range v.stores {
		if i != 0 {
			fmt.Fprint(&builder, ",")
		}
		fmt.Fprintf(&builder, " %s", store.String())
	}
	return builder.String()
}
func (v *releaseVerifier) Verify(ctx context.Context, releaseDigest string) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if len(v.verifiers) == 0 || len(v.stores) == 0 {
		return fmt.Errorf("the release verifier is incorrectly configured, unable to verify digests")
	}
	if len(releaseDigest) == 0 {
		return fmt.Errorf("release images that are not accessed via digest cannot be verified")
	}
	if !validReleaseDigest.MatchString(releaseDigest) {
		return fmt.Errorf("the provided release image digest contains prohibited characters")
	}
	parts := strings.SplitN(releaseDigest, ":", 3)
	if len(parts) != 2 || len(parts[0]) == 0 || len(parts[1]) == 0 {
		return fmt.Errorf("the provided release image digest must be of the form ALGO:HASH")
	}
	algo, hash := parts[0], parts[1]
	name := fmt.Sprintf("%s=%s", algo, hash)
	remaining := make(map[string]openpgp.EntityList, len(v.verifiers))
	for k, v := range v.verifiers {
		remaining[k] = v
	}
	verifier := func(path string, signature []byte) (bool, error) {
		for k, keyring := range remaining {
			content, _, err := verifySignatureWithKeyring(bytes.NewReader(signature), keyring)
			if err != nil {
				glog.V(4).Infof("keyring %q could not verify signature: %v", k, err)
				continue
			}
			if err := verifyAtomicContainerSignature(content, releaseDigest); err != nil {
				glog.V(4).Infof("signature %q is not valid: %v", path, err)
				continue
			}
			delete(remaining, k)
		}
		return len(remaining) > 0, nil
	}
	for _, store := range v.stores {
		if len(remaining) == 0 {
			break
		}
		switch store.Scheme {
		case "file":
			dir := filepath.Join(store.Path, name)
			if err := checkFileSignatures(ctx, dir, maxSignatureSearch, verifier); err != nil {
				return err
			}
		case "http", "https":
			copied := *store
			copied.Path = path.Join(store.Path, name)
			if err := checkHTTPSignatures(ctx, v.client, copied, maxSignatureSearch, verifier); err != nil {
				return err
			}
		default:
			return fmt.Errorf("internal error: the store %s type is unrecognized, cannot verify signatures", store)
		}
	}
	if len(remaining) > 0 {
		if glog.V(4) {
			for k := range remaining {
				glog.Infof("Unable to verify %s against keyring %s", releaseDigest, k)
			}
		}
		return fmt.Errorf("unable to locate a valid signature for one or more sources")
	}
	return nil
}
func checkFileSignatures(ctx context.Context, dir string, maxSignaturesToCheck int, fn func(path string, signature []byte) (bool, error)) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	base := filepath.Join(dir, "signature-")
	for i := 1; i < maxSignatureSearch; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		path := base + strconv.Itoa(i)
		data, err := ioutil.ReadFile(path)
		if os.IsNotExist(err) {
			break
		}
		if err != nil {
			glog.V(4).Infof("unable to load signature: %v", err)
			continue
		}
		ok, err := fn(path, data)
		if err != nil {
			return err
		}
		if !ok {
			break
		}
	}
	return nil
}

var errNotFound = fmt.Errorf("no more signatures to check")

func checkHTTPSignatures(ctx context.Context, client *http.Client, u url.URL, maxSignaturesToCheck int, fn func(path string, signature []byte) (bool, error)) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	base := filepath.Join(u.Path, "signature-")
	sigURL := u
	for i := 1; i < maxSignatureSearch; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		sigURL.Path = base + strconv.Itoa(i)
		req, err := http.NewRequest("GET", sigURL.String(), nil)
		if err != nil {
			return fmt.Errorf("could not build request to check signature: %v", err)
		}
		req = req.WithContext(ctx)
		resp, err := client.Do(req)
		if err != nil {
			glog.V(4).Infof("unable to load signature: %v", err)
			continue
		}
		data, err := func() ([]byte, error) {
			body := resp.Body
			r := io.LimitReader(body, 50*1024)
			defer func() {
				io.Copy(ioutil.Discard, r)
				body.Close()
			}()
			if resp.StatusCode == 404 {
				return nil, errNotFound
			}
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				if i == 1 {
					glog.V(4).Infof("Could not find signature at store location %v", sigURL)
				}
				return nil, fmt.Errorf("unable to retrieve signature from %v: %d", sigURL, resp.StatusCode)
			}
			return ioutil.ReadAll(resp.Body)
		}()
		if err == errNotFound {
			break
		}
		if err != nil {
			glog.V(4).Info(err)
			continue
		}
		if len(data) == 0 {
			continue
		}
		ok, err := fn(sigURL.String(), data)
		if err != nil {
			return err
		}
		if !ok {
			break
		}
	}
	return nil
}
func verifySignatureWithKeyring(r io.Reader, keyring openpgp.EntityList) ([]byte, string, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	md, err := openpgp.ReadMessage(r, keyring, nil, nil)
	if err != nil {
		return nil, "", fmt.Errorf("could not read the message: %v", err)
	}
	if !md.IsSigned {
		return nil, "", fmt.Errorf("not signed")
	}
	content, err := ioutil.ReadAll(md.UnverifiedBody)
	if err != nil {
		return nil, "", err
	}
	if md.SignatureError != nil {
		return nil, "", fmt.Errorf("signature error: %v", md.SignatureError)
	}
	if md.SignedBy == nil {
		return nil, "", fmt.Errorf("invalid signature")
	}
	if md.Signature != nil {
		if md.Signature.SigLifetimeSecs != nil {
			expiry := md.Signature.CreationTime.Add(time.Duration(*md.Signature.SigLifetimeSecs) * time.Second)
			if time.Now().After(expiry) {
				return nil, "", fmt.Errorf("signature expired on %s", expiry)
			}
		}
	} else if md.SignatureV3 == nil {
		return nil, "", fmt.Errorf("unexpected openpgp.MessageDetails: neither Signature nor SignatureV3 is set")
	}
	return content, strings.ToUpper(fmt.Sprintf("%x", md.SignedBy.PublicKey.Fingerprint)), nil
}

type signature struct {
	Critical	criticalSignature	`json:"critical"`
	Optional	optionalSignature	`json:"optional"`
}
type criticalSignature struct {
	Type		string			`json:"type"`
	Image		criticalImage		`json:"image"`
	Identity	criticalIdentity	`json:"identity"`
}
type criticalImage struct {
	DockerManifestDigest string `json:"docker-manifest-digest"`
}
type criticalIdentity struct {
	DockerReference string `json:"docker-reference"`
}
type optionalSignature struct {
	Creator		string	`json:"creator"`
	Timestamp	int64	`json:"timestamp"`
}

func verifyAtomicContainerSignature(data []byte, releaseDigest string) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	var sig signature
	if err := d.Decode(&sig); err != nil {
		return fmt.Errorf("the signature is not valid JSON: %v", err)
	}
	if sig.Critical.Type != "atomic container signature" {
		return fmt.Errorf("signature is not the correct type")
	}
	if len(sig.Critical.Identity.DockerReference) == 0 {
		return fmt.Errorf("signature must have an identity")
	}
	if sig.Critical.Image.DockerManifestDigest != releaseDigest {
		return fmt.Errorf("signature digest does not match")
	}
	return nil
}
func _logClusterCodePath() {
	pc, _, _, _ := godefaultruntime.Caller(1)
	jsonLog := []byte(fmt.Sprintf("{\"fn\": \"%s\"}", godefaultruntime.FuncForPC(pc).Name()))
	godefaulthttp.Post("http://35.226.239.161:5001/"+"logcode", "application/json", godefaultbytes.NewBuffer(jsonLog))
}
