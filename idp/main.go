package main

import (
	"crypto/tls"
	"crypto/x509"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

//go:embed web/*
var webFiles embed.FS

type kubeClient struct {
	baseURL string
	token   string
	client  *http.Client
}

type metadata struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Labels    map[string]string `json:"labels"`
}

type namespaceList struct {
	Items []struct {
		Metadata metadata `json:"metadata"`
	} `json:"items"`
}

type deploymentList struct {
	Items []struct {
		Metadata metadata `json:"metadata"`
		Spec struct {
			Replicas int `json:"replicas"`
			Template struct {
				Spec struct {
					Containers []struct {
						Name  string `json:"name"`
						Image string `json:"image"`
						Resources struct {
							Requests map[string]string `json:"requests"`
							Limits   map[string]string `json:"limits"`
						} `json:"resources"`
					} `json:"containers"`
				} `json:"spec"`
			} `json:"template"`
		} `json:"spec"`
		Status struct {
			Replicas          int `json:"replicas"`
			AvailableReplicas int `json:"availableReplicas"`
			UpdatedReplicas   int `json:"updatedReplicas"`
		} `json:"status"`
	} `json:"items"`
}

type nodeList struct {
	Items []struct {
		Metadata metadata `json:"metadata"`
		Status struct {
			Conditions []struct {
				Type   string `json:"type"`
				Status string `json:"status"`
			} `json:"conditions"`
		} `json:"status"`
	} `json:"items"`
}

type serviceView struct {
	Name              string `json:"name"`
	Team              string `json:"team"`
	Namespace         string `json:"namespace"`
	Image             string `json:"image"`
	DesiredReplicas   int    `json:"desiredReplicas"`
	AvailableReplicas int    `json:"availableReplicas"`
	Healthy           bool   `json:"healthy"`
	CPURequest        string `json:"cpuRequest"`
	CPULimit          string `json:"cpuLimit"`
	MemoryRequest     string `json:"memoryRequest"`
	MemoryLimit       string `json:"memoryLimit"`
}

type teamView struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type overview struct {
	Teams           int `json:"teams"`
	Services        int `json:"services"`
	HealthyServices int `json:"healthyServices"`
	ReadyNodes      int `json:"readyNodes"`
	TotalNodes      int `json:"totalNodes"`
}

func newKubeClient() (*kubeClient, error) {
	token, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return nil, err
	}

	caData, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
	if err != nil {
		return nil, err
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caData) {
		return nil, fmt.Errorf("failed to load Kubernetes CA")
	}

	return &kubeClient{
		baseURL: "https://kubernetes.default.svc",
		token:   strings.TrimSpace(string(token)),
		client: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: pool,
				},
			},
		},
	}, nil
}

func (k *kubeClient) get(path string, target any) error {
	req, err := http.NewRequest(http.MethodGet, k.baseURL+path, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+k.token)

	resp, err := k.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("kubernetes API returned %s: %s", resp.Status, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

func services(k *kubeClient) ([]serviceView, error) {
	var deployments deploymentList

	if err := k.get("/apis/apps/v1/deployments", &deployments); err != nil {
		return nil, err
	}

	result := make([]serviceView, 0)

	for _, d := range deployments.Items {
		if !strings.HasPrefix(d.Metadata.Namespace, "team-") {
			continue
		}

		image := ""
		cpuRequest := ""
		cpuLimit := ""
		memoryRequest := ""
		memoryLimit := ""

		if len(d.Spec.Template.Spec.Containers) > 0 {
			container := d.Spec.Template.Spec.Containers[0]

			image = container.Image
			cpuRequest = container.Resources.Requests["cpu"]
			cpuLimit = container.Resources.Limits["cpu"]
			memoryRequest = container.Resources.Requests["memory"]
			memoryLimit = container.Resources.Limits["memory"]
		}

		result = append(result, serviceView{
			Name:              d.Metadata.Name,
			Team:              strings.TrimPrefix(d.Metadata.Namespace, "team-"),
			Namespace:         d.Metadata.Namespace,
			Image:             image,
			CPURequest:        cpuRequest,
			CPULimit:          cpuLimit,
			MemoryRequest:     memoryRequest,
			MemoryLimit:       memoryLimit,
			DesiredReplicas:   d.Spec.Replicas,
			AvailableReplicas: d.Status.AvailableReplicas,
			Healthy: d.Spec.Replicas > 0 &&
				d.Status.AvailableReplicas == d.Spec.Replicas,
		})
	}

	return result, nil
}

func teams(k *kubeClient) ([]teamView, error) {
	var namespaces namespaceList

	if err := k.get("/api/v1/namespaces", &namespaces); err != nil {
		return nil, err
	}

	result := make([]teamView, 0)

	for _, n := range namespaces.Items {
		if !strings.HasPrefix(n.Metadata.Name, "team-") {
			continue
		}

		result = append(result, teamView{
			Name:      strings.TrimPrefix(n.Metadata.Name, "team-"),
			Namespace: n.Metadata.Name,
		})
	}

	return result, nil
}

func nodes(k *kubeClient) (ready int, total int, err error) {
	var list nodeList

	if err := k.get("/api/v1/nodes", &list); err != nil {
		return 0, 0, err
	}

	total = len(list.Items)

	for _, node := range list.Items {
		for _, condition := range node.Status.Conditions {
			if condition.Type == "Ready" && condition.Status == "True" {
				ready++
				break
			}
		}
	}

	return ready, total, nil
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(value)
}

func main() {
	kube, err := newKubeClient()
	if err != nil {
		log.Fatalf("kubernetes client: %v", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/api/provision", provisionHandler(kube))

	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("/api/services", func(w http.ResponseWriter, r *http.Request) {
		items, err := services(kube)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, items)
	})

	mux.HandleFunc("/api/teams", func(w http.ResponseWriter, r *http.Request) {
		items, err := teams(kube)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, items)
	})

	mux.HandleFunc("/api/overview", func(w http.ResponseWriter, r *http.Request) {
		serviceList, err := services(kube)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		teamList, err := teams(kube)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		readyNodes, totalNodes, err := nodes(kube)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		healthy := 0
		for _, service := range serviceList {
			if service.Healthy {
				healthy++
			}
		}

		writeJSON(w, overview{
			Teams:           len(teamList),
			Services:        len(serviceList),
			HealthyServices: healthy,
			ReadyNodes:      readyNodes,
			TotalNodes:      totalNodes,
		})
	})

	staticFS, err := fs.Sub(webFiles, "web")
	if err != nil {
		log.Fatal(err)
	}

	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	server := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Println("IDP listening on :8080")

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
