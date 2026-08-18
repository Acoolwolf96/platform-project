package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

type provisionRequest struct {
	Name  string `json:"name"`
	Team  string `json:"team"`
	Image string `json:"image"`
	Size  string `json:"size"`
	Port  int    `json:"port"`
}

type sizeSpec struct {
	CPURequest    string
	CPULimit      string
	MemoryRequest string
	MemoryLimit   string
}

var serviceNamePattern =
	regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

var imagePattern =
	regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/:@-]*$`)

var serviceSizes = map[string]sizeSpec{
	"small": {
		CPURequest:    "25m",
		CPULimit:      "100m",
		MemoryRequest: "32Mi",
		MemoryLimit:   "64Mi",
	},
	"medium": {
		CPURequest:    "50m",
		CPULimit:      "200m",
		MemoryRequest: "64Mi",
		MemoryLimit:   "128Mi",
	},
}

func provisionHandler(kube *kubeClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req provisionRequest

		decoder := json.NewDecoder(io.LimitReader(r.Body, 4096))
		decoder.DisallowUnknownFields()

		if err := decoder.Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		req.Name = strings.TrimSpace(req.Name)
		req.Team = strings.TrimSpace(req.Team)
		req.Image = strings.TrimSpace(req.Image)
		req.Size = strings.TrimSpace(req.Size)

		if len(req.Name) == 0 ||
			len(req.Name) > 63 ||
			!serviceNamePattern.MatchString(req.Name) {
			http.Error(w, "invalid service name", http.StatusBadRequest)
			return
		}

		if !imagePattern.MatchString(req.Image) {
			http.Error(w, "invalid container image", http.StatusBadRequest)
			return
		}

		if req.Port < 1 || req.Port > 65535 {
			http.Error(w, "invalid container port", http.StatusBadRequest)
			return
		}

		size, ok := serviceSizes[req.Size]
		if !ok {
			http.Error(w, "invalid service size", http.StatusBadRequest)
			return
		}

		validTeam, err := teamExists(kube, req.Team)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if !validTeam {
			http.Error(w, "team does not exist", http.StatusBadRequest)
			return
		}

		exists, err := serviceExists(kube, req.Team, req.Name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if exists {
			http.Error(w, "service already exists", http.StatusConflict)
			return
		}

		path := fmt.Sprintf(
			"gitops/team-%s/%s.yaml",
			req.Team,
			req.Name,
		)

		manifest := buildServiceManifest(req, size)

		if err := createGitHubFile(
			path,
			manifest,
			fmt.Sprintf(
				"Provision %s for %s team",
				req.Name,
				req.Team,
			),
		); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)

		json.NewEncoder(w).Encode(map[string]string{
			"status":  "requested",
			"service": req.Name,
			"team":    req.Team,
			"path":    path,
		})
	}
}

func teamExists(kube *kubeClient, requested string) (bool, error) {
	items, err := teams(kube)
	if err != nil {
		return false, err
	}

	for _, team := range items {
		if team.Name == requested {
			return true, nil
		}
	}

	return false, nil
}

func serviceExists(
	kube *kubeClient,
	team string,
	name string,
) (bool, error) {
	items, err := services(kube)
	if err != nil {
		return false, err
	}

	for _, service := range items {
		if service.Team == team && service.Name == name {
			return true, nil
		}
	}

	return false, nil
}

func createGitHubFile(
	path string,
	content string,
	message string,
) error {
	token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	owner := strings.TrimSpace(os.Getenv("GITHUB_OWNER"))
	repo := strings.TrimSpace(os.Getenv("GITHUB_REPO"))
	branch := strings.TrimSpace(os.Getenv("GITHUB_BRANCH"))

	if token == "" {
		return fmt.Errorf("GitHub token is not configured")
	}

	if owner == "" || repo == "" {
		return fmt.Errorf("GitHub repository is not configured")
	}

	if branch == "" {
		branch = "main"
	}

	payload := map[string]string{
		"message": message,
		"content": base64.StdEncoding.EncodeToString([]byte(content)),
		"branch":  branch,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/contents/%s",
		owner,
		repo,
		path,
	)

	request, err := http.NewRequest(
		http.MethodPut,
		url,
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}

	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", "2026-03-10")
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		responseBody, _ := io.ReadAll(response.Body)

		return fmt.Errorf(
			"GitHub returned %s: %s",
			response.Status,
			string(responseBody),
		)
	}

	return nil
}

func buildServiceManifest(
	req provisionRequest,
	size sizeSpec,
) string {
	namespace := "team-" + req.Team

	return fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: %[1]s
  namespace: %[2]s
  labels:
    app: %[1]s
    platform.acoolwolf.dev/team: %[3]s
    platform.acoolwolf.dev/managed-by: idp
spec:
  replicas: 1
  selector:
    matchLabels:
      app: %[1]s
  template:
    metadata:
      labels:
        app: %[1]s
        platform.acoolwolf.dev/team: %[3]s
        platform.acoolwolf.dev/managed-by: idp
    spec:
      containers:
        - name: %[1]s
          image: %[4]s
          ports:
            - containerPort: %[5]d
          resources:
            requests:
              cpu: %[6]s
              memory: %[7]s
            limits:
              cpu: %[8]s
              memory: %[9]s
          readinessProbe:
            tcpSocket:
              port: %[5]d
            initialDelaySeconds: 2
            periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: %[1]s
  namespace: %[2]s
  labels:
    app: %[1]s
    platform.acoolwolf.dev/managed-by: idp
spec:
  selector:
    app: %[1]s
  ports:
    - port: %[5]d
      targetPort: %[5]d
`,
		req.Name,
		namespace,
		req.Team,
		req.Image,
		req.Port,
		size.CPURequest,
		size.MemoryRequest,
		size.CPULimit,
		size.MemoryLimit,
	)
}
