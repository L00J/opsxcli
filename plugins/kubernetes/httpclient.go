package kubernetes

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

// K8sHTTPClient 轻量级 Kubernetes HTTP 客户端
type K8sHTTPClient struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

// getHomeDir 获取用户主目录
func getHomeDir() string {
	if runtime.GOOS == "windows" {
		return os.Getenv("USERPROFILE")
	}
	return os.Getenv("HOME")
}

// KubeConfig kubeconfig 文件结构（简化版）
type KubeConfig struct {
	Clusters []struct {
		Name    string `yaml:"name"`
		Cluster struct {
			Server                   string `yaml:"server"`
			CertificateAuthorityData string `yaml:"certificate-authority-data"`
		} `yaml:"cluster"`
	} `yaml:"clusters"`
	Users []struct {
		Name string `yaml:"name"`
		User struct {
			ClientCertificateData string `yaml:"client-certificate-data"`
			ClientKeyData         string `yaml:"client-key-data"`
			Token                 string `yaml:"token"`
			Exec                  *struct {
				APIVersion string   `yaml:"apiVersion"`
				Command    string   `yaml:"command"`
				Args       []string `yaml:"args"`
			} `yaml:"exec"`
		} `yaml:"user"`
	} `yaml:"users"`
	Contexts []struct {
		Name    string `yaml:"name"`
		Context struct {
			Cluster string `yaml:"cluster"`
			User    string `yaml:"user"`
		} `yaml:"context"`
	} `yaml:"contexts"`
	CurrentContext string `yaml:"current-context"`
}

// NewK8sHTTPClient 创建新的 HTTP 客户端
func NewK8sHTTPClient(kubeconfigPath string) (*K8sHTTPClient, error) {
	// 获取 kubeconfig 路径
	if kubeconfigPath == "" {
		if home := getHomeDir(); home != "" {
			kubeconfigPath = filepath.Join(home, ".kube", "config")
		}
	}

	// 读取 kubeconfig
	data, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("读取 kubeconfig 失败: %v", err)
	}

	var config KubeConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析 kubeconfig 失败: %v", err)
	}

	// 查找当前 context
	var currentCluster, currentUser string
	for _, ctx := range config.Contexts {
		if ctx.Name == config.CurrentContext {
			currentCluster = ctx.Context.Cluster
			currentUser = ctx.Context.User
			break
		}
	}

	// 查找 cluster 信息
	var serverURL, caData string
	for _, cluster := range config.Clusters {
		if cluster.Name == currentCluster {
			serverURL = cluster.Cluster.Server
			caData = cluster.Cluster.CertificateAuthorityData
			break
		}
	}

	// 查找用户认证信息
	var token, certData, keyData string
	for _, user := range config.Users {
		if user.Name == currentUser {
			// 检查是否使用 exec 认证提供者（AWS EKS）
			if user.User.Exec != nil {
				// 执行凭证命令获取动态 token
				cmd := exec.Command(user.User.Exec.Command, user.User.Exec.Args...)
				output, err := cmd.Output()
				if err != nil {
					return nil, fmt.Errorf("执行凭证命令失败: %v", err)
				}

				// 解析 JSON 输出获取 token
				var execCredential struct {
					Status struct {
						Token string `json:"token"`
					} `json:"status"`
				}
				if err := json.Unmarshal(output, &execCredential); err != nil {
					return nil, fmt.Errorf("解析凭证输出失败: %v", err)
				}
				token = execCredential.Status.Token
			} else {
				// 使用静态 token 或证书
				token = user.User.Token
			}
			certData = user.User.ClientCertificateData
			keyData = user.User.ClientKeyData
			break
		}
	}

	// 创建 TLS 配置
	tlsConfig := &tls.Config{InsecureSkipVerify: false}

	// 加载 CA 证书
	if caData != "" {
		caCert, err := base64.StdEncoding.DecodeString(caData)
		if err != nil {
			return nil, fmt.Errorf("解码 CA 证书失败: %v", err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig.RootCAs = caCertPool
	}

	// 加载客户端证书
	if certData != "" && keyData != "" {
		cert, err := base64.StdEncoding.DecodeString(certData)
		if err != nil {
			return nil, fmt.Errorf("解码客户端证书失败: %v", err)
		}
		key, err := base64.StdEncoding.DecodeString(keyData)
		if err != nil {
			return nil, fmt.Errorf("解码客户端密钥失败: %v", err)
		}
		clientCert, err := tls.X509KeyPair(cert, key)
		if err != nil {
			return nil, fmt.Errorf("加载客户端证书失败: %v", err)
		}
		tlsConfig.Certificates = []tls.Certificate{clientCert}
	}

	return &K8sHTTPClient{
		baseURL: serverURL,
		httpClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
			},
		},
		token: token,
	}, nil
}

// Get 执行 GET 请求
func (c *K8sHTTPClient) Get(path string, result interface{}) error {
	url := c.baseURL + path

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	// 添加认证
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API 返回错误 %d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

// List 获取资源列表
func (c *K8sHTTPClient) List(apiPath string, result interface{}) error {
	return c.Get(apiPath, result)
}

// ListAllNamespaces 列出所有命名空间的资源
func (c *K8sHTTPClient) ListAllNamespaces(resourceType string, result interface{}) error {
	var path string
	switch resourceType {
	case "deployments":
		path = "/apis/apps/v1/deployments"
	case "services":
		path = "/api/v1/services"
	case "pods":
		path = "/api/v1/pods"
	case "ingresses":
		path = "/apis/networking.k8s.io/v1/ingresses"
	default:
		return fmt.Errorf("不支持的资源类型: %s", resourceType)
	}

	return c.Get(path, result)
}

// ===== 最小化结构体定义 =====

// DeploymentList Deployment 列表
type DeploymentList struct {
	Items []Deployment `json:"items"`
}

// Deployment 部署（完整定义）
type Deployment struct {
	APIVersion string   `json:"apiVersion" yaml:"apiVersion"`
	Kind       string   `json:"kind" yaml:"kind"`
	Metadata   Metadata `json:"metadata" yaml:"metadata"`
	Spec       struct {
		Replicas *int32 `json:"replicas" yaml:"replicas"`
		Selector *struct {
			MatchLabels map[string]string `json:"matchLabels" yaml:"matchLabels"`
		} `json:"selector,omitempty" yaml:"selector,omitempty"`
		Strategy *struct {
			Type string `json:"type,omitempty" yaml:"type,omitempty"`
		} `json:"strategy,omitempty" yaml:"strategy,omitempty"`
		Template struct {
			Metadata Metadata `json:"metadata" yaml:"metadata"`
			Spec     PodSpec  `json:"spec" yaml:"spec"`
		} `json:"template" yaml:"template"`
	} `json:"spec" yaml:"spec"`
	Status *struct {
		AvailableReplicas int32 `json:"availableReplicas" yaml:"availableReplicas,omitempty"`
	} `json:"status,omitempty" yaml:"status,omitempty"`
}

// ServiceList Service 列表
type ServiceList struct {
	Items []Service `json:"items"`
}

// Service 服务（最小化）
type Service struct {
	APIVersion string   `json:"apiVersion" yaml:"apiVersion"`
	Kind       string   `json:"kind" yaml:"kind"`
	Metadata   Metadata `json:"metadata" yaml:"metadata"`
	Spec       struct {
		Type      string            `json:"type" yaml:"type"`
		ClusterIP string            `json:"clusterIP" yaml:"clusterIP"`
		Selector  map[string]string `json:"selector" yaml:"selector"`
		Ports     []struct {
			Name       string      `json:"name,omitempty" yaml:"name,omitempty"`
			Protocol   string      `json:"protocol" yaml:"protocol"`
			Port       int32       `json:"port" yaml:"port"`
			TargetPort interface{} `json:"targetPort,omitempty" yaml:"targetPort,omitempty"` // 可能是 int 或 string（命名端口）
		} `json:"ports" yaml:"ports"`
	} `json:"spec" yaml:"spec"`
}

// IngressList Ingress 列表
type IngressList struct {
	Items []Ingress `json:"items"`
}

// Ingress 入口（完整定义）
type Ingress struct {
	APIVersion string   `json:"apiVersion" yaml:"apiVersion"`
	Kind       string   `json:"kind" yaml:"kind"`
	Metadata   Metadata `json:"metadata" yaml:"metadata"`
	Spec       struct {
		IngressClassName string `json:"ingressClassName,omitempty" yaml:"ingressClassName,omitempty"`
		Rules            []struct {
			Host string `json:"host" yaml:"host"`
			HTTP struct {
				Paths []struct {
					Path     string `json:"path" yaml:"path"`
					PathType string `json:"pathType" yaml:"pathType"`
					Backend  struct {
						Service struct {
							Name string `json:"name" yaml:"name"`
							Port struct {
								Number int32 `json:"number" yaml:"number"`
							} `json:"port" yaml:"port"`
						} `json:"service" yaml:"service"`
					} `json:"backend" yaml:"backend"`
				} `json:"paths" yaml:"paths"`
			} `json:"http" yaml:"http"`
		} `json:"rules" yaml:"rules"`
	} `json:"spec" yaml:"spec"`
}

// PodList Pod 列表
type PodList struct {
	Items []Pod `json:"items"`
}

// Pod 容器组（最小化）
type Pod struct {
	Metadata Metadata  `json:"metadata"`
	Spec     PodSpec   `json:"spec"`
	Status   PodStatus `json:"status"`
}

// PodStatus Pod 状态
type PodStatus struct {
	Phase             string `json:"phase"`
	PodIP             string `json:"podIP"`
	HostIP            string `json:"hostIP"`
	NominatedNodeName string `json:"nominatedNodeName"`
	ContainerStatuses []struct {
		Name         string `json:"name"`
		Ready        bool   `json:"ready"`
		RestartCount int32  `json:"restartCount"`
		State        struct {
			Running *struct{} `json:"running,omitempty"`
		} `json:"state"`
	} `json:"containerStatuses"`
}

// PodSpec Pod 规格
type PodSpec struct {
	NodeName                      string      `json:"nodeName,omitempty" yaml:"nodeName,omitempty"`
	Containers                    []Container `json:"containers" yaml:"containers"`
	TerminationGracePeriodSeconds *int64      `json:"terminationGracePeriodSeconds,omitempty" yaml:"terminationGracePeriodSeconds,omitempty"`
}

// Container 容器（完整定义）
type Container struct {
	Name            string              `json:"name" yaml:"name"`
	Image           string              `json:"image,omitempty" yaml:"image,omitempty"`
	ImagePullPolicy string              `json:"imagePullPolicy,omitempty" yaml:"imagePullPolicy,omitempty"`
	Ports           []ContainerPort     `json:"ports,omitempty" yaml:"ports,omitempty"`
	Env             []EnvVar            `json:"env,omitempty" yaml:"env,omitempty"`
	Resources       ResourceRequirements `json:"resources,omitempty" yaml:"resources,omitempty"`
	LivenessProbe   *Probe              `json:"livenessProbe,omitempty" yaml:"livenessProbe,omitempty"`
	ReadinessProbe  *Probe              `json:"readinessProbe,omitempty" yaml:"readinessProbe,omitempty"`
	Lifecycle       *Lifecycle          `json:"lifecycle,omitempty" yaml:"lifecycle,omitempty"`
}

// ContainerPort 容器端口
type ContainerPort struct {
	Name          string `json:"name,omitempty" yaml:"name,omitempty"`
	ContainerPort int32  `json:"containerPort" yaml:"containerPort"`
	Protocol      string `json:"protocol,omitempty" yaml:"protocol,omitempty"`
}

// EnvVar 环境变量
type EnvVar struct {
	Name  string `json:"name" yaml:"name"`
	Value string `json:"value,omitempty" yaml:"value,omitempty"`
}

// Probe 探针
type Probe struct {
	HTTPGet             *HTTPGetAction `json:"httpGet,omitempty" yaml:"httpGet,omitempty"`
	InitialDelaySeconds int32          `json:"initialDelaySeconds,omitempty" yaml:"initialDelaySeconds,omitempty"`
	TimeoutSeconds      int32          `json:"timeoutSeconds,omitempty" yaml:"timeoutSeconds,omitempty"`
	PeriodSeconds       int32          `json:"periodSeconds,omitempty" yaml:"periodSeconds,omitempty"`
	SuccessThreshold    int32          `json:"successThreshold,omitempty" yaml:"successThreshold,omitempty"`
	FailureThreshold    int32          `json:"failureThreshold,omitempty" yaml:"failureThreshold,omitempty"`
}

// HTTPGetAction HTTP GET 探针
type HTTPGetAction struct {
	Path   string      `json:"path" yaml:"path"`
	Port   interface{} `json:"port" yaml:"port"` // 可以是 int 或 string
	Scheme string      `json:"scheme,omitempty" yaml:"scheme,omitempty"`
}

// Lifecycle 生命周期钩子
type Lifecycle struct {
	PreStop *LifecycleHandler `json:"preStop,omitempty" yaml:"preStop,omitempty"`
}

// LifecycleHandler 生命周期处理器
type LifecycleHandler struct {
	Exec *ExecAction `json:"exec,omitempty" yaml:"exec,omitempty"`
}

// ExecAction 执行动作
type ExecAction struct {
	Command []string `json:"command" yaml:"command"`
}

// ResourceRequirements 资源需求
type ResourceRequirements struct {
	Requests ResourceList `json:"requests,omitempty"`
	Limits   ResourceList `json:"limits,omitempty"`
}

// ResourceList 资源列表
type ResourceList map[string]string

// Metadata 元数据
type Metadata struct {
	Name              string            `json:"name" yaml:"name"`
	Namespace         string            `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Labels            map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Annotations       map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	CreationTimestamp string            `json:"creationTimestamp,omitempty" yaml:"-"` // 导出时忽略
}
