package sequoia

import (
	"strings"
)

type Config struct {
	Client       string
	Scope        string
	Test         string
	Provider     string
	SkipSetup    bool `yaml:"skip_setup"`
	SkipTest     bool `yaml:"skip_test"`
	SkipTeardown bool `yaml:"skip_teardown"`
	Repeat       int
}

func NewConfigSpec(fileName string) Config {
	var config Config
	ReadYamlFile(fileName, &config)
	return config
}

type BucketSpec struct {
	Name     string
	Names    []string
	Count    uint8
	Ram      string
	Replica  uint8
	Type     string
	Sasl     string
	Eviction string
}

type ServerSpec struct {
	Name         string
	Names        []string
	Count        uint8
	Ram          string
	IndexRam     string `yaml:"index_ram"`
	RestUsername string `yaml:"rest_username"`
	RestPassword string `yaml:"rest_password"`
	RestPort     string `yaml:"rest_port"`
	InitNodes    uint8  `yaml:"init_nodes"`
	DataPath     string `yaml:"data_path"`
	IndexPath    string `yaml:"index_path"`
	IndexStorage string `yaml:"index_storage"`
	Buckets      string
	BucketSpecs  []BucketSpec
	NodesActive  uint8
	Services     map[string]uint8
	NodeServices map[string][]string
}

func (s *ServerSpec) InitNodeServices() {

	var i uint8
	numNodes := s.Count
	numIndexNodes := s.Services["index"]
	numQueryNodes := s.Services["query"]
	numDataNodes := s.Services["data"]
	s.NodeServices = make(map[string][]string)

	// Spread Strategy
	// make first set of nodes data
	// and second set index to avoid
	// overlapping if possible when specific
	// number of service types provided
	indexStartPos := numNodes - numQueryNodes - numIndexNodes
	if indexStartPos < 0 {
		indexStartPos = 0
	}

	queryStartPos := numNodes - numQueryNodes
	if queryStartPos < 0 {
		queryStartPos = 0
	}

	for i = 0; i < numNodes; i = i + 1 {
		name := s.Names[i]
		s.NodeServices[name] = []string{}
		if numDataNodes > 0 {
			s.NodeServices[name] = append(s.NodeServices[name], "data")
			numDataNodes--
		}
		if i >= indexStartPos && numIndexNodes > 0 {
			s.NodeServices[name] = append(s.NodeServices[name], "index")
			numIndexNodes--
		}
		if i >= queryStartPos && numQueryNodes > 0 {
			s.NodeServices[name] = append(s.NodeServices[name], "query")
			numQueryNodes--
		}
		// must have at least data service
		if len(s.NodeServices[name]) == 0 {
			s.NodeServices[name] = append(s.NodeServices[name], "data")
		}
	}
}

type ScopeSpec struct {
	Buckets []BucketSpec
	Servers []ServerSpec
}

func (s *ScopeSpec) ApplyToAllServers(operation func(string, *ServerSpec)) {
	s.ApplyToServers(operation, 0, 0)
}

func (s *ScopeSpec) ApplyToServers(operation func(string, *ServerSpec),
	startIdx int, endIdx int) {

	useLen := false
	if endIdx == 0 {
		useLen = true
	}

	for i, server := range s.Servers {
		if useLen {
			endIdx = len(server.Names)
		}
		for _, serverName := range server.Names[startIdx:endIdx] {
			operation(serverName, &server)
			s.Servers[i] = server // allowed apply func to modify server
		}
	}
}

func (s *ScopeSpec) ToAttr(attr string) string {

	switch attr {

	case "rest_username":
		return "RestUsername"
	case "rest_password":
		return "RestPassword"
	case "name":
		return "Name"
	case "ram":
		return "Ram"
	case "rest_port":
		return "RestPort"
	}

	return ""
}

func NewScopeSpec(fileName string) ScopeSpec {

	// init from yaml
	var spec ScopeSpec
	ReadYamlFile(fileName, &spec)

	// init bucket section of spec
	bucketNameMap := make(map[string]BucketSpec)
	for i, bucket := range spec.Buckets {
		spec.Buckets[i].Names = ExpandName(bucket.Name, bucket.Count)
		if spec.Buckets[i].Type == "" {
			spec.Buckets[i].Type = "couchbase"
		}
		if spec.Buckets[i].Replica == 0 {
			spec.Buckets[i].Replica = 1
		}
		bucketNameMap[bucket.Name] = spec.Buckets[i]
	}

	// init server section of spec
	for i, server := range spec.Servers {
		spec.Servers[i].Names = ExpandName(server.Name, server.Count)
		spec.Servers[i].BucketSpecs = make([]BucketSpec, 0)
		// map server buckets to bucket objects
		bucketList := strings.Split(spec.Servers[i].Buckets, ",")
		for _, bucketName := range bucketList {
			if bucketSpec, ok := bucketNameMap[bucketName]; ok {
				spec.Servers[i].BucketSpecs = append(spec.Servers[i].BucketSpecs, bucketSpec)
			}
		}
		// init node services
		spec.Servers[i].InitNodeServices()
	}

	return spec
}
