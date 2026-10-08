package apps

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const annotationPrefix = "launcher.example.net/"

var httpRoutes = schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "httproutes"}

type KubernetesSource struct{ client dynamic.Interface }

// KubernetesConfig uses rotating pod credentials when deployed, or the standard
// KUBECONFIG/current-context loading rules on a developer's machine.
func KubernetesConfig() (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}
	if !errors.Is(err, rest.ErrNotInCluster) {
		return nil, fmt.Errorf("load in-cluster Kubernetes configuration: %w", err)
	}
	config, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("load Kubernetes kubeconfig: %w", err)
	}
	return config, nil
}

func NewKubernetesSource(config *rest.Config) (*KubernetesSource, error) {
	bounded := rest.CopyConfig(config)
	bounded.Timeout = requestTimeout
	bounded.UserAgent = "homelab-dashboard"
	client, err := dynamic.NewForConfig(bounded)
	if err != nil {
		return nil, err
	}
	return &KubernetesSource{client: client}, nil
}

func (s *KubernetesSource) List(ctx context.Context) ([]App, error) {
	routes, err := s.client.Resource(httpRoutes).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return normalizeRoutes(routes), nil
}

func normalizeRoutes(routes *unstructured.UnstructuredList) []App {
	result := make([]App, 0, len(routes.Items))
	for _, route := range routes.Items {
		if route.GetName() == "homelab-dashboard-http" {
			continue
		}
		annotations := route.GetAnnotations()
		if strings.EqualFold(strings.TrimSpace(annotations[annotationPrefix+"enabled"]), "false") {
			continue
		}
		hosts, _, err := unstructured.NestedStringSlice(route.Object, "spec", "hostnames")
		if err != nil {
			continue
		}
		internal := []string{}
		seen := map[string]bool{}
		for _, host := range hosts {
			if !strings.HasSuffix(host, ".example.net") || len(validation.IsDNS1123Subdomain(host)) > 0 || seen[host] {
				continue
			}
			internal = append(internal, host)
			seen[host] = true
		}
		if len(internal) == 0 {
			continue
		}
		name := strings.TrimSpace(annotations[annotationPrefix+"name"])
		if name == "" {
			name = humanize(strings.TrimSuffix(route.GetName(), "-http"))
		}
		group := strings.TrimSpace(annotations[annotationPrefix+"group"])
		if group == "" {
			group = "Apps"
		}
		result = append(result, App{
			ID: route.GetNamespace() + "/" + route.GetName(), Name: name, URL: "https://" + internal[0],
			Aliases: append([]string{}, internal[1:]...), Description: annotations[annotationPrefix+"description"],
			Group: group, Icon: annotations[annotationPrefix+"icon"],
		})
	}
	sort.Slice(result, func(i, j int) bool {
		left, right := strings.ToLower(result[i].Name), strings.ToLower(result[j].Name)
		if left == right {
			return result[i].ID < result[j].ID
		}
		return left < right
	})
	return result
}

func humanize(name string) string {
	words := strings.FieldsFunc(name, func(r rune) bool { return r == '-' || r == '_' })
	for i, word := range words {
		letters := []rune(word)
		letters[0] = unicode.ToUpper(letters[0])
		words[i] = string(letters)
	}
	return strings.Join(words, " ")
}
