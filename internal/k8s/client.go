package k8s

import (
	"context"
	"fmt"
	"time"

	"github.com/pteich/crdlens/internal/config"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client encapsulates all Kubernetes clients
type Client struct {
	KubeClient          kubernetes.Interface
	DynamicClient       dynamic.Interface
	DiscoveryClient     discovery.DiscoveryInterface
	ApiextensionsClient clientset.Interface
	Config              *rest.Config
	Context             string
	Namespace           string
}

// NewClient initializes Kubernetes clients based on the provided configuration
func NewClient(cfg *config.Config) (*Client, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if cfg.Kubeconfig != "" {
		loadingRules.ExplicitPath = cfg.Kubeconfig
	}

	configOverrides := &clientcmd.ConfigOverrides{}
	if cfg.Context != "" {
		configOverrides.CurrentContext = cfg.Context
	}

	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	rawConfig, err := clientConfig.RawConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get raw config: %w", err)
	}

	currentContext := rawConfig.CurrentContext
	if cfg.Context != "" {
		currentContext = cfg.Context
	}

	namespace, _, err := clientConfig.Namespace()
	if err != nil {
		namespace = "default"
	}

	// Update config if namespace was not specified
	if cfg.Namespace == "" {
		cfg.Namespace = namespace
	}

	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery client: %w", err)
	}

	apiextensionsClient, err := clientset.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create apiextensions client: %w", err)
	}

	return &Client{
		KubeClient:          kubeClient,
		DynamicClient:       dynamicClient,
		DiscoveryClient:     discoveryClient,
		ApiextensionsClient: apiextensionsClient,
		Config:              restConfig,
		Context:             currentContext,
		Namespace:           cfg.Namespace,
	}, nil
}

// ListNamespaces returns a list of all namespaces in the cluster
func (c *Client) ListNamespaces(ctx context.Context) ([]string, error) {
	namespaces, err := c.KubeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	nsList := make([]string, len(namespaces.Items))
	for i, ns := range namespaces.Items {
		nsList[i] = ns.Name
	}
	return nsList, nil
}

// Discovery returns a new DiscoveryService
func (c *Client) Discovery() *DiscoveryService {
	return NewDiscoveryService(c.ApiextensionsClient)
}

// Dynamic returns a new DynamicService
func (c *Client) Dynamic() *DynamicService {
	return NewDynamicService(c.DynamicClient)
}

// Events returns a new EventService
func (c *Client) Events(namespace string) *EventService {
	return NewEventService(c.KubeClient.CoreV1().Events(namespace))
}

// NewDefaultCache returns a new Cache with a default TTL
func (c *Client) NewDefaultCache() *Cache {
	return NewCache(5 * time.Minute)
}
