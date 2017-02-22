// +build integration

package main

import (
	"encoding/json"
	"testing"

	"github.com/atlassian/smith"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	metav1 "k8s.io/client-go/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/apis/meta/v1/unstructured"
)

func TestDeployment(t *testing.T) {

}

func tmplDeploymentResources(r *require.Assertions) []smith.Resource {
	c := apiv1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: apiv1.ObjectMeta{
			Name: "config1",
			Labels: map[string]string{
				"configLabel":           "configValue",
				"overlappingLabel":      "overlappingConfigValue",
				smith.TemplateNameLabel: "configLabel123",
			},
		},
		Data: map[string]string{
			"a": "b",
		},
	}
	data, err := json.Marshal(&c)
	r.NoError(err)

	r1 := unstructured.Unstructured{}
	r.NoError(r1.UnmarshalJSON(data))
	return []smith.Resource{
		{
			Name: "resource1",
			Spec: r1,
		},
	}
}
