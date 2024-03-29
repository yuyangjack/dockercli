package v1beta1

import (
	"github.com/yuyangjack/dockercli/kubernetes/client/clientset/scheme"
	"github.com/yuyangjack/dockercli/kubernetes/compose/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
)

// ComposeV1beta1Interface defines the methods a compose v1beta1 client has
type ComposeV1beta1Interface interface {
	RESTClient() rest.Interface
	StacksGetter
}

// ComposeV1beta1Client is used to interact with features provided by the compose.docker.com group.
type ComposeV1beta1Client struct {
	restClient rest.Interface
}

// Stacks returns a stack client
func (c *ComposeV1beta1Client) Stacks(namespace string) StackInterface {
	return newStacks(c, namespace)
}

// NewForConfig creates a new ComposeV1beta1Client for the given config.
func NewForConfig(c *rest.Config) (*ComposeV1beta1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &ComposeV1beta1Client{client}, nil
}

// NewForConfigOrDie creates a new ComposeV1beta1Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *ComposeV1beta1Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new ComposeV1beta1Client for the given RESTClient.
func New(c rest.Interface) *ComposeV1beta1Client {
	return &ComposeV1beta1Client{c}
}

func setConfigDefaults(config *rest.Config) error {
	gv := v1beta1.SchemeGroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *ComposeV1beta1Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
