package v1alpha1

import (
	"reflect"

	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextcs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

const (
	ClusterNSCRDPlural   string = "clusternamespaces"
	ClusterNSCRDGroup    string = "nrp-nautilus.io"
	ClusterNSCRDVersion  string = "v1alpha1"
	ClusterNSFullCRDName string = ClusterNSCRDPlural + "." + ClusterNSCRDGroup
)

// Create the CRD resource, ignore error if it already exists
func CreateNSCRD(clientset apiextcs.Interface) error {
	crd := &apiextv1beta1.CustomResourceDefinition{
		ObjectMeta: meta_v1.ObjectMeta{Name: ClusterNSFullCRDName},
		Spec: apiextv1beta1.CustomResourceDefinitionSpec{
			Group: ClusterNSCRDGroup,
			Versions: []apiextv1beta1.CustomResourceDefinitionVersion{
				{
					Name:    ClusterNSCRDVersion,
					Served:  true,
					Storage: true,
				},
			},
			Scope: apiextv1beta1.NamespaceScoped,
			Names: apiextv1beta1.CustomResourceDefinitionNames{
				Plural: ClusterNSCRDPlural,
				Kind:   reflect.TypeOf(ClusterNamespace{}).Name(),
			},
		},
	}

	_, err := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	if err != nil && apierrors.IsAlreadyExists(err) {
		return nil
	}
	return err

	// Note the original apiextensions example adds logic to wait for creation and exception handling
}

func NewClient(cfg *rest.Config) (*rest.RESTClient, *runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	SchemeBuilder := runtime.NewSchemeBuilder(addKnownTypes)
	if err := SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, nil, err
	}
	config := *cfg
	config.GroupVersion = &SchemeGroupVersion
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{
		CodecFactory: serializer.NewCodecFactory(scheme)}

	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, nil, err
	}
	return client, scheme, nil
}

func MakeClusterNSCrdClient(cl *rest.RESTClient, scheme *runtime.Scheme) *ClusterNSCrdClient {
	return &ClusterNSCrdClient{cl: cl, plural: ClusterNSCRDPlural,
		codec: runtime.NewParameterCodec(scheme)}
}

// +k8s:deepcopy-gen=false
type ClusterNSCrdClient struct {
	cl     *rest.RESTClient
	plural string
	codec  runtime.ParameterCodec
}

func (f *ClusterNSCrdClient) Create(obj *ClusterNamespace) (*ClusterNamespace, error) {
	var result ClusterNamespace
	err := f.cl.Post().
		Namespace(obj.Namespace).Resource(f.plural).
		Body(obj).Do().Into(&result)
	return &result, err
}

func (f *ClusterNSCrdClient) Update(obj *ClusterNamespace) (*ClusterNamespace, error) {
	var result ClusterNamespace
	err := f.cl.Put().
		Namespace(obj.Namespace).Resource(f.plural).Name(obj.Name).
		Body(obj).Do().Into(&result)
	return &result, err
}

func (f *ClusterNSCrdClient) Delete(name string, namespace string, options *meta_v1.DeleteOptions) error {
	return f.cl.Delete().
		Namespace(namespace).Resource(f.plural).
		Name(name).Body(options).Do().
		Error()
}

func (f *ClusterNSCrdClient) Get(name string, namespace string) (*ClusterNamespace, error) {
	var result ClusterNamespace
	err := f.cl.Get().
		Namespace(namespace).Resource(f.plural).
		Name(name).Do().Into(&result)
	return &result, err
}

func (f *ClusterNSCrdClient) List(namespace string, opts meta_v1.ListOptions) (*ClusterNamespaceList, error) {
	var result ClusterNamespaceList
	err := f.cl.Get().
		Namespace(namespace).Resource(f.plural).
		VersionedParams(&opts, f.codec).
		Do().Into(&result)
	return &result, err
}

// Create a new List watch for our TPR
func (f *ClusterNSCrdClient) NewListWatch(namespace string) *cache.ListWatch {
	return cache.NewListWatchFromClient(f.cl, f.plural, namespace, fields.Everything())
}
