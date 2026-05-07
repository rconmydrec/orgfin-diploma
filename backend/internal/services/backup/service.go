package backup

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	cronJobName = "orgfin-api-go-backup"
	// namespacePath is the in-cluster path used to auto-detect the pod namespace.
	namespacePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
)

// K8sBackupService implements BackupJobCreator by creating a Kubernetes Job
// from an existing CronJob spec.
type K8sBackupService struct {
	clientset kubernetes.Interface
	namespace string
	logger    *slog.Logger
}

// NewK8sBackupService creates a K8sBackupService using in-cluster config.
// Returns an error if called outside a Kubernetes environment.
func NewK8sBackupService(namespace string, logger *slog.Logger) (*K8sBackupService, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	ns := resolveNamespace(namespace)

	return &K8sBackupService{
		clientset: clientset,
		namespace: ns,
		logger:    logger,
	}, nil
}

// NewK8sBackupServiceWithClient creates a K8sBackupService with an injected
// clientset. This is used for testing with a fake clientset.
func NewK8sBackupServiceWithClient(clientset kubernetes.Interface, namespace string, logger *slog.Logger) *K8sBackupService {
	return &K8sBackupService{
		clientset: clientset,
		namespace: resolveNamespace(namespace),
		logger:    logger,
	}
}

// CreateBackupJob reads the CronJob spec and creates a one-off Job from it.
func (s *K8sBackupService) CreateBackupJob(ctx context.Context) (string, error) {
	cronJob, err := s.clientset.BatchV1().CronJobs(s.namespace).Get(ctx, cronJobName, metav1.GetOptions{})
	if err != nil {
		s.logger.Error("failed to get cronjob", "name", cronJobName, "namespace", s.namespace, "error", err)
		return "", fmt.Errorf("get cronjob %s: %w", cronJobName, err)
	}

	timestamp := time.Now().Format("20060102-150405")
	jobName := fmt.Sprintf("%s-manual-%s", cronJobName, timestamp)

	var ttl int32 = 3600

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: s.namespace,
			Labels: map[string]string{
				"app":        cronJobName,
				"created-by": "manual-trigger",
			},
		},
		Spec: batchv1.JobSpec{
			Template:                cronJob.Spec.JobTemplate.Spec.Template,
			TTLSecondsAfterFinished: &ttl,
		},
	}

	created, err := s.clientset.BatchV1().Jobs(s.namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		s.logger.Error("failed to create backup job", "name", jobName, "namespace", s.namespace, "error", err)
		return "", fmt.Errorf("create job %s: %w", jobName, err)
	}

	s.logger.Info("backup job created", "name", created.Name, "namespace", s.namespace)
	return created.Name, nil
}

// resolveNamespace returns the namespace to use. If the provided namespace is
// non-empty it is returned as-is. Otherwise it attempts to read the in-cluster
// namespace file, falling back to "default".
func resolveNamespace(namespace string) string {
	if namespace != "" {
		return namespace
	}

	data, err := os.ReadFile(namespacePath)
	if err == nil && len(data) > 0 {
		return string(data)
	}

	return "default"
}
