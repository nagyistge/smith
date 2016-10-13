package main

import (
	"context"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/ash2k/smith"
	"github.com/ash2k/smith/pkg/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflow(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)
	c := newClient(t, r)

	templateName := "template1"
	templateNamespace := "default"

	var templateCreated bool
	var tmpl smith.Template
	_ = c.Delete(context.Background(), smith.TemplateResourceGroupVersion, templateNamespace, smith.TemplateResourcePath, templateName)
	defer func() {
		if templateCreated {
			// Cleanup after test and after server has stopped
			ctxTimeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			log.Printf("Deleting template %s", tmpl.Name)
			a.Nil(c.Delete(ctxTimeout, smith.TemplateResourceGroupVersion, tmpl.Namespace, smith.TemplateResourcePath, tmpl.Name))
			for _, resource := range tmpl.Spec.Resources {
				log.Printf("Deleting resource %s", resource.Spec.Name)
				a.Nil(c.Delete(ctxTimeout, resource.Spec.APIVersion, tmpl.Namespace, client.ResourceKindToPath(resource.Spec.Kind), resource.Spec.Name))
			}
		}
	}()

	var wg sync.WaitGroup
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := runWithClient(ctx, c); err != nil && err != context.Canceled && err != context.DeadlineExceeded {
			t.Error(err)
		}
	}()

	res := resources()

	time.Sleep(5 * time.Second) // Wait until the app starts and creates the Template TPR

	r.Nil(c.Create(ctx, smith.TemplateResourceGroupVersion, templateNamespace, smith.TemplateResourcePath, &smith.Template{
		TypeMeta: smith.TypeMeta{
			Kind:       smith.TemplateResourceKind,
			APIVersion: smith.TemplateResourceGroupVersion,
		},
		ObjectMeta: smith.ObjectMeta{
			Name: templateName,
		},
		Spec: smith.TemplateSpec{
			Resources: res,
		},
	}, &tmpl))
	templateCreated = true

	for _, resource := range res {
		func() {
			ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			for event := range c.Watch(ctxTimeout, resource.Spec.APIVersion, templateNamespace, client.ResourceKindToPath(resource.Spec.Kind), nil, genericEventFactory) {
				switch ev := event.(type) {
				case error:
					t.Logf("Something went wrong with watch: %v", ev)
				case *smith.GenericWatchEvent:
					if ev.Type == smith.Added &&
						ev.Object.TypeMeta == resource.Spec.TypeMeta &&
						ev.Object.Name == resource.Spec.Name {
						t.Logf("received event for resource %q of kind %q", resource.Spec.Name, resource.Spec.Kind)
						return
					}
					t.Logf("event %#v", ev)
				default:
					t.Fatalf("unexpected event type: %T", event)
				}
			}
			t.Fatalf("expecting event for %q resource of kind %q", resource.Spec.Name, resource.Spec.Kind)
		}()
	}
}

func resources() []smith.Resource {
	tm1 := smith.TypeMeta{
		Kind:       "ConfigMap",
		APIVersion: "v1",
	}
	om1 := smith.ObjectMeta{
		Name: "config1",
	}
	return []smith.Resource{
		{
			ObjectMeta: smith.ObjectMeta{
				Name: "resource1",
			},
			Spec: smith.ResourceSpec{
				TypeMeta:   tm1,
				ObjectMeta: om1,
				Resource: &smith.ConfigMap{
					TypeMeta:   tm1,
					ObjectMeta: om1,
					Data: map[string]string{
						"a": "b",
					},
				},
			},
		},
	}
}