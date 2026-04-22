package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- parseCPU (from top.go) ---

func TestParseCPU_Nanocores(t *testing.T) {
	assert.Equal(t, int64(100), parseCPU("100n"))
}

func TestParseCPU_Millicores(t *testing.T) {
	assert.Equal(t, int64(100*1000000), parseCPU("100m"))
}

func TestParseCPU_Cores(t *testing.T) {
	assert.Equal(t, int64(2*1000000000), parseCPU("2"))
}

func TestParseCPU_Empty(t *testing.T) {
	assert.Equal(t, int64(0), parseCPU(""))
}

func TestParseCPU_Whitespace(t *testing.T) {
	assert.Equal(t, int64(0), parseCPU("  "))
}

// --- parseMemory (from top.go) ---

func TestParseMemory_Ki(t *testing.T) {
	assert.Equal(t, int64(1024*100), parseMemory("100Ki"))
}

func TestParseMemory_Mi(t *testing.T) {
	assert.Equal(t, int64(1024*1024*256), parseMemory("256Mi"))
}

func TestParseMemory_Gi(t *testing.T) {
	assert.Equal(t, int64(1024*1024*1024*2), parseMemory("2Gi"))
}

func TestParseMemory_Bytes(t *testing.T) {
	assert.Equal(t, int64(1024), parseMemory("1024"))
}

func TestParseMemory_Empty(t *testing.T) {
	assert.Equal(t, int64(0), parseMemory(""))
}

// --- formatCPU (from top.go) ---

func TestFormatCPU_Millicores(t *testing.T) {
	assert.Equal(t, "500m", formatCPU(500000000))
}

func TestFormatCPU_Zero(t *testing.T) {
	assert.Equal(t, "0m", formatCPU(0))
}

func TestFormatCPU_Small(t *testing.T) {
	assert.Equal(t, "1m", formatCPU(1000000))
}

// --- formatMemory (from top.go) ---

func TestFormatMemory_Mi(t *testing.T) {
	assert.Equal(t, "256Mi", formatMemory(256*1024*1024))
}

func TestFormatMemory_Zero(t *testing.T) {
	assert.Equal(t, "0Mi", formatMemory(0))
}

func TestFormatMemory_Gi(t *testing.T) {
	assert.Equal(t, "2048Mi", formatMemory(2*1024*1024*1024))
}

// --- extractDeploymentName (from resource.go) ---

func TestExtractDeploymentName_Standard(t *testing.T) {
	result := extractDeploymentName("my-app-7d9b8f6c5d-x2k9p")
	assert.Equal(t, "my-app", result)
}

func TestExtractDeploymentName_TwoParts(t *testing.T) {
	result := extractDeploymentName("my-app-abc")
	assert.Equal(t, "my-app", result)
}

func TestExtractDeploymentName_NoDash(t *testing.T) {
	result := extractDeploymentName("mypod")
	assert.Equal(t, "mypod", result)
}

func TestExtractDeploymentName_Empty(t *testing.T) {
	result := extractDeploymentName("")
	assert.Equal(t, "", result)
}

func TestExtractDeploymentName_SingleDash(t *testing.T) {
	// "my-app" -> splits to ["my", "app"], lastPart "app" is alphanumeric len 3 -> strips to "my"
	result := extractDeploymentName("my-app")
	assert.Equal(t, "my", result)
}

func TestExtractDeploymentName_LongName(t *testing.T) {
	// "my-deployment-name" has no 5-char suffix + 8-10 char hash pattern
	result := extractDeploymentName("my-deployment-name")
	// lastPart "name" is alphanumeric len 4 >= 3, so strips to "my-deployment"
	assert.Equal(t, "my-deployment", result)
}

// --- isAlphaNumeric (from resource.go) ---

func TestIsAlphaNumeric_Alpha(t *testing.T) {
	assert.True(t, isAlphaNumeric("abc"))
}

func TestIsAlphaNumeric_Numeric(t *testing.T) {
	assert.True(t, isAlphaNumeric("123"))
}

func TestIsAlphaNumeric_Alphanumeric(t *testing.T) {
	assert.True(t, isAlphaNumeric("abc123"))
}

func TestIsAlphaNumeric_Special(t *testing.T) {
	assert.False(t, isAlphaNumeric("abc-123"))
}

func TestIsAlphaNumeric_Empty(t *testing.T) {
	assert.False(t, isAlphaNumeric(""))
}

// --- hasUpperCase (from resource.go) ---

func TestHasUpperCase_WithUpper(t *testing.T) {
	assert.True(t, hasUpperCase("abcA"))
}

func TestHasUpperCase_NoUpper(t *testing.T) {
	assert.False(t, hasUpperCase("abc"))
}

func TestHasUpperCase_Numbers(t *testing.T) {
	assert.False(t, hasUpperCase("123"))
}

func TestHasUpperCase_Empty(t *testing.T) {
	assert.False(t, hasUpperCase(""))
}

// --- parseCPUValue (from resource.go) ---

func TestParseCPUValue_Millicores(t *testing.T) {
	assert.Equal(t, 0.5, parseCPUValue("500m"))
}

func TestParseCPUValue_Cores(t *testing.T) {
	assert.Equal(t, 2.0, parseCPUValue("2"))
}

func TestParseCPUValue_Empty(t *testing.T) {
	assert.Equal(t, 0.0, parseCPUValue(""))
}

func TestParseCPUValue_Small(t *testing.T) {
	assert.InDelta(t, 0.1, parseCPUValue("100m"), 0.001)
}

// --- parseMemoryValue (from resource.go) ---

func TestParseMemoryValue_Mi(t *testing.T) {
	assert.Equal(t, 256.0, parseMemoryValue("256Mi"))
}

func TestParseMemoryValue_Gi(t *testing.T) {
	assert.Equal(t, 2048.0, parseMemoryValue("2Gi"))
}

func TestParseMemoryValue_Ki(t *testing.T) {
	assert.InDelta(t, 100.0/1024.0, parseMemoryValue("100Ki"), 0.001)
}

func TestParseMemoryValue_Empty(t *testing.T) {
	assert.Equal(t, 0.0, parseMemoryValue(""))
}

func TestParseMemoryValue_Lowercase(t *testing.T) {
	assert.Equal(t, 256.0, parseMemoryValue("256mi"))
}

func TestParseMemoryValue_M(t *testing.T) {
	assert.Equal(t, 256.0, parseMemoryValue("256M"))
}

func TestParseMemoryValue_G(t *testing.T) {
	assert.Equal(t, 2000.0, parseMemoryValue("2G"))
}

func TestParseMemoryValue_K(t *testing.T) {
	assert.InDelta(t, 100.0/1000.0, parseMemoryValue("100K"), 0.001)
}

func TestParseMemoryValue_T(t *testing.T) {
	assert.Equal(t, 2e6, parseMemoryValue("2T"))
}

func TestParseMemoryValue_Ti(t *testing.T) {
	assert.Equal(t, 2*1024.0*1024.0, parseMemoryValue("2Ti"))
}

// --- formatCPUValue (from resource.go) ---

func TestFormatCPUValue_Zero(t *testing.T) {
	assert.Equal(t, "0m", formatCPUValue(0))
}

func TestFormatCPUValue_Millicore(t *testing.T) {
	assert.Equal(t, "500m", formatCPUValue(0.5))
}

func TestFormatCPUValue_Cores(t *testing.T) {
	assert.Equal(t, "2.00 cores", formatCPUValue(2.0))
}

func TestFormatCPUValue_Small(t *testing.T) {
	assert.Equal(t, "100m", formatCPUValue(0.1))
}

func TestFormatCPUValue_Decimal(t *testing.T) {
	assert.Equal(t, "50.5m", formatCPUValue(0.0505))
}

// --- formatMemoryValue (from resource.go) ---

func TestFormatMemoryValue_Zero(t *testing.T) {
	assert.Equal(t, "0 Mi", formatMemoryValue(0))
}

func TestFormatMemoryValue_Mi(t *testing.T) {
	assert.Equal(t, "512 Mi", formatMemoryValue(512))
}

func TestFormatMemoryValue_Gi(t *testing.T) {
	assert.Equal(t, "1.00 Gi", formatMemoryValue(1024))
}

func TestFormatMemoryValue_DecimalMi(t *testing.T) {
	assert.Equal(t, "512.5 Mi", formatMemoryValue(512.5))
}

// --- parseCapacityProvisioned (from resource.go) ---

func TestParseCapacityProvisioned_Normal(t *testing.T) {
	cpu, mem := parseCapacityProvisioned("4vCPU 8GB")
	assert.Equal(t, 4.0, cpu)
	assert.Equal(t, 8192.0, mem)
}

func TestParseCapacityProvisioned_Empty(t *testing.T) {
	cpu, mem := parseCapacityProvisioned("")
	assert.Equal(t, 0.0, cpu)
	assert.Equal(t, 0.0, mem)
}

func TestParseCapacityProvisioned_NA(t *testing.T) {
	cpu, mem := parseCapacityProvisioned("N/A")
	assert.Equal(t, 0.0, cpu)
	assert.Equal(t, 0.0, mem)
}

func TestParseCapacityProvisioned_CPUOnly(t *testing.T) {
	cpu, mem := parseCapacityProvisioned("2vCPU")
	assert.Equal(t, 2.0, cpu)
	assert.Equal(t, 0.0, mem)
}

func TestParseCapacityProvisioned_MemoryOnly(t *testing.T) {
	cpu, mem := parseCapacityProvisioned("16GB")
	assert.Equal(t, 0.0, cpu)
	assert.Equal(t, 16384.0, mem)
}

func TestParseCapacityProvisioned_DecimalCPU(t *testing.T) {
	cpu, mem := parseCapacityProvisioned("2.5vCPU 4GB")
	assert.Equal(t, 2.5, cpu)
	assert.Equal(t, 4096.0, mem)
}

// --- cleanAnnotations (from yaml.go) ---

func TestCleanAnnotations_Nil(t *testing.T) {
	assert.Nil(t, cleanAnnotations(nil))
}

func TestCleanAnnotations_RuntimeFields(t *testing.T) {
	annotations := map[string]string{
		"kubectl.kubernetes.io/last-applied-configuration": "{}",
		"deployment.kubernetes.io/revision":                "1",
		"my-custom-annotation":                             "value",
	}
	cleaned := cleanAnnotations(annotations)
	assert.Equal(t, 1, len(cleaned))
	assert.Equal(t, "value", cleaned["my-custom-annotation"])
}

func TestCleanAnnotations_AllRuntime(t *testing.T) {
	annotations := map[string]string{
		"kubectl.kubernetes.io/last-applied-configuration": "{}",
		"deployment.kubernetes.io/revision":                "1",
	}
	cleaned := cleanAnnotations(annotations)
	assert.Nil(t, cleaned)
}

func TestCleanAnnotations_NoRuntime(t *testing.T) {
	annotations := map[string]string{
		"app":     "myapp",
		"version": "1.0",
	}
	cleaned := cleanAnnotations(annotations)
	assert.Equal(t, 2, len(cleaned))
}

// --- cleanDeployment (from yaml.go) ---

func TestCleanDeployment_ClearsRuntime(t *testing.T) {
	replicas := int32(3)
	dep := &Deployment{
		Metadata: Metadata{
			CreationTimestamp: "2024-01-01T00:00:00Z",
			Annotations: map[string]string{
				"kubectl.kubernetes.io/last-applied-configuration": "{}",
				"my-annotation": "value",
			},
		},
	}
	dep.Spec.Replicas = &replicas
	dep.Spec.Template.Metadata.Name = "should-be-cleared"
	dep.Spec.Template.Metadata.Namespace = "should-be-cleared"
	dep.Spec.Template.Metadata.CreationTimestamp = "2024-01-01"
	dep.Spec.Template.Metadata.Annotations = map[string]string{"key": "val"}
	dep.Spec.Template.Spec.NodeName = "node-1"
	dep.Status = &struct {
		AvailableReplicas int32 `json:"availableReplicas" yaml:"availableReplicas,omitempty"`
	}{AvailableReplicas: 3}

	cleanDeployment(dep)

	assert.Equal(t, "", dep.Metadata.CreationTimestamp)
	assert.Equal(t, 1, len(dep.Metadata.Annotations))
	assert.Equal(t, "", dep.Spec.Template.Metadata.Name)
	assert.Equal(t, "", dep.Spec.Template.Metadata.Namespace)
	assert.Equal(t, "", dep.Spec.Template.Metadata.CreationTimestamp)
	assert.Equal(t, "", dep.Spec.Template.Spec.NodeName)
	assert.Nil(t, dep.Status)
}

// --- cleanService (from yaml.go) ---

func TestCleanService_ClearsRuntime(t *testing.T) {
	svc := &Service{
		Metadata: Metadata{
			CreationTimestamp: "2024-01-01T00:00:00Z",
			Annotations: map[string]string{
				"deployment.kubernetes.io/revision": "1",
				"custom":                            "val",
			},
		},
	}
	svc.Spec.ClusterIP = "10.0.0.1"

	cleanService(svc)

	assert.Equal(t, "", svc.Metadata.CreationTimestamp)
	assert.Equal(t, "", svc.Spec.ClusterIP)
	assert.Equal(t, 1, len(svc.Metadata.Annotations))
}

// --- cleanIngress (from yaml.go) ---

func TestCleanIngress_ClearsRuntime(t *testing.T) {
	ing := &Ingress{
		Metadata: Metadata{
			CreationTimestamp: "2024-01-01T00:00:00Z",
			Annotations:       map[string]string{"key": "val"},
		},
	}

	cleanIngress(ing)

	assert.Equal(t, "", ing.Metadata.CreationTimestamp)
	assert.Equal(t, 1, len(ing.Metadata.Annotations))
}

// --- matchesSelector (from yaml.go) ---

func TestMatchesSelector_Match(t *testing.T) {
	labels := map[string]string{"app": "myapp", "version": "v1"}
	selector := map[string]string{"app": "myapp"}
	assert.True(t, matchesSelector(labels, selector))
}

func TestMatchesSelector_NoMatch(t *testing.T) {
	labels := map[string]string{"app": "other"}
	selector := map[string]string{"app": "myapp"}
	assert.False(t, matchesSelector(labels, selector))
}

func TestMatchesSelector_EmptySelector(t *testing.T) {
	labels := map[string]string{"app": "myapp"}
	// empty selector returns false per implementation
	assert.False(t, matchesSelector(labels, nil))
}

func TestMatchesSelector_NilLabels(t *testing.T) {
	selector := map[string]string{"app": "myapp"}
	assert.False(t, matchesSelector(nil, selector))
}

func TestMatchesSelector_MultipleKeys(t *testing.T) {
	labels := map[string]string{"app": "myapp", "version": "v1", "env": "prod"}
	selector := map[string]string{"app": "myapp", "env": "prod"}
	assert.True(t, matchesSelector(labels, selector))
}

// --- writeYAML (from yaml.go) ---

func TestWriteYAML_CreatesFile(t *testing.T) {
	tmpDir := t.TempDir()
	filename := tmpDir + "/test.yaml"

	data := map[string]string{"key": "value"}
	err := writeYAML(filename, []interface{}{data})
	assert.NoError(t, err)
}

func TestWriteYAML_MultipleResourcesPure(t *testing.T) {
	tmpDir := t.TempDir()
	filename := tmpDir + "/multi.yaml"

	res1 := map[string]string{"kind": "Deployment"}
	res2 := map[string]string{"kind": "Service"}
	err := writeYAML(filename, []interface{}{res1, res2})
	assert.NoError(t, err)
}

// --- systemNamespaces (from resource.go) ---

func TestSystemNamespaces_Contains(t *testing.T) {
	assert.True(t, systemNamespaces["kube-system"])
	assert.True(t, systemNamespaces["ingress-nginx"])
	assert.True(t, systemNamespaces["monitor"])
	assert.False(t, systemNamespaces["default"])
	assert.False(t, systemNamespaces["production"])
}
