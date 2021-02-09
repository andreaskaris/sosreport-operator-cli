package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	// "k8s.io/apimachinery/pkg/api/errors"
	cli "github.com/andreaskaris/sosreport-operator-cli/pkg/cli"
	supportv1alpha1 "github.com/andreaskaris/sosreport-operator/api/v1alpha1"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	//"github.com/go-yaml/yaml"
	"github.com/ghodss/yaml"
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

/*
We simplify here - if the role is "master", create a toleration for master roles
If the role name contains "master", do the same
That's overly simplistic and will not catch all use cases.
For the time being, it's o.k. though
*/
func (c *Client) buildTolerations(selectorString string, selectorType string) ([]corev1.Toleration, error) {
	var tolerations []corev1.Toleration

	if (selectorType == "role" && selectorString == "master") ||
		(selectorType == "hostname" && strings.Contains(selectorString, "master")) {
		tolerations = []corev1.Toleration{
			corev1.Toleration{
				Key:    "node-role.kubernetes.io/master",
				Effect: corev1.TaintEffectNoSchedule,
			},
		}
	}
	return tolerations, nil
}

func (c *Client) writeYaml(writeDirectory string, sosreportName string, object interface{}) error {
	yamlString, err := yaml.Marshal(object)
	if err != nil {
		return err
	}
	log.Debug(fmt.Sprintf(
		"Generating YAML file '%s' with content:\n---\n%s\n---",
		sosreportName,
		yamlString,
	),
	)
	yamlByteString := []byte(yamlString)
	if writeDirectory == "" {
		writeDirectory = "."
	}

	fi, err := os.Stat(writeDirectory)
	if err != nil {
		return err
	}
	if !fi.Mode().IsDir() {
		return fmt.Errorf("Path '%s' is not a directory", writeDirectory)
	}

	fullPath := writeDirectory + "/" + sosreportName + ".yaml"
	err = ioutil.WriteFile(fullPath, yamlByteString, 0644)
	if err != nil {
		return err
	}
	log.Info(fmt.Sprintf("Created file '%s'", fullPath))

	return nil
}

func (c *Client) CreateSosreport(commandLine *cli.Cli) error {
	sosreportName, _ := c.buildSosreportName()
	log.Debug(fmt.Sprintf("sosreportName: %s", sosreportName))

	sosreportNamespace, _ := c.buildSosreportNamespace()
	log.Debug(fmt.Sprintf("sosreportNamespace: %s", sosreportNamespace))

	var err error
	var nodeSelector map[string]string
	var tolerations []corev1.Toleration

	if commandLine.NodeName != "" {
		log.Debug(fmt.Sprintf("Building nodeSelector based on NodeName '%s'", commandLine.NodeName))
		nodeSelector, err = c.buildNodeSelector(commandLine.NodeName, "hostname")
		if err != nil {
			return err
		}
		tolerations, err = c.buildTolerations(commandLine.NodeName, "hostname")
		if err != nil {
			return err
		}
	} else if commandLine.Role != "" {
		log.Debug(fmt.Sprintf("Building nodeSelector based on Role '%s'", commandLine.Role))
		nodeSelector, err = c.buildNodeSelector(commandLine.Role, "role")
		if err != nil {
			return err
		}
		tolerations, err = c.buildTolerations(commandLine.Role, "role")
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

	if commandLine.DryRun {
		err = c.writeYaml(commandLine.YamlDir, sosreportName, sosreport)
		return err
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
