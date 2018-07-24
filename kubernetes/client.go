package kubernetes

import (
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// CreateClient function is used to create a client for k8s and returns interface
func CreateClient(kubeconfig string) kubernetes.Interface {
	restConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic("build config from flags failed" + err.Error())
	}
	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		panic("new client from config failed" + err.Error())
	}
	return client
}

// UpdateConfigMap function is used to update the conf file.
func UpdateConfigMap(k kubernetes.Interface, conf, ns string) error {

	var (
		cConf []byte
		err   error
	)

	cObj := v1.ConfigMap{}

	cList, err := k.CoreV1().ConfigMaps(ns).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, cm := range cList.Items {
		if cm.Name == "mesher-configmap" {
			cObj.Name = "mesher-configmap"
			cObj.Namespace = ns
			cObj.Kind = "ConfigMap"
			cObj.APIVersion = "v1"
			cObj.Data = make(map[string]string)

			fInfo, err := ioutil.ReadDir(conf)
			if err != nil {
				return err
			}

			for _, f := range fInfo {
				cConf, err = ioutil.ReadFile(conf + f.Name())
				if err != nil {
					return err
				}
				cObj.Data[f.Name()] = string(cConf)
			}
		}
	}

	configMap, err := k.CoreV1().ConfigMaps(ns).Update(&cObj)
	if err != nil {
		return err
	}

	log.Infof("After Update configuration is:", configMap)

	return nil
}
