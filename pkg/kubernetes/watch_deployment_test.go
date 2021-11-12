package kubernetes

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func TestGetReasonMessageMapFromStatuses(t *testing.T) {

	type args struct {
		containerStatus  []corev1.ContainerStatus
		reasonMessageMap map[string]string
	}
	tests := []struct {
		name           string
		args           args
		expectedOutput map[string]string
	}{
		{
			name: "test 1",
			args: args{
				containerStatus: []corev1.ContainerStatus{
					{
						State: corev1.ContainerState{
							Waiting: &corev1.ContainerStateWaiting{
								Reason:  "a",
								Message: "b",
							},
						},
					},
				},
				reasonMessageMap: map[string]string{},
			},
			expectedOutput: map[string]string{"a": "b"},
		},
		{
			name: "test 2",
			args: args{
				containerStatus: []corev1.ContainerStatus{
					{
						State: corev1.ContainerState{
							Waiting: nil,
						},
					},
				},
				reasonMessageMap: map[string]string{},
			},
			expectedOutput: map[string]string{},
		},
		{
			name: "test 3",
			args: args{
				containerStatus: []corev1.ContainerStatus{
					{
						State: corev1.ContainerState{
							Waiting: &corev1.ContainerStateWaiting{
								Reason:  "ContainerCreating",
								Message: "b",
							},
						},
					},
				},
				reasonMessageMap: map[string]string{},
			},
			expectedOutput: map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if output := getReasonMessageMapFromStatuses(tt.args.containerStatus, tt.args.reasonMessageMap); !reflect.DeepEqual(output, tt.expectedOutput) {
				t.Errorf("getReasonMessageMapFromStatuses() = %v, expectedOutput %v", output, tt.expectedOutput)
			}
		})
	}
}

func TestGetReasonMessageMapFromPodConditions(t *testing.T) {
	type args struct {
		conditions       []corev1.PodCondition
		reasonMessageMap map[string]string
	}
	tests := []struct {
		name           string
		args           args
		expectedOutput map[string]string
	}{
		{
			name: "test 1",
			args: args{
				conditions: []corev1.PodCondition{
					{
						Status:  corev1.ConditionFalse,
						Reason:  "a",
						Message: "b",
					},
				},
				reasonMessageMap: map[string]string{},
			},
			expectedOutput: map[string]string{"a": "b"},
		},
		{
			name: "test 2",
			args: args{
				conditions: []corev1.PodCondition{
					{
						Status:  corev1.ConditionTrue,
						Reason:  "a",
						Message: "b",
					},
				},
				reasonMessageMap: map[string]string{},
			},
			expectedOutput: map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if output := getReasonMessageMapFromPodConditions(tt.args.conditions, tt.args.reasonMessageMap); !reflect.DeepEqual(output, tt.expectedOutput) {
				t.Errorf("getReasonMessageMapFromPodConditions() = %v, expectedOutput %v", output, tt.expectedOutput)
			}
		})
	}
}

func TestGetPods(t *testing.T) {
	podo := corev1.Pod{
		TypeMeta:   metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test", Labels: map[string]string{"foo": "bar"}},
		Spec:       corev1.PodSpec{},
	}
	client := k8sfake.NewSimpleClientset(&podo)

	type args struct {
		ctx           context.Context
		namespace     string
		labelSelector string
	}
	tests := []struct {
		name           string
		args           args
		client         *k8sfake.Clientset
		expectedOutput []corev1.Pod
	}{
		{
			name: "test 1",
			args: args{
				namespace:     "test",
				labelSelector: "foo=bar",
			},
			client: client,
			expectedOutput: []corev1.Pod{
				{
					TypeMeta:   metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test", Labels: map[string]string{"foo": "bar"}},
					Spec:       corev1.PodSpec{},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, _ := getPods(tt.args.ctx, tt.client, tt.args.namespace, tt.args.labelSelector)
			if !reflect.DeepEqual(output, tt.expectedOutput) {
				t.Errorf("getPods() = %v, expectedOutput %v", output, tt.expectedOutput)
			}
		})
	}
}

func TestGetErrorEvents(t *testing.T) {
	podo := corev1.Pod{
		TypeMeta:   metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test", Labels: map[string]string{"foo": "bar"}},
		Spec:       corev1.PodSpec{},
		Status: corev1.PodStatus{
			Conditions: []corev1.PodCondition{
				{
					Reason:  "conditionReason",
					Message: "conditionMessage",
				},
			},
			InitContainerStatuses: []corev1.ContainerStatus{
				{
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason:  "InitContainerStatusReason",
							Message: "InitContainerStatusMessage",
						},
					},
				},
			},

			ContainerStatuses: []corev1.ContainerStatus{
				{
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason:  "containerStatusReason",
							Message: "containerStatusMessage",
						},
					},
				},
			},
		},
	}
	var ProgressDeadlineSeconds int32
	ProgressDeadlineSeconds = 10
	deployment := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
			Labels:    map[string]string{"foo": "bar"},
		},
		Spec: appsv1.DeploymentSpec{
			ProgressDeadlineSeconds: &ProgressDeadlineSeconds,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"foo": "bar"},
				},
				Spec: podo.Spec,
			},
		},
		Status: appsv1.DeploymentStatus{},
	}
	rs := appsv1.ReplicaSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
			Labels:    map[string]string{"foo": "bar"},
		},
		Spec: appsv1.ReplicaSetSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"foo": "bar"},
				},
			},
		},
	}
	client := k8sfake.NewSimpleClientset(&podo, &deployment, &rs)

	outputString := fmt.Sprintf("*Rollout for Deployment `test` (RS: `test`) in `test` namespace failed after `10` seconds on the `` cluster.*\n\n*Retrieved the following errors:*\n```\n* InitContainerStatusReason - InitContainerStatusMessage\n```\n```\n* containerStatusReason - containerStatusMessage\n```\n```\n* conditionReason - conditionMessage\n```")
	type args struct {
		ctx           context.Context
		namespace     string
		newDeployment *appsv1.Deployment
	}
	tests := []struct {
		name           string
		args           args
		expectedOutput string
	}{
		{
			name: "test 1",
			args: args{
				namespace:     "test",
				newDeployment: &deployment,
			},
			expectedOutput: outputString,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, _ := getErrorEvents(tt.args.ctx, client, tt.args.namespace, tt.args.newDeployment)
			if output != tt.expectedOutput {
				t.Errorf("getErrorEvents() = %v, expectedOutput %v", output, tt.expectedOutput)
			}
		})
	}
}
