//revive:disable:package-comments
package main

import (
	"context"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	katastromaorgv0 "github.com/katastroma/tropis/apis/katastroma.org/v0"
)

func main() {
	ctrl.SetLogger(zap.New())
	log := ctrl.Log.WithName("adopt")
	ctx := context.Background()

	namespace := os.Getenv("POD_NAMESPACE")
	if namespace == "" {
		log.Error(fmt.Errorf("namespace"), "POD_NAMESPACE is required")
		os.Exit(1)
	}

	rootName := os.Getenv("ROOT_NAME")
	if rootName == "" {
		rootName = katastromaorgv0.DefaultRootAppName
	}

	scheme := runtime.NewScheme()
	if err := katastromaorgv0.AddToScheme(scheme); err != nil {
		log.Error(err, "failed to register application types")
		os.Exit(1)
	}

	c, err := client.New(ctrl.GetConfigOrDie(), client.Options{Scheme: scheme})
	if err != nil {
		log.Error(err, "unable to create client")
		os.Exit(1)
	}

	root := &katastromaorgv0.Application{
		TypeMeta: metav1.TypeMeta{
			APIVersion: katastromaorgv0.Group + "/" + katastromaorgv0.Version,
			Kind:       katastromaorgv0.KindApplication,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      rootName,
		},
	}

	if err := c.Create(ctx, root); client.IgnoreAlreadyExists(err) != nil {
		log.Error(err, "failed to create root Application")
		os.Exit(1)
	}

	log.Info("root Application created", "name", rootName, "namespace", namespace)
}
