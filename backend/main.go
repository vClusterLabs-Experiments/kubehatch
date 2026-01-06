package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

// VclusterConfig describes the structure of the vcluster.yaml file.
type VclusterConfig struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Spec struct {
		Replicas int `yaml:"replicas"`
		Service  struct {
			Type string `yaml:"type,omitempty"`
		} `yaml:"service,omitempty"`
	} `yaml:"spec"`
}

// ServiceJSON is used to parse Kubernetes service JSON output.
type ServiceJSON struct {
	Status struct {
		LoadBalancer struct {
			Ingress []struct {
				IP       string `json:"ip"`
				Hostname string `json:"hostname"`
			} `json:"ingress"`
		} `json:"loadBalancer"`
	} `json:"status"`
	Spec struct {
		Ports []struct {
			Port int `json:"port"`
		} `json:"ports"`
	} `json:"spec"`
}

// VclusterResponse is the JSON response that includes the generated kubeconfig.
type VclusterResponse struct {
	Kubeconfig string `json:"kubeconfig"`
}

// VclusterInfo represents information about a vcluster
type VclusterInfo struct {
	Name         string    `json:"name"`
	Namespace    string    `json:"namespace"`
	Status       string    `json:"status"`
	HA           bool      `json:"ha"`
	LoadBalancer bool      `json:"loadBalancer"`
	Endpoint     string    `json:"endpoint,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	Owner        string    `json:"owner,omitempty"` // User/team who created it
}

// NamespaceJSON is used to parse Kubernetes namespace JSON output
type NamespaceJSON struct {
	Metadata struct {
		Name              string    `json:"name"`
		CreationTimestamp time.Time `json:"creationTimestamp"`
	} `json:"metadata"`
}

// StatefulSetJSON is used to parse StatefulSet status
type StatefulSetJSON struct {
	Status struct {
		Replicas      int `json:"replicas"`
		ReadyReplicas int `json:"readyReplicas"`
	} `json:"status"`
	Spec struct {
		Replicas int `json:"replicas"`
	} `json:"spec"`
}

func main() {
	http.HandleFunc("/api/vcluster", corsMiddleware(vclusterHandler))
	http.HandleFunc("/api/vcluster/", corsMiddleware(vclusterDetailHandler))
	http.HandleFunc("/api/vclusters", corsMiddleware(vclustersListHandler))
	http.HandleFunc("/download", corsMiddleware(downloadHandler))
	log.Println("Backend API running on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func filterEnv(env []string, keys []string) []string {
	filtered := []string{}
	for _, e := range env {
		skip := false
		for _, key := range keys {
			if strings.HasPrefix(e, key+"=") {
				skip = true
				break
			}
		}
		if !skip {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// getUserFromRequest extracts username from Basic Auth or returns "default"
func getUserFromRequest(r *http.Request) string {
	username, _, ok := r.BasicAuth()
	if ok && username != "" {
		return username
	}
	// Fallback: try to get from header (for OAuth proxy or other auth)
	if user := r.Header.Get("X-Forwarded-User"); user != "" {
		return user
	}
	if user := r.Header.Get("X-Remote-User"); user != "" {
		return user
	}
	// Default user if no auth
	return "default"
}

func vclusterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "Error parsing multipart form: "+err.Error(), http.StatusBadRequest)
		return
	}
	clusterName := r.FormValue("clusterName")
	ha := r.FormValue("ha") == "on"
	useLoadBalancer := r.FormValue("loadbalancer") == "on"
	if clusterName == "" {
		http.Error(w, "clusterName is required", http.StatusBadRequest)
		return
	}

	// Get current user from authentication
	currentUser := getUserFromRequest(r)
	log.Printf("Request from user: %s, creating cluster: %s", currentUser, clusterName)

	reqID := strconv.FormatInt(time.Now().UnixNano(), 10)
	workingDir := filepath.Join(".", "requests", reqID)
	if err := os.MkdirAll(workingDir, 0755); err != nil {
		http.Error(w, "Error creating working directory: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var hostKubeconfig string
	file, _, err := r.FormFile("kubeconfigFile")
	if err == nil && file != nil {
		defer file.Close()
		uploadPath := filepath.Join(workingDir, "uploaded.yaml")
		outFile, err := os.Create(uploadPath)
		if err != nil {
			http.Error(w, "Error creating file: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer outFile.Close()
		if _, err = io.Copy(outFile, file); err != nil {
			http.Error(w, "Error saving uploaded file: "+err.Error(), http.StatusInternalServerError)
			return
		}
		hostKubeconfig, err = filepath.Abs(uploadPath)
		if err != nil {
			http.Error(w, "Error determining absolute path: "+err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		// Try default kubeconfig path first
		defaultPath := "/var/secrets/kubeconfig"
		if _, err := os.Stat(defaultPath); err == nil {
			hostKubeconfig = defaultPath
		} else {
			// If not found, use in-cluster config (empty string means use default kubectl behavior)
			hostKubeconfig = ""
			log.Printf("Request %s: No kubeconfig provided, using in-cluster config", reqID)
		}
	}

	log.Printf("Request %s: User=%s, clusterName=%s, HA=%v, LoadBalancer=%v, kubeconfig=%s", reqID, currentUser, clusterName, ha, useLoadBalancer, hostKubeconfig)

	if err := createVclusterYAML(workingDir, clusterName, ha, useLoadBalancer); err != nil {
		http.Error(w, fmt.Sprintf("Error creating YAML: %v", err), http.StatusInternalServerError)
		return
	}

	if err := createVirtualCluster(workingDir, clusterName, hostKubeconfig, useLoadBalancer); err != nil {
		http.Error(w, fmt.Sprintf("Error creating virtual cluster: %v", err), http.StatusInternalServerError)
		return
	}

	// Store user info with cluster (we'll use this for filtering)
	// For now, vcluster creates namespace as vcluster-<cluster-name>
	// We'll track ownership separately
	log.Printf("Request %s: Waiting for 1 minute for the cluster to be ready...", reqID)
	time.Sleep(1 * time.Minute)

	if err := fetchAndPatchKubeconfigFromSecret(workingDir, clusterName, hostKubeconfig, useLoadBalancer); err != nil {
		http.Error(w, fmt.Sprintf("Error fetching kubeconfig from secret: %v", err), http.StatusInternalServerError)
		return
	}

	kcPath := filepath.Join(workingDir, ".vcluster", clusterName, "kubeconfig.yaml")
	kcData, err := os.ReadFile(kcPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading kubeconfig: %v", err), http.StatusInternalServerError)
		return
	}
	// Set cluster owner annotation after cluster is created
	if err := setClusterOwner(hostKubeconfig, clusterName, currentUser); err != nil {
		log.Printf("Warning: failed to set cluster owner: %v", err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:  "reqid",
		Value: reqID,
		Path:  "/",
	})
	// Set cluster owner annotation after cluster is created
	if err := setClusterOwner(hostKubeconfig, clusterName, currentUser); err != nil {
		log.Printf("Warning: failed to set cluster owner: %v", err)
	}

	resp := VclusterResponse{
		Kubeconfig: string(kcData),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// setClusterOwner sets the owner annotation on the namespace
func setClusterOwner(hostKubeconfig, clusterName, owner string) error {
	namespace := "vcluster-" + clusterName
	args := []string{"annotate", "namespace", namespace, fmt.Sprintf("kubehatch.io/owner=%s", owner), "--overwrite"}
	if hostKubeconfig != "" {
		args = append([]string{"--kubeconfig", hostKubeconfig}, args...)
	}
	cmd := exec.Command("kubectl", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set owner annotation: %v, output: %s", err, string(out))
	}
	log.Printf("Set cluster %s owner to %s", clusterName, owner)
	return nil
}

func vclusterDetailHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/vcluster/")
	parts := strings.Split(path, "/")
	if len(parts) < 1 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	clusterName := parts[0]
	hostKubeconfig := getDefaultKubeconfig()

	if r.Method == http.MethodDelete {
		deleteVclusterHandler(w, r, clusterName, hostKubeconfig)
		return
	}

	if len(parts) == 2 && parts[1] == "kubeconfig" {
		getKubeconfigHandler(w, r, clusterName, hostKubeconfig)
		return
	}

	http.Error(w, "Not found", http.StatusNotFound)
}

func deleteVclusterHandler(w http.ResponseWriter, r *http.Request, clusterName, hostKubeconfig string) {
	log.Printf("Deleting vcluster: %s", clusterName)

	args := []string{
		"delete", clusterName,
		"--delete-namespace",
		"--yes",
	}
	cmd := exec.Command("vcluster", args...)
	env := filterEnv(os.Environ(), []string{"KUBERNETES_SERVICE_HOST", "KUBERNETES_SERVICE_PORT", "KUBERNETES_PORT"})
	if hostKubeconfig != "" {
		env = append(env, "KUBECONFIG="+hostKubeconfig)
	}
	cmd.Env = env

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error deleting vcluster: %v, output: %s", err, string(out))
		http.Error(w, fmt.Sprintf("Error deleting vcluster: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully deleted vcluster: %s", clusterName)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Cluster deleted successfully"})
}

func getKubeconfigHandler(w http.ResponseWriter, r *http.Request, clusterName, hostKubeconfig string) {
	// Always use secret method and update endpoint
	getKubeconfigFromSecret(w, r, clusterName, hostKubeconfig)
}

// Get kubeconfig from secret and update endpoint
func getKubeconfigFromSecret(w http.ResponseWriter, r *http.Request, clusterName, hostKubeconfig string) {
	namespace := "vcluster-" + clusterName

	// Use vcluster connect --print to get a working kubeconfig (includes port-forwarding setup)
	log.Printf("Getting kubeconfig for %s using vcluster connect", clusterName)
	args := []string{"connect", clusterName, "--namespace", namespace, "--print"}
	if hostKubeconfig != "" {
		args = append([]string{"--kubeconfig", hostKubeconfig}, args...)
	}
	cmd := exec.Command("vcluster", args...)
	env := filterEnv(os.Environ(), []string{"KUBERNETES_SERVICE_HOST", "KUBERNETES_SERVICE_PORT", "KUBERNETES_PORT"})
	if hostKubeconfig != "" {
		env = append(env, "KUBECONFIG="+hostKubeconfig)
	}
	cmd.Env = env

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error getting kubeconfig via vcluster connect for %s: %v, output: %s", clusterName, err, string(out))
		// Fallback to secret method
		log.Printf("Falling back to secret method for %s", clusterName)
		getKubeconfigFromSecretFallback(w, r, clusterName, hostKubeconfig)
		return
	}

	kcData := out

	// Check if LoadBalancer is enabled and update endpoint
	useLoadBalancer := checkLoadBalancerEnabled(hostKubeconfig, clusterName)
	if useLoadBalancer {
		endpoint, err := getExternalEndpoint(hostKubeconfig, clusterName)
		if err == nil && endpoint != "" {
			kcData, err = updateKubeconfigEndpoint(kcData, endpoint)
			if err != nil {
				log.Printf("Warning: failed to update kubeconfig endpoint: %v", err)
			}
		}
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=kubeconfig-%s.yaml", clusterName))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(kcData)
}

// Fallback method using secret
func getKubeconfigFromSecretFallback(w http.ResponseWriter, r *http.Request, clusterName, hostKubeconfig string) {
	namespace := "vcluster-" + clusterName
	secretName := "vc-" + clusterName

	args := []string{"get", "secret", secretName, "-n", namespace, "--template={{.data.config}}"}
	if hostKubeconfig != "" {
		args = append([]string{"--kubeconfig", hostKubeconfig}, args...)
	}
	cmd := exec.Command("kubectl", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error getting kubeconfig for %s: %v, output: %s", clusterName, err, string(out))
		http.Error(w, fmt.Sprintf("Error getting kubeconfig: %v", err), http.StatusNotFound)
		return
	}

	base64Data := strings.TrimSpace(string(out))
	if base64Data == "" {
		http.Error(w, "Kubeconfig secret is empty", http.StatusNotFound)
		return
	}

	decoded, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		log.Printf("Error decoding kubeconfig for %s: %v", clusterName, err)
		http.Error(w, fmt.Sprintf("Error decoding kubeconfig: %v", err), http.StatusInternalServerError)
		return
	}

	// For kind clusters, add note that port-forwarding is needed
	// The kubeconfig will have localhost:8443 which requires port-forwarding
	log.Printf("Note: For kind clusters, user needs to run 'vcluster connect %s -n %s' for port-forwarding", clusterName, namespace)

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=kubeconfig-%s.yaml", clusterName))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(decoded)
}

// Get ClusterIP service endpoint for vcluster
func getClusterIPEndpoint(hostKubeconfig, clusterName string) (string, error) {
	namespace := "vcluster-" + clusterName
	svcName := clusterName

	args := []string{"get", "svc", svcName, "-n", namespace, "-o", "json"}
	if hostKubeconfig != "" {
		args = append([]string{"--kubeconfig", hostKubeconfig}, args...)
	}
	cmd := exec.Command("kubectl", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get service: %v", err)
	}

	var svc ServiceJSON
	if err := json.Unmarshal(out, &svc); err != nil {
		return "", fmt.Errorf("failed to parse service: %v", err)
	}

	// Get ClusterIP
	clusterIPArgs := []string{"get", "svc", svcName, "-n", namespace, "-o", "jsonpath={.spec.clusterIP}"}
	if hostKubeconfig != "" {
		clusterIPArgs = append([]string{"--kubeconfig", hostKubeconfig}, clusterIPArgs...)
	}
	clusterIPCmd := exec.Command("kubectl", clusterIPArgs...)
	clusterIPOut, err := clusterIPCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get ClusterIP: %v", err)
	}

	clusterIP := strings.TrimSpace(string(clusterIPOut))
	if clusterIP == "" {
		return "", fmt.Errorf("ClusterIP is empty")
	}

	if len(svc.Spec.Ports) == 0 {
		return "", fmt.Errorf("no ports configured")
	}

	port := svc.Spec.Ports[0].Port
	var endpoint string
	if port == 443 {
		endpoint = "https://" + clusterIP
	} else {
		endpoint = "https://" + clusterIP + ":" + strconv.Itoa(port)
	}

	return endpoint, nil
}

func vclustersListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get current user for filtering
	currentUser := getUserFromRequest(r)
	log.Printf("Listing clusters for user: %s", currentUser)

	hostKubeconfig := getDefaultKubeconfig()
	if hostKubeconfig == "" {
		// Return empty list if no kubeconfig available
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]VclusterInfo{})
		return
	}

	clusters, err := listVclusters(hostKubeconfig, currentUser)
	if err != nil {
		log.Printf("Error listing vclusters: %v", err)
		log.Printf("Using kubeconfig: %s", hostKubeconfig)
		// Return empty list on error rather than failing
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]VclusterInfo{})
		return
	}
	log.Printf("Found %d vclusters for user %s", len(clusters), currentUser)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(clusters)
}

func listVclusters(hostKubeconfig, currentUser string) ([]VclusterInfo, error) {
	// List all namespaces that start with "vcluster-"
	args := []string{"get", "namespaces", "-o", "json"}
	if hostKubeconfig != "" {
		args = append([]string{"--kubeconfig", hostKubeconfig}, args...)
	}
	cmd := exec.Command("kubectl", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %v", err)
	}

	var namespaceList struct {
		Items []NamespaceJSON `json:"items"`
	}
	if err := json.Unmarshal(out, &namespaceList); err != nil {
		return nil, fmt.Errorf("failed to parse namespace list: %v", err)
	}

	var clusters []VclusterInfo
	log.Printf("Processing %d namespaces", len(namespaceList.Items))
	for _, ns := range namespaceList.Items {
		if !strings.HasPrefix(ns.Metadata.Name, "vcluster-") {
			continue
		}

		clusterName := strings.TrimPrefix(ns.Metadata.Name, "vcluster-")
		log.Printf("Processing cluster: %s (namespace: %s)", clusterName, ns.Metadata.Name)
		info, err := getVclusterInfo(hostKubeconfig, clusterName, ns.Metadata.CreationTimestamp, currentUser)
		if err != nil {
			log.Printf("Error getting info for cluster %s: %v", clusterName, err)
			// Still add basic info even if detailed info fails
			info = VclusterInfo{
				Name:      clusterName,
				Namespace: ns.Metadata.Name,
				CreatedAt: ns.Metadata.CreationTimestamp,
				Status:    "Unknown",
				Owner:     getClusterOwner(hostKubeconfig, clusterName), // Try to get owner
			}
		}

		// Filter: Only show clusters owned by current user (or all if user is "default" or "admin")
		if currentUser == "default" || currentUser == "admin" || info.Owner == "" || info.Owner == currentUser {
			clusters = append(clusters, info)
			log.Printf("Added cluster %s to list (status: %s, owner: %s)", clusterName, info.Status, info.Owner)
		} else {
			log.Printf("Skipping cluster %s (owner: %s, current user: %s)", clusterName, info.Owner, currentUser)
		}
	}

	log.Printf("Returning %d clusters", len(clusters))
	return clusters, nil
}

// getClusterOwner tries to determine cluster owner from namespace or annotations
func getClusterOwner(hostKubeconfig, clusterName string) string {
	namespace := "vcluster-" + clusterName
	// Try to get owner from namespace annotation
	args := []string{"get", "namespace", namespace, "-o", "jsonpath={.metadata.annotations.kubehatch\\.io/owner}"}
	if hostKubeconfig != "" {
		args = append([]string{"--kubeconfig", hostKubeconfig}, args...)
	}
	cmd := exec.Command("kubectl", args...)
	out, err := cmd.CombinedOutput()
	if err == nil && strings.TrimSpace(string(out)) != "" {
		return strings.TrimSpace(string(out))
	}
	// If namespace starts with vcluster-<user>-, extract user
	if strings.HasPrefix(namespace, "vcluster-") {
		parts := strings.Split(strings.TrimPrefix(namespace, "vcluster-"), "-")
		if len(parts) > 1 {
			// Check if it's user-prefixed format: vcluster-<user>-<cluster>
			// For now, assume standard format unless we detect user prefix
			return ""
		}
	}
	return ""
}

func getVclusterInfo(hostKubeconfig, clusterName string, createdAt time.Time, currentUser string) (VclusterInfo, error) {
	namespace := "vcluster-" + clusterName
	info := VclusterInfo{
		Name:      clusterName,
		Namespace: namespace,
		CreatedAt: createdAt,
		Status:    "Unknown",
		Owner:     getClusterOwner(hostKubeconfig, clusterName),
	}

	// Check if StatefulSet exists to determine HA
	args := []string{"get", "statefulset", clusterName, "-n", namespace, "-o", "json"}
	if hostKubeconfig != "" {
		args = append([]string{"--kubeconfig", hostKubeconfig}, args...)
	}
	cmd := exec.Command("kubectl", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// StatefulSet might not exist yet, try to get status from namespace
		info.Status = "Pending"
		return info, nil
	}

	var sts StatefulSetJSON
	if err := json.Unmarshal(out, &sts); err == nil {
		info.HA = sts.Spec.Replicas > 1
		if sts.Status.ReadyReplicas == sts.Spec.Replicas && sts.Spec.Replicas > 0 {
			info.Status = "Running"
		} else if sts.Status.ReadyReplicas > 0 {
			info.Status = "Pending"
		} else {
			info.Status = "Pending"
		}
	}

	// Check for LoadBalancer service
	svcArgs := []string{"get", "svc", clusterName, "-n", namespace, "-o", "json"}
	if hostKubeconfig != "" {
		svcArgs = append([]string{"--kubeconfig", hostKubeconfig}, svcArgs...)
	}
	svcCmd := exec.Command("kubectl", svcArgs...)
	svcOut, err := svcCmd.CombinedOutput()
	if err == nil {
		var svc ServiceJSON
		if err := json.Unmarshal(svcOut, &svc); err == nil {
			if svc.Spec.Ports != nil && len(svc.Spec.Ports) > 0 {
				// Check if it's a LoadBalancer type by checking the service type
				typeArgs := []string{"get", "svc", clusterName, "-n", namespace, "-o", "jsonpath={.spec.type}"}
				if hostKubeconfig != "" {
					typeArgs = append([]string{"--kubeconfig", hostKubeconfig}, typeArgs...)
				}
				typeCmd := exec.Command("kubectl", typeArgs...)
				typeOut, _ := typeCmd.CombinedOutput()
				if strings.TrimSpace(string(typeOut)) == "LoadBalancer" {
					info.LoadBalancer = true
					endpoint, err := getExternalEndpoint(hostKubeconfig, clusterName)
					if err == nil {
						info.Endpoint = endpoint
					}
				}
			}
		}
	}

	return info, nil
}

func checkLoadBalancerEnabled(hostKubeconfig, clusterName string) bool {
	namespace := "vcluster-" + clusterName
	args := []string{"get", "svc", clusterName, "-n", namespace, "-o", "jsonpath={.spec.type}"}
	if hostKubeconfig != "" {
		args = append([]string{"--kubeconfig", hostKubeconfig}, args...)
	}
	cmd := exec.Command("kubectl", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "LoadBalancer"
}

func getExternalEndpoint(hostKubeconfig, clusterName string) (string, error) {
	namespace := "vcluster-" + clusterName
	svcName := clusterName

	args := []string{"get", "svc", svcName, "-n", namespace, "-o", "json"}
	if hostKubeconfig != "" {
		args = append([]string{"--kubeconfig", hostKubeconfig}, args...)
	}
	cmd := exec.Command("kubectl", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get service: %v", err)
	}

	var svc ServiceJSON
	if err := json.Unmarshal(out, &svc); err != nil {
		return "", fmt.Errorf("failed to parse service: %v", err)
	}

	if len(svc.Status.LoadBalancer.Ingress) == 0 {
		return "", fmt.Errorf("no external endpoint available")
	}

	ing := svc.Status.LoadBalancer.Ingress[0]
	var external string
	if ing.IP != "" {
		external = ing.IP
	} else if ing.Hostname != "" {
		external = ing.Hostname
	} else {
		return "", fmt.Errorf("no valid external endpoint")
	}

	if len(svc.Spec.Ports) == 0 {
		return "", fmt.Errorf("no ports configured")
	}

	port := svc.Spec.Ports[0].Port
	var endpoint string
	if port == 443 {
		endpoint = "https://" + external
	} else {
		endpoint = "https://" + external + ":" + strconv.Itoa(port)
	}

	return endpoint, nil
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("reqid")
	if err != nil {
		http.Error(w, "Request ID not set", http.StatusBadRequest)
		return
	}
	clusterName := r.URL.Query().Get("clusterName")
	if clusterName == "" {
		http.Error(w, "clusterName query parameter required", http.StatusBadRequest)
		return
	}
	kcPath := filepath.Join(".", "requests", cookie.Value, ".vcluster", clusterName, "kubeconfig.yaml")
	w.Header().Set("Content-Disposition", "attachment; filename=kubeconfig.yaml")
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, kcPath)
}

func getDefaultKubeconfig() string {
	// First try the mounted secret path (for Kubernetes deployment)
	defaultPath := "/var/secrets/kubeconfig"
	if _, err := os.Stat(defaultPath); err == nil {
		return defaultPath
	}
	// For local development, try ~/.kube/config
	homeDir, err := os.UserHomeDir()
	if err == nil {
		localKubeconfig := filepath.Join(homeDir, ".kube", "config")
		if _, err := os.Stat(localKubeconfig); err == nil {
			return localKubeconfig
		}
	}
	// If neither exists, use empty string (kubectl will use in-cluster config or default)
	return ""
}

func createVclusterYAML(workingDir, clusterName string, ha, useLoadBalancer bool) error {
	cfg := VclusterConfig{
		APIVersion: "v1",
		Kind:       "VirtualCluster",
	}
	cfg.Metadata.Name = clusterName
	if ha {
		cfg.Spec.Replicas = 3
	} else {
		cfg.Spec.Replicas = 1
	}
	if useLoadBalancer {
		cfg.Spec.Service.Type = "LoadBalancer"
	}
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("error marshalling YAML: %v", err)
	}
	yamlPath := filepath.Join(workingDir, "vcluster.yaml")
	if err := os.WriteFile(yamlPath, data, 0644); err != nil {
		return fmt.Errorf("error writing vcluster.yaml: %v", err)
	}
	log.Println("Generated vcluster.yaml:")
	log.Println(string(data))
	return nil
}

func getFreePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func createVirtualCluster(workingDir, clusterName, hostKubeconfig string, useLoadBalancer bool) error {
	args := []string{
		"create", clusterName,
		"--config", "vcluster.yaml",
		"--connect=false",
		"--debug",
	}
	if useLoadBalancer {
		args = append(args, "--expose")
	}
	cmd := exec.Command("vcluster", args...)
	cmd.Dir = workingDir
	env := filterEnv(os.Environ(), []string{"KUBERNETES_SERVICE_HOST", "KUBERNETES_SERVICE_PORT", "KUBERNETES_PORT"})
	if hostKubeconfig != "" {
		env = append(env, "KUBECONFIG="+hostKubeconfig)
	}
	cmd.Env = env
	log.Printf("DEBUG: executing vcluster command: vcluster %s (in %s)", strings.Join(args, " "), workingDir)
	log.Printf("DEBUG: Full command args: %v", cmd.Args)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("DEBUG: vcluster create command output:\n%s", string(out))
		return fmt.Errorf("vcluster create failed: %v\nOutput:\n%s", err, string(out))
	}
	log.Println("DEBUG: vcluster create command finished, output:")
	log.Println(string(out))
	return nil
}

func fetchAndPatchKubeconfigFromSecret(workingDir, clusterName, hostKubeconfig string, useLoadBalancer bool) error {
	namespace := "vcluster-" + clusterName

	// For kind clusters, use vcluster connect --print to get a working kubeconfig
	// This includes the proper port-forwarding setup
	log.Printf("DEBUG: Getting kubeconfig using vcluster connect for %s", clusterName)

	retryTimeout := time.After(3 * time.Minute)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	var kcData []byte
	for {
		args := []string{"connect", clusterName, "--namespace", namespace, "--print"}
		if hostKubeconfig != "" {
			args = append([]string{"--kubeconfig", hostKubeconfig}, args...)
		}
		cmd := exec.Command("vcluster", args...)
		env := filterEnv(os.Environ(), []string{"KUBERNETES_SERVICE_HOST", "KUBERNETES_SERVICE_PORT", "KUBERNETES_PORT"})
		if hostKubeconfig != "" {
			env = append(env, "KUBECONFIG="+hostKubeconfig)
		}
		cmd.Env = env

		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("DEBUG: vcluster connect failed (cluster may not be ready): %v", err)
		} else {
			kcData = out
			log.Printf("DEBUG: successfully retrieved kubeconfig using vcluster connect")
			break
		}

		select {
		case <-retryTimeout:
			// Fallback to secret method
			log.Printf("DEBUG: vcluster connect timed out, falling back to secret method")
			return fetchKubeconfigFromSecretFallback(workingDir, clusterName, hostKubeconfig, useLoadBalancer)
		case <-ticker.C:
			log.Println("DEBUG: vcluster not ready yet, retrying connect...")
		}
	}

	// If LoadBalancer is enabled, try to update endpoint
	if useLoadBalancer {
		log.Println("DEBUG: polling for external endpoint of virtual cluster...")
		externalEndpoint, err := pollForExternalEndpoint(hostKubeconfig, clusterName)
		if err == nil && externalEndpoint != "" {
			kcData, err = updateKubeconfigEndpoint(kcData, externalEndpoint)
			if err != nil {
				log.Printf("Warning: failed to update kubeconfig endpoint: %v", err)
			}
		}
	}

	newDir := filepath.Join(workingDir, ".vcluster", clusterName)
	newPath := filepath.Join(newDir, "kubeconfig.yaml")
	if err := os.MkdirAll(newDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(newPath, kcData, 0644); err != nil {
		return fmt.Errorf("failed to write kubeconfig file: %v", err)
	}
	log.Println("DEBUG: kubeconfig written to", newPath)
	return nil
}

// Fallback method to get kubeconfig from secret
func fetchKubeconfigFromSecretFallback(workingDir, clusterName, hostKubeconfig string, useLoadBalancer bool) error {
	namespace := "vcluster-" + clusterName
	secretName := "vc-" + clusterName
	var kcData []byte

	retryTimeout := time.After(2 * time.Minute)
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		args := []string{"get", "secret", secretName, "-n", namespace, "--template={{.data.config}}"}
		if hostKubeconfig != "" {
			args = append([]string{"--kubeconfig", hostKubeconfig}, args...)
		}
		cmd := exec.Command("kubectl", args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("DEBUG: failed to get secret %s in namespace %s: %v, output: %s", secretName, namespace, err, string(out))
		} else {
			base64Data := strings.TrimSpace(string(out))
			if base64Data == "" {
				log.Printf("DEBUG: secret %s exists but data is empty, retrying...", secretName)
			} else {
				decoded, err := base64.StdEncoding.DecodeString(base64Data)
				if err != nil {
					log.Printf("DEBUG: failed to decode base64 kubeconfig: %v", err)
				} else if len(decoded) > 0 {
					kcData = decoded
					log.Printf("DEBUG: successfully retrieved kubeconfig from secret %s", secretName)
					break
				}
			}
		}
		select {
		case <-retryTimeout:
			return fmt.Errorf("timed out waiting for vcluster secret %s in namespace %s", secretName, namespace)
		case <-ticker.C:
			log.Println("DEBUG: secret not ready yet, retrying...")
		}
	}

	if useLoadBalancer {
		log.Println("DEBUG: polling for external endpoint of virtual cluster...")
		externalEndpoint, err := pollForExternalEndpoint(hostKubeconfig, clusterName)
		if err == nil && externalEndpoint != "" {
			kcData, err = updateKubeconfigEndpoint(kcData, externalEndpoint)
			if err != nil {
				log.Printf("Warning: failed to update kubeconfig endpoint: %v", err)
			}
		}
	}

	newDir := filepath.Join(workingDir, ".vcluster", clusterName)
	newPath := filepath.Join(newDir, "kubeconfig.yaml")
	if err := os.MkdirAll(newDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(newPath, kcData, 0644); err != nil {
		return fmt.Errorf("failed to write kubeconfig file: %v", err)
	}
	log.Println("DEBUG: kubeconfig written to", newPath)
	return nil
}

// Old function name kept for compatibility
func fetchKubeconfigFromSecret(workingDir, clusterName, hostKubeconfig string, useLoadBalancer bool) error {
	return fetchKubeconfigFromSecretFallback(workingDir, clusterName, hostKubeconfig, useLoadBalancer)
	namespace := "vcluster-" + clusterName
	secretName := "vc-" + clusterName
	var kcData []byte

	retryTimeout := time.After(2 * time.Minute)
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		args := []string{"get", "secret", secretName, "-n", namespace, "--template={{.data.config}}"}
		if hostKubeconfig != "" {
			args = append([]string{"--kubeconfig", hostKubeconfig}, args...)
		}
		cmd := exec.Command("kubectl", args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("DEBUG: failed to get secret %s in namespace %s: %v, output: %s", secretName, namespace, err, string(out))
		} else {
			base64Data := strings.TrimSpace(string(out))
			if base64Data == "" {
				log.Printf("DEBUG: secret %s exists but data is empty, retrying...", secretName)
			} else {
				decoded, err := base64.StdEncoding.DecodeString(base64Data)
				if err != nil {
					log.Printf("DEBUG: failed to decode base64 kubeconfig: %v", err)
				} else if len(decoded) > 0 {
					kcData = decoded
					log.Printf("DEBUG: successfully retrieved kubeconfig from secret %s", secretName)
					break
				}
			}
		}
		select {
		case <-retryTimeout:
			return fmt.Errorf("timed out waiting for vcluster secret %s in namespace %s", secretName, namespace)
		case <-ticker.C:
			log.Println("DEBUG: secret not ready yet, retrying...")
		}
	}

	if useLoadBalancer {
		log.Println("DEBUG: polling for external endpoint of virtual cluster...")
		externalEndpoint, err := pollForExternalEndpoint(hostKubeconfig, clusterName)
		if err == nil && externalEndpoint != "" {
			kcData, err = updateKubeconfigEndpoint(kcData, externalEndpoint)
			if err != nil {
				log.Printf("Warning: failed to update kubeconfig endpoint: %v", err)
			}
		}
	}

	newDir := filepath.Join(workingDir, ".vcluster", clusterName)
	newPath := filepath.Join(newDir, "kubeconfig.yaml")
	if err := os.MkdirAll(newDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(newPath, kcData, 0644); err != nil {
		return fmt.Errorf("failed to write kubeconfig file: %v", err)
	}
	log.Println("DEBUG: kubeconfig written to", newPath)
	return nil
}

func pollForExternalEndpoint(hostKubeconfig, clusterName string) (string, error) {
	ns := "vcluster-" + clusterName
	svcName := clusterName
	timeout := time.After(3 * time.Minute)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-timeout:
			return "", fmt.Errorf("timed out waiting for external endpoint")
		case <-ticker.C:
			cmd := exec.Command("kubectl", "--kubeconfig", hostKubeconfig, "get", "svc", svcName, "-n", ns, "-o", "json")
			out, err := cmd.CombinedOutput()
			if err != nil {
				log.Println("DEBUG: kubectl get svc error:", err, "output:", string(out))
				continue
			}
			var svc ServiceJSON
			if err := json.Unmarshal(out, &svc); err != nil {
				log.Println("DEBUG: error unmarshaling service JSON:", err)
				continue
			}
			if len(svc.Status.LoadBalancer.Ingress) > 0 {
				ing := svc.Status.LoadBalancer.Ingress[0]
				var external string
				if ing.IP != "" {
					external = ing.IP
				} else if ing.Hostname != "" {
					external = ing.Hostname
				} else {
					continue
				}
				if len(svc.Spec.Ports) == 0 {
					continue
				}
				port := svc.Spec.Ports[0].Port
				var endpoint string
				if port == 443 {
					endpoint = "https://" + external
				} else {
					endpoint = "https://" + external + ":" + strconv.Itoa(port)
				}
				log.Println("DEBUG: found external endpoint:", endpoint)
				return endpoint, nil
			}
			log.Println("DEBUG: external endpoint not available yet; polling...")
		}
	}
}

func updateKubeconfigEndpoint(kcData []byte, newEndpoint string) ([]byte, error) {
	var config map[string]interface{}
	if err := yaml.Unmarshal(kcData, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal kubeconfig: %v", err)
	}
	clusters, ok := config["clusters"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("kubeconfig missing 'clusters' field")
	}
	for _, c := range clusters {
		clusterEntry, ok := c.(map[interface{}]interface{})
		if !ok {
			continue
		}
		clusterData, ok := clusterEntry["cluster"].(map[interface{}]interface{})
		if !ok {
			continue
		}
		clusterData["server"] = newEndpoint
	}
	updated, err := yaml.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal updated kubeconfig: %v", err)
	}
	return updated, nil
}
