package backup

import (
	"context"
	"log/slog"
	"os"
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func createFakeCronJob(namespace string) *batchv1.CronJob {
	return &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cronJobName,
			Namespace: namespace,
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "0 14 * * *",
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:    "backup",
									Image:   "postgres:16-alpine",
									Command: []string{"/bin/sh", "-c", "pg_dump"},
								},
							},
							RestartPolicy: corev1.RestartPolicyNever,
						},
					},
				},
			},
		},
	}
}

func TestCreateBackupJob_Success(t *testing.T) {
	namespace := "test-ns"
	cronJob := createFakeCronJob(namespace)
	clientset := fake.NewSimpleClientset(cronJob)

	svc := NewK8sBackupServiceWithClient(clientset, namespace, testLogger())

	jobName, err := svc.CreateBackupJob(context.Background())
	require.NoError(t, err)
	assert.Contains(t, jobName, "orgfin-api-go-backup-manual-")

	// Verify the job was actually created in the fake cluster
	jobs, err := clientset.BatchV1().Jobs(namespace).List(context.Background(), metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, jobs.Items, 1)

	job := jobs.Items[0]
	assert.Equal(t, jobName, job.Name)
	assert.Equal(t, namespace, job.Namespace)
	assert.NotNil(t, job.Spec.TTLSecondsAfterFinished)
	assert.Equal(t, int32(3600), *job.Spec.TTLSecondsAfterFinished)

	// Verify the job spec comes from the CronJob's jobTemplate
	assert.Equal(t, "postgres:16-alpine", job.Spec.Template.Spec.Containers[0].Image)
	assert.Equal(t, "backup", job.Spec.Template.Spec.Containers[0].Name)

	// Verify labels
	assert.Equal(t, cronJobName, job.Labels["app"])
	assert.Equal(t, "manual-trigger", job.Labels["created-by"])
}

func TestCreateBackupJob_CronJobNotFound(t *testing.T) {
	namespace := "test-ns"
	clientset := fake.NewSimpleClientset() // No CronJob registered

	svc := NewK8sBackupServiceWithClient(clientset, namespace, testLogger())

	jobName, err := svc.CreateBackupJob(context.Background())
	assert.Error(t, err)
	assert.Empty(t, jobName)
	assert.Contains(t, err.Error(), "get cronjob")
}

func TestCreateBackupJob_JobCreationFailure(t *testing.T) {
	namespace := "test-ns"
	cronJob := createFakeCronJob(namespace)
	clientset := fake.NewSimpleClientset(cronJob)

	// Add a reactor that makes Job creation fail
	clientset.PrependReactor("create", "jobs", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, assert.AnError
	})

	svc := NewK8sBackupServiceWithClient(clientset, namespace, testLogger())

	jobName, err := svc.CreateBackupJob(context.Background())
	assert.Error(t, err)
	assert.Empty(t, jobName)
	assert.Contains(t, err.Error(), "create job")
}

func TestResolveNamespace_ExplicitValue(t *testing.T) {
	ns := resolveNamespace("my-namespace")
	assert.Equal(t, "my-namespace", ns)
}

func TestResolveNamespace_FallbackToDefault(t *testing.T) {
	// When namespace is empty and the in-cluster file doesn't exist,
	// it should fall back to "default".
	ns := resolveNamespace("")
	assert.Equal(t, "default", ns)
}
