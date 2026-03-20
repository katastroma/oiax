//revive:disable:package-comments
package main

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"os"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/watch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/katastroma/pedalion/internal"
)

//go:embed crds/katastroma.org_applications.yaml
var applicationCRD []byte

//go:embed crds/katastroma.org_repocredentials.yaml
var repoCredentialCRD []byte

func main() {
	ctrl.SetLogger(zap.New())
	log := ctrl.Log.WithName("init")
	ctx := context.Background()
	restConfig := ctrl.GetConfigOrDie()

	scheme := runtime.NewScheme()
	if err := apiextensionsv1.AddToScheme(scheme); err != nil {
		log.Error(err, "failed to register apiextensions types")
		os.Exit(1)
	}

	c, err := client.New(restConfig, client.Options{Scheme: scheme})
	if err != nil {
		log.Error(err, "unable to create client")
		os.Exit(1)
	}

	crds := [][]byte{applicationCRD, repoCredentialCRD}

	// Apply each CRD via SSA
	for _, raw := range crds {
		u := &unstructured.Unstructured{}
		err = yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(raw), len(raw)).Decode(u)
		if err != nil {
			log.Error(err, "failed to decode CRD YAML")
			os.Exit(1)
		}

		cfg := client.ApplyConfigurationFromUnstructured(u)
		err = c.Apply(ctx, cfg, client.FieldOwner(internal.FieldOwner), client.ForceOwnership)
		if err != nil {
			log.Error(err, "failed to apply CRD", "name", u.GetName())
			os.Exit(1)
		}

		log.Info("CRD applied", "name", u.GetName())
	}

	// Watch for all CRDs to become Established
	extClient, err := apiextensionsclient.NewForConfig(restConfig)
	if err != nil {
		log.Error(err, "unable to create apiextensions client")
		os.Exit(1)
	}

	for _, raw := range crds {
		u := &unstructured.Unstructured{}
		yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(raw), len(raw)).Decode(u)
		name := u.GetName()

		watcher, err := extClient.ApiextensionsV1().CustomResourceDefinitions().Watch(ctx, metav1.ListOptions{
			FieldSelector: fields.OneTermEqualSelector("metadata.name", name).String(),
		})
		if err != nil {
			log.Error(err, "failed to watch CRD", "name", name)
			os.Exit(1)
		}

		established := false
		for event := range watcher.ResultChan() {
			if event.Type == watch.Error {
				log.Error(fmt.Errorf("watch error"), "unexpected watch error", "name", name)
				os.Exit(1)
			}

			crd, ok := event.Object.(*apiextensionsv1.CustomResourceDefinition)
			if !ok {
				continue
			}

			for _, cond := range crd.Status.Conditions {
				if cond.Type == apiextensionsv1.Established && cond.Status == apiextensionsv1.ConditionTrue {
					log.Info("CRD established", "name", name)
					established = true
					break
				}
			}

			if established {
				break
			}
		}

		watcher.Stop()

		if !established {
			log.Error(fmt.Errorf("watch closed"), "CRD not established", "name", name)
			os.Exit(1)
		}
	}
}
