package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	// "k8s.io/apimachinery/pkg/api/errors"
	cli "github.com/andreaskaris/sosreport-operator-cli/pkg/cli"
	supportv1alpha1 "github.com/andreaskaris/sosreport-operator/api/v1alpha1"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	//
	// Uncomment to load all auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth"
	//
	// Or uncomment to load specific auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/openstack"
)

type Client struct {
	clientset     *kubernetes.Clientset
	sosreport     *supportv1alpha1.Sosreport
	namespace     string
	ctx           context.Context
	sosreportName string
}

/*
Constructor for a new Kubernetes Client
*/
func NewClient() (*Client, error) {
	c := new(Client)

	c.ctx = context.TODO()

	// use the current context in kubeconfig
	// config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	apiConfig, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	if err != nil {
		return nil, err
	}

	// https://stackoverflow.com/questions/55314152/how-to-get-namespace-from-current-context-set-in-kube-config
	namespace := apiConfig.Contexts[apiConfig.CurrentContext].Namespace
	if namespace == "" {
		namespace = "default"
	}
	c.namespace = namespace

	// https://github.com/kubernetes/client-go/issues/711
	clientConfig := clientcmd.NewDefaultClientConfig(*apiConfig, nil)
	config, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	// create the clientset
	c.clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Client) buildSosreportName() (string, error) {
	s := fmt.Sprintf("sosreport-cli-%v", time.Now().Unix())
	return s, nil
}

func (c *Client) buildSosreportNamespace() (string, error) {
	return c.namespace, nil
}

/*
Build NodeSelector
Valid selectorType: "role", "hostnamme"
*/
func (c *Client) buildNodeSelector(selectorString string, selectorType string) (map[string]string, error) {
	var nodeSelector map[string]string

	switch selectorType {
		case "role":
			nodeSelector = map[string]string{
				fmt.Sprintf("node-role.kubernetes.io/%s", selectorString): "",
			}
		case "hostname":
			nodeSelector = map[string]string{
				"kubernetes.io/hostname": selectorString,
			}
		default:
			return nil, fmt.Errorf("Invalid selector type: %s", selectorType)
	}

	return nodeSelector, nil
}

func (c *Client) buildTolerations(t string) ([]corev1.Toleration, error) {
	tolerations := []corev1.Toleration{
		corev1.Toleration{
			Key:    "node-role.kubernetes.io/master",
			Effect: corev1.TaintEffectNoSchedule,
		},
		corev1.Toleration{
			Key:    "node.kubernetes.io/not-ready",
			Effect: corev1.TaintEffectNoSchedule,
		},
	}
	return tolerations, nil
}

func (c *Client) CreateSosreport(commandLine *cli.Cli) error {
	sosreportName, _ := c.buildSosreportName()
	log.Debug(fmt.Sprintf("sosreportName: %s", sosreportName))

	sosreportNamespace, _ := c.buildSosreportNamespace()
	log.Debug(fmt.Sprintf("sosreportNamespace: %s", sosreportNamespace))

	var err error
	var nodeSelector map[string]string
	if commandLine.NodeName != "" {
		log.Debug(fmt.Sprintf("Building nodeSelector based on NodeName '%s'", commandLine.NodeName))
		nodeSelector, err = c.buildNodeSelector(commandLine.NodeName, "hostname")
		if err != nil {
			return err
		}
	} else if commandLine.Role != "" {
		log.Debug(fmt.Sprintf("Building nodeSelector based on Role '%s'", commandLine.Role))
		nodeSelector, err = c.buildNodeSelector(commandLine.Role, "role")
		if err != nil {
			return err
		}
	}
	if commandLine.NodeName != "" && commandLine.Role != "" {
		log.Warn(
			fmt.Sprintf(
				"Both NodeName ('%s') and Role ('%s') provided. Skipping Role.",
				commandLine.NodeName,
				commandLine.Role,
			),
		)
	}
	log.Debug(fmt.Sprintf("Using generated nodeSelector:\n%s", nodeSelector))

	tolerations, _ := c.buildTolerations("")
	log.Debug(fmt.Sprintf("tolerations: %s", tolerations))

	apiName := "support.openshift.io"
	apiVersion := "v1alpha1"
	apiString := fmt.Sprintf("%s/%s", apiName, apiVersion)
	kind := "Sosreport"
	kindLowerPlural := "sosreports"

	sosreport := &supportv1alpha1.Sosreport{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiString,
			Kind:       kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      sosreportName,
			Namespace: sosreportNamespace,
		},
		Spec: supportv1alpha1.SosreportSpec{
			NodeSelector: nodeSelector,
			Tolerations:  tolerations,
		},
	}

	// https://stackoverflow.com/questions/63408493/create-get-a-custom-kubernetes-resource
	// https://stackoverflow.com/questions/52029656/how-to-retrieve-kubernetes-metrics-via-client-go-and-golang?rq=1
	path := fmt.Sprintf("/apis/%s/%s/namespaces/%s/%s",
		apiName,
		apiVersion,
		c.namespace,
		kindLowerPlural,
	)
	log.Debug(fmt.Sprintf("Path: %s", path))

	body, err := json.Marshal(sosreport)
	if err != nil {
		return err
	}
	log.Debug(fmt.Sprintf("Body JSON of Sosreport to create:\n%s", body))

	generatedSosreport := supportv1alpha1.Sosreport{}
	err = c.clientset.RESTClient().
		Post().
		AbsPath(path).
		Body(body).
		Do(context.TODO()).
		Into(&generatedSosreport)

	if err != nil {
		return err
	}

	generatedSosreportJson, err := json.MarshalIndent(generatedSosreport, "", "    ")
	if err != nil {
		return err
	}
	log.Trace(fmt.Sprintf("Created Sosreport:\n%s", generatedSosreportJson))

	// write sosreportName for later retrieval
	c.sosreportName = sosreportName
	log.Info(fmt.Sprintf("Created Sosreport %s", c.sosreportName))

	return nil
}

/*
Print all pods in the cluster - just for testing
*/
func (c *Client) PrintPods() {
	pods, _ := c.clientset.CoreV1().Pods("").List(c.ctx, metav1.ListOptions{})
	fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))
}

