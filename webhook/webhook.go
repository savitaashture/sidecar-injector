package webhook

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"path/filepath"
	"time"
	"crypto/tls"

	"gopkg.in/yaml.v2"
	"k8s.io/api/admission/v1beta1"
	admissionregistration "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/kubernetes/pkg/apis/core/v1"
	log "github.com/Sirupsen/logrus"
	"github.com/howeyc/fsnotify"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()

	// (https://github.com/kubernetes/kubernetes/issues/57982)
	defaulter = runtime.ObjectDefaulter(runtimeScheme)
)

var ignoredNamespaces = []string {
	metav1.NamespaceSystem,
	metav1.NamespacePublic,
}

const (
	webhookInjectKey = "sidecar-injector-mesher.io/inject"
	webhookStatusKey = "sidecar-injector-mesher.io/status"
)

type WebHookServer struct {
	SidecarConfig    *Config
	Server           *http.Server
	Watch *fsnotify.Watcher
	Lock sync.RWMutex
}

// Webhook Server parameters
type WebHookParameters struct {
	Port int
	CertFile string
	KeyFile string
	SidecarConfigFile string
	HealthCheckInterval time.Duration
	HealthCheckFile string
}

type Config struct {
	Containers  []corev1.Container  `yaml:"containers"`
	Volumes     []corev1.Volume     `yaml:"volumes"`
	ImagePullSecret []corev1.LocalObjectReference `yaml:"imagePullSecrets"`
}

type operation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func init() {
	_ = corev1.AddToScheme(runtimeScheme)
	_ = admissionregistration.AddToScheme(runtimeScheme)
	// https://github.com/kubernetes/kubernetes/issues/57982
	_ = v1.AddToScheme(runtimeScheme)
}

func NewWebhook(p WebHookParameters) (*WebHookServer, error){
	sidecarConfig, err := loadConfig(p.SidecarConfigFile)
	if err != nil {
		log.Errorf("Filed to load configuration: %v", err)
		return nil, err
	}

	crt, err := tls.LoadX509KeyPair(p.CertFile, p.KeyFile)
	if err != nil {
		log.Errorf("Filed to load key pair: %v", err)
		return nil, err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Errorf("failed to create a watcher object: %v", err)
		return nil, err
	}

	for _, file := range []string{p.SidecarConfigFile, p.CertFile, p.KeyFile}{
		watchFile, _ := filepath.Split(file)
		if err := watcher.Watch(watchFile); err != nil {
			log.Errorf("failed to watch the files: %v", err)
			return nil, fmt.Errorf("could not watch %v: %v", file, err)
		}
	}

	wh := &WebHookServer {
		SidecarConfig:    sidecarConfig,
		Server:           &http.Server {
			Addr:        fmt.Sprintf(":%v", p.Port),
			TLSConfig:   &tls.Config{Certificates: []tls.Certificate{crt}},
		},
		Watch: watcher,
	}

	// define http server and server handler
	h := http.NewServeMux()
	h.HandleFunc("/webhookmutation", wh.webhookMutation)
	wh.Server.Handler = h

	return wh, nil
}

// (https://github.com/kubernetes/kubernetes/issues/57982)
func applyDefaultsWorkaround(containers []corev1.Container, volumes []corev1.Volume, secrets []corev1.LocalObjectReference) {
	defaulter.Default(&corev1.Pod {
		Spec: corev1.PodSpec {
			Containers:     containers,
			Volumes:        volumes,
			ImagePullSecrets: secrets,
		},
	})
}

func loadConfig(cfgFile string) (*Config, error) {
	data, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	
	return &cfg, nil
}

func mutationRequired(ignoredList []string, metadata *metav1.ObjectMeta) bool {
	// ignore special kubernete system namespaces
	for _, namespace := range ignoredList {
		if metadata.Namespace == namespace {
			log.Infof("Skip mutation for %v for it' in special namespace:%v", metadata.Name, metadata.Namespace)
			return false
		}
	}

	annotations := metadata.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	status := annotations[webhookStatusKey]
	
	// determine whether to perform mutation based on annotation for the target resource
	var required bool
	if strings.ToLower(status) == "injected" {
		required = false
	} else {
		switch strings.ToLower(annotations[webhookInjectKey]) {
		default:
			required = false
		case "y", "yes", "true", "on":
			required = true
		}
	}
	
	log.Infof("Mutation policy for %v/%v: status: %q required:%v", metadata.Namespace, metadata.Name, status, required)
	return required
}

func addContainer(target, added []corev1.Container, basePath string) (patch []operation) {
	first := len(target) == 0
	var value interface{}
	for _, add := range added {
		value = add
		path := basePath
		if first {
			first = false
			value = []corev1.Container{add}
		} else {
			path = path + "/-"
		}
		patch = append(patch, operation {
			Op:    "add",
			Path:  path,
			Value: value,
		})
	}
	return patch
}

func addVolume(target, added []corev1.Volume, basePath string) (patch []operation) {
	first := len(target) == 0
	var value interface{}
	for _, add := range added {
		value = add
		path := basePath
		if first {
			first = false
			value = []corev1.Volume{add}
		} else {
			path = path + "/-"
		}
		patch = append(patch, operation {
			Op:    "add",
			Path:  path,
			Value: value,
		})
	}
	return patch
}

func addImagePullSecrets(target, added []corev1.LocalObjectReference, basePath string) (patch []operation) {
	first := len(target) == 0
	var value interface{}
	for _, add := range added {
		value = add
		path := basePath
		if first {
			first = false
			value = []corev1.LocalObjectReference{add}
		} else {
			path = path + "/-"
		}
		patch = append(patch, operation {
			Op:    "add",
			Path:  path,
			Value: value,
		})
	}
	return patch
}


func updateAnnotation(target map[string]string, added map[string]string) (patch []operation) {
	for key, value := range added {
		if target == nil || target[key] == "" {
			target = map[string]string{}
			patch = append(patch, operation {
				Op:   "add",
				Path: "/metadata/annotations",
				Value: map[string]string{
					key: value,
				},
			})
		} else {
			patch = append(patch, operation {
				Op:    "replace",
				Path:  "/metadata/annotations/" + key,
				Value: value,
			})
		}
	}
	return patch
}

// create mutation patch for resoures
func createPatch(pod *corev1.Pod, sidecarConfig *Config, annotations map[string]string) ([]byte, error) {
	var patch []operation
	
	patch = append(patch, addContainer(pod.Spec.Containers, sidecarConfig.Containers, "/spec/containers")...)
	patch = append(patch, addVolume(pod.Spec.Volumes, sidecarConfig.Volumes, "/spec/volumes")...)
	patch = append(patch, addImagePullSecrets(pod.Spec.ImagePullSecrets, sidecarConfig.ImagePullSecret, "/spec/imagePullSecrets")...)

	patch = append(patch, updateAnnotation(pod.Annotations, annotations)...)

	return json.Marshal(patch)
}

// main mutation process
func (wh *WebHookServer) mutation(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	req := ar.Request
	var pod corev1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		log.Errorf("Could not unmarshal raw object: %v", err)
		return &v1beta1.AdmissionResponse {
			Result: &metav1.Status {
				Message: err.Error(),
			},
		}
	}

	log.Infof("AdmissionReview for Kind=%v, Namespace=%v Name=%v (%v) UID=%v patchOperation=%v UserInfo=%v",
		req.Kind, req.Namespace, req.Name, pod.Name, req.UID, req.Operation, req.UserInfo)
	
	// determine whether to perform mutation
	if !mutationRequired(ignoredNamespaces, &pod.ObjectMeta) {
		log.Infof("Skipping mutation for %s/%s due to policy check", pod.Namespace, pod.Name)
		return &v1beta1.AdmissionResponse {
			Allowed: true, 
		}
	}
	
	// Workaround: https://github.com/kubernetes/kubernetes/issues/57982
	applyDefaultsWorkaround(wh.SidecarConfig.Containers, wh.SidecarConfig.Volumes, wh.SidecarConfig.ImagePullSecret)
	annotations := map[string]string{webhookStatusKey: "injected"}
	patchBytes, err := createPatch(&pod, wh.SidecarConfig, annotations)
	if err != nil {
		return &v1beta1.AdmissionResponse {
			Result: &metav1.Status {
				Message: err.Error(),
			},
		}
	}
	
	log.Infof("Response %v\n", string(patchBytes))
	return &v1beta1.AdmissionResponse {
		Allowed: true,
		Patch:   patchBytes,
		PatchType: func() *v1beta1.PatchType {
			pt := v1beta1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}

// Serve method for webhook server
func (wh *WebHookServer) webhookMutation(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}

	if len(body) == 0 {
		log.Errorf("empty request body")
		return
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		log.Errorf("Content-Type=%s, expect application/json", contentType)
		return
	}

	var aResponse *v1beta1.AdmissionResponse
	aRequest := v1beta1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, &aRequest); err != nil {
		log.Errorf("Can't decode body: %v", err)
		aResponse = &v1beta1.AdmissionResponse {
			Result: &metav1.Status {
				Message: err.Error(),
			},
		}
	} else {
		aResponse = wh.mutation(&aRequest)
	}

	admissionReview := v1beta1.AdmissionReview{}
	if aResponse != nil {
		admissionReview.Response = aResponse
		if aRequest.Request != nil {
			admissionReview.Response.UID = aRequest.Request.UID
		}
	}

	resp, err := json.Marshal(admissionReview)
	if err != nil {
		log.Errorf("Can't encode response: %v", err)
	}

	log.Infof("Ready to write reponse ...")
	if _, err := w.Write(resp); err != nil {
		log.Errorf("Can't write response: %v", err)
	}
}


func (wh *WebHookServer)Run(stop <-chan struct{}, p WebHookParameters){
	var healthChan <-chan time.Time

	if p.HealthCheckInterval != 0 && p.HealthCheckFile != "" {
		t := time.NewTicker(p.HealthCheckInterval)
		healthChan = t.C
		defer t.Stop()
	}

	go func() {
		if err := wh.Server.ListenAndServeTLS("", ""); err != nil {
			log.Errorf("Filed to listen and serve webhook server: %v", err)
		}
	}()

	defer wh.Server.Close()
	defer wh.Watch.Close()

	var timerChan <- chan time.Time

	for{
		select{
		case <-timerChan:
			sidecarConfig, err := loadConfig(p.SidecarConfigFile)
			if err != nil {
				log.Errorf("update error: %v", err)
				break
			}
			pair, err := tls.LoadX509KeyPair(p.CertFile, p.KeyFile)
			if err != nil {
				log.Errorf("reload cert error: %v", err)
				break
			}

			wh.Lock.Lock()
			wh.SidecarConfig = sidecarConfig
			wh.Server.TLSConfig = &tls.Config{Certificates: []tls.Certificate{pair}}
			wh.Lock.Unlock()
		case event := <- wh.Watch.Event:
			if event.IsModify() || event.IsCreate() {
				timerChan = time.After(100*time.Microsecond)
			}
		case err := <-wh.Watch.Error:
			log.Errorf("watcher error: %v", err)
		case <-healthChan:
			content := []byte(`ok`)
			if err := ioutil.WriteFile(p.HealthCheckFile, content, 0644); err != nil {
				log.Errorf("health check update of %q failed: %v", p.HealthCheckFile, err)
			}

		case <-stop:
			return
		}
	}
}