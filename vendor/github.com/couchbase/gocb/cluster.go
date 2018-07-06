package gocb

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/opentracing/opentracing-go"
	"gopkg.in/couchbase/gocbcore.v7"
	"gopkg.in/couchbaselabs/gocbconnstr.v1"
)

// Cluster represents a connection to a specific Couchbase cluster.
type Cluster struct {
	auth             Authenticator
	agentConfig      gocbcore.AgentConfig
	n1qlTimeout      time.Duration
	ftsTimeout       time.Duration
	analyticsTimeout time.Duration

	clusterLock sync.RWMutex
	queryCache  map[string]*n1qlCache
	bucketList  []*Bucket
	httpCli     *http.Client

	analyticsHosts []string
}

// Connect creates a new Cluster object for a specific cluster.
func Connect(connSpecStr string) (*Cluster, error) {
	spec, err := gocbconnstr.Parse(connSpecStr)
	if err != nil {
		return nil, err
	}

	if spec.Bucket != "" {
		return nil, errors.New("Connection string passed to Connect() must not have any bucket specified!")
	}

	fetchOption := func(name string) (string, bool) {
		optValue := spec.Options[name]
		if len(optValue) == 0 {
			return "", false
		}
		return optValue[len(optValue)-1], true
	}

	initialTracer := &opentracing.NoopTracer{}
	tracerAddRef(initialTracer)

	config := gocbcore.AgentConfig{
		UserString:           "gocb/" + Version(),
		ConnectTimeout:       60000 * time.Millisecond,
		ServerConnectTimeout: 7000 * time.Millisecond,
		NmvRetryDelay:        100 * time.Millisecond,
		UseKvErrorMaps:       true,
		UseDurations:         true,
		NoRootTraceSpans:     true,
		Tracer:               initialTracer,
	}
	err = config.FromConnStr(connSpecStr)
	if err != nil {
		return nil, err
	}

	httpCli := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: config.TlsConfig,
		},
	}

	cluster := &Cluster{
		agentConfig:      config,
		n1qlTimeout:      75 * time.Second,
		ftsTimeout:       75 * time.Second,
		analyticsTimeout: 75 * time.Second,

		httpCli:    httpCli,
		queryCache: make(map[string]*n1qlCache),
	}

	if valStr, ok := fetchOption("n1ql_timeout"); ok {
		val, err := strconv.ParseInt(valStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("n1ql_timeout option must be a number")
		}
		cluster.n1qlTimeout = time.Duration(val) * time.Millisecond
	}

	if valStr, ok := fetchOption("fts_timeout"); ok {
		val, err := strconv.ParseInt(valStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("fts_timeout option must be a number")
		}
		cluster.ftsTimeout = time.Duration(val) * time.Millisecond
	}

	return cluster, nil
}

// SetTracer allows you to specify a custom tracer to use for this cluster.
// EXPERIMENTAL
func (c *Cluster) SetTracer(tracer opentracing.Tracer) {
	if c.agentConfig.Tracer != nil {
		tracerDecRef(c.agentConfig.Tracer)
	}

	tracerAddRef(tracer)
	c.agentConfig.Tracer = tracer
}

// EnhancedErrors returns the current enhanced error message state.
func (c *Cluster) EnhancedErrors() bool {
	return c.agentConfig.UseEnhancedErrors
}

// SetEnhancedErrors sets the current enhanced error message state.
func (c *Cluster) SetEnhancedErrors(enabled bool) {
	c.agentConfig.UseEnhancedErrors = enabled
}

// ConnectTimeout returns the maximum time to wait when attempting to connect to a bucket.
func (c *Cluster) ConnectTimeout() time.Duration {
	return c.agentConfig.ConnectTimeout
}

// SetConnectTimeout sets the maximum time to wait when attempting to connect to a bucket.
func (c *Cluster) SetConnectTimeout(timeout time.Duration) {
	c.agentConfig.ConnectTimeout = timeout
}

// ServerConnectTimeout returns the maximum time to attempt to connect to a single node.
func (c *Cluster) ServerConnectTimeout() time.Duration {
	return c.agentConfig.ServerConnectTimeout
}

// SetServerConnectTimeout sets the maximum time to attempt to connect to a single node.
func (c *Cluster) SetServerConnectTimeout(timeout time.Duration) {
	c.agentConfig.ServerConnectTimeout = timeout
}

// N1qlTimeout returns the maximum time to wait for a cluster-level N1QL query to complete.
func (c *Cluster) N1qlTimeout() time.Duration {
	return c.n1qlTimeout
}

// SetN1qlTimeout sets the maximum time to wait for a cluster-level N1QL query to complete.
func (c *Cluster) SetN1qlTimeout(timeout time.Duration) {
	c.n1qlTimeout = timeout
}

// FtsTimeout returns the maximum time to wait for a cluster-level FTS query to complete.
func (c *Cluster) FtsTimeout() time.Duration {
	return c.ftsTimeout
}

// SetFtsTimeout sets the maximum time to wait for a cluster-level FTS query to complete.
func (c *Cluster) SetFtsTimeout(timeout time.Duration) {
	c.ftsTimeout = timeout
}

// AnalyticsTimeout returns the maximum time to wait for a cluster-level Analytics query to complete.
func (c *Cluster) AnalyticsTimeout() time.Duration {
	return c.analyticsTimeout
}

// SetAnalyticsTimeout sets the maximum time to wait for a cluster-level Analytics query to complete.
func (c *Cluster) SetAnalyticsTimeout(timeout time.Duration) {
	c.analyticsTimeout = timeout
}

// NmvRetryDelay returns the time to wait between retrying an operation due to not my vbucket.
func (c *Cluster) NmvRetryDelay() time.Duration {
	return c.agentConfig.NmvRetryDelay
}

// SetNmvRetryDelay sets the time to wait between retrying an operation due to not my vbucket.
func (c *Cluster) SetNmvRetryDelay(delay time.Duration) {
	c.agentConfig.NmvRetryDelay = delay
}

// InvalidateQueryCache forces the internal cache of prepared queries to be cleared.
func (c *Cluster) InvalidateQueryCache() {
	c.clusterLock.Lock()
	c.queryCache = make(map[string]*n1qlCache)
	c.clusterLock.Unlock()
}

// Close shuts down all buckets in this cluster and invalidates any references this cluster has.
func (c *Cluster) Close() error {
	var overallErr error

	// We have an upper bound on how many buckets we try
	// to close soely for deadlock prevention
	for i := 0; i < 1024; i++ {
		c.clusterLock.Lock()
		if len(c.bucketList) == 0 {
			c.clusterLock.Unlock()
			break
		}

		bucket := c.bucketList[0]
		c.clusterLock.Unlock()

		err := bucket.Close()
		if err != nil && gocbcore.ErrorCause(err) != gocbcore.ErrShutdown {
			logWarnf("Failed to close a bucket in cluster close: %s", err)
			overallErr = err
		}
	}

	if c.agentConfig.Tracer != nil {
		tracerDecRef(c.agentConfig.Tracer)
		c.agentConfig.Tracer = nil
	}

	return overallErr
}

func (c *Cluster) makeAgentConfig(bucket, password string, forceMt bool) (*gocbcore.AgentConfig, error) {
	auth := c.auth
	useCertificates := c.agentConfig.TlsConfig != nil && len(c.agentConfig.TlsConfig.Certificates) > 0
	if useCertificates {
		if auth == nil {
			return nil, ErrMixedCertAuthentication
		}
		_, ok := auth.(CertAuthenticator)
		if !ok {
			return nil, ErrMixedCertAuthentication
		}
	}

	if auth == nil {
		authMap := make(BucketAuthenticatorMap)
		authMap[bucket] = BucketAuthenticator{
			Password: password,
		}
		auth = ClusterAuthenticator{
			Buckets: authMap,
		}
	} else {
		if password != "" {
			return nil, ErrMixedAuthentication
		}
		_, ok := auth.(CertAuthenticator)
		if ok && !useCertificates {
			return nil, ErrMixedCertAuthentication
		}
	}

	config := c.agentConfig

	config.BucketName = bucket
	config.Password = password
	config.Auth = &coreAuthWrapper{
		auth:       auth,
		bucketName: bucket,
	}

	if forceMt {
		config.UseMutationTokens = true
	}

	return &config, nil
}

// Authenticate specifies an Authenticator interface to use to authenticate with cluster services.
func (c *Cluster) Authenticate(auth Authenticator) error {
	c.auth = auth
	return nil
}

func (c *Cluster) openBucket(bucket, password string, forceMt bool) (*Bucket, error) {
	agentConfig, err := c.makeAgentConfig(bucket, password, forceMt)
	if err != nil {
		return nil, err
	}

	b, err := createBucket(c, agentConfig)
	if err != nil {
		return nil, err
	}

	c.clusterLock.Lock()
	c.bucketList = append(c.bucketList, b)
	c.clusterLock.Unlock()

	return b, nil
}

// OpenBucket opens a new connection to the specified bucket.
func (c *Cluster) OpenBucket(bucket, password string) (*Bucket, error) {
	return c.openBucket(bucket, password, false)
}

// OpenBucketWithMt opens a new connection to the specified bucket and enables mutation tokens.
// MutationTokens allow you to execute queries and durability requirements with very specific
// operation-level consistency.
func (c *Cluster) OpenBucketWithMt(bucket, password string) (*Bucket, error) {
	return c.openBucket(bucket, password, true)
}

func (c *Cluster) closeBucket(bucket *Bucket) {
	c.clusterLock.Lock()
	for i, e := range c.bucketList {
		if e == bucket {
			c.bucketList = append(c.bucketList[0:i], c.bucketList[i+1:]...)
			break
		}
	}
	c.clusterLock.Unlock()
}

// Manager returns a ClusterManager object for performing cluster management operations on this cluster.
func (c *Cluster) Manager(username, password string) *ClusterManager {
	var mgmtHosts []string
	for _, host := range c.agentConfig.HttpAddrs {
		if c.agentConfig.TlsConfig != nil {
			mgmtHosts = append(mgmtHosts, "https://"+host)
		} else {
			mgmtHosts = append(mgmtHosts, "http://"+host)
		}
	}

	tlsConfig := c.agentConfig.TlsConfig
	return &ClusterManager{
		hosts:    mgmtHosts,
		username: username,
		password: password,
		httpCli: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
			},
		},
		cluster: c,
	}
}

// StreamingBucket represents a bucket connection used for streaming data over DCP.
type StreamingBucket struct {
	client *gocbcore.Agent
}

// IoRouter returns the underlying gocb agent managing connections.
func (b *StreamingBucket) IoRouter() *gocbcore.Agent {
	return b.client
}

// OpenStreamingBucket opens a new connection to the specified bucket for the purpose of streaming data.
func (c *Cluster) OpenStreamingBucket(streamName, bucket, password string) (*StreamingBucket, error) {
	agentConfig, err := c.makeAgentConfig(bucket, password, false)
	if err != nil {
		return nil, err
	}
	cli, err := gocbcore.CreateDcpAgent(agentConfig, streamName, 0)
	if err != nil {
		return nil, err
	}

	return &StreamingBucket{
		client: cli,
	}, nil
}

func (c *Cluster) randomBucket() (*Bucket, error) {
	c.clusterLock.RLock()
	if len(c.bucketList) == 0 {
		c.clusterLock.RUnlock()
		return nil, ErrNoOpenBuckets
	}
	bucket := c.bucketList[0]
	c.clusterLock.RUnlock()
	return bucket, nil
}
