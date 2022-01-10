/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package status

import (
	"testing"

	apiv1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/stolostron/management-ingress/pkg/ingress/annotations/class"
	"github.com/stolostron/management-ingress/pkg/ingress/store"
	"github.com/stolostron/management-ingress/pkg/k8s"
	"github.com/stolostron/management-ingress/pkg/task"
)

func buildLoadBalancerIngressByIP() []apiv1.LoadBalancerIngress {
	return []apiv1.LoadBalancerIngress{
		{
			IP:       "10.0.0.1",
			Hostname: "foo1",
		},
		{
			IP:       "10.0.0.2",
			Hostname: "foo2",
		},
		{
			IP:       "10.0.0.3",
			Hostname: "",
		},
		{
			IP:       "",
			Hostname: "foo4",
		},
	}
}

func buildSimpleClientSet() *testclient.Clientset {
	return testclient.NewSimpleClientset(
		&apiv1.PodList{Items: []apiv1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo1",
					Namespace: apiv1.NamespaceDefault,
					Labels: map[string]string{
						"lable_sig": "foo_pod",
					},
				},
				Spec: apiv1.PodSpec{
					NodeName: "foo_node_2",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo2",
					Namespace: apiv1.NamespaceDefault,
					Labels: map[string]string{
						"lable_sig": "foo_no",
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo3",
					Namespace: metav1.NamespaceSystem,
					Labels: map[string]string{
						"lable_sig": "foo_pod",
					},
				},
				Spec: apiv1.PodSpec{
					NodeName: "foo_node_2",
				},
			},
		}},
		&apiv1.ServiceList{Items: []apiv1.Service{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: apiv1.NamespaceDefault,
				},
				Status: apiv1.ServiceStatus{
					LoadBalancer: apiv1.LoadBalancerStatus{
						Ingress: buildLoadBalancerIngressByIP(),
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo_non_exist",
					Namespace: apiv1.NamespaceDefault,
				},
			},
		}},
		&apiv1.NodeList{Items: []apiv1.Node{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo_node_1",
				},
				Status: apiv1.NodeStatus{
					Addresses: []apiv1.NodeAddress{
						{
							Type:    apiv1.NodeInternalIP,
							Address: "10.0.0.1",
						}, {
							Type:    apiv1.NodeExternalIP,
							Address: "10.0.0.2",
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo_node_2",
				},
				Status: apiv1.NodeStatus{
					Addresses: []apiv1.NodeAddress{
						{
							Type:    apiv1.NodeInternalIP,
							Address: "11.0.0.1",
						},
						{
							Type:    apiv1.NodeExternalIP,
							Address: "11.0.0.2",
						},
					},
				},
			},
		}},
		&apiv1.EndpointsList{Items: []apiv1.Endpoints{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress-controller-leader",
					Namespace: apiv1.NamespaceDefault,
					SelfLink:  "/api/v1/namespaces/default/endpoints/ingress-controller-leader",
				},
			}}},
		&networking.IngressList{Items: buildIngresses()},
	)
}

func fakeSynFn(interface{}) error {
	return nil
}

func buildIngresses() []networking.Ingress {
	return []networking.Ingress{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo_ingress_1",
				Namespace: apiv1.NamespaceDefault,
			},
			Status: networking.IngressStatus{
				LoadBalancer: apiv1.LoadBalancerStatus{
					Ingress: []apiv1.LoadBalancerIngress{
						{
							IP:       "10.0.0.1",
							Hostname: "foo1",
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo_ingress_different_class",
				Namespace: metav1.NamespaceDefault,
				Annotations: map[string]string{
					class.IngressKey: "no-nginx",
				},
			},
			Status: networking.IngressStatus{
				LoadBalancer: apiv1.LoadBalancerStatus{
					Ingress: []apiv1.LoadBalancerIngress{
						{
							IP:       "0.0.0.0",
							Hostname: "foo.bar.com",
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo_ingress_2",
				Namespace: apiv1.NamespaceDefault,
			},
			Status: networking.IngressStatus{
				LoadBalancer: apiv1.LoadBalancerStatus{
					Ingress: []apiv1.LoadBalancerIngress{},
				},
			},
		},
	}
}

func buildIngressListener() store.IngressLister {
	s := cache.NewStore(cache.MetaNamespaceKeyFunc)
	s.Add(&networking.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo_ingress_non_01",
			Namespace: apiv1.NamespaceDefault,
		}})
	s.Add(&networking.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo_ingress_1",
			Namespace: apiv1.NamespaceDefault,
		},
		Status: networking.IngressStatus{
			LoadBalancer: apiv1.LoadBalancerStatus{
				Ingress: buildLoadBalancerIngressByIP(),
			},
		},
	})

	return store.IngressLister{Store: s}
}

func buildStatusSync() statusSync {
	return statusSync{
		pod: &k8s.PodInfo{
			Name:      "foo_base_pod",
			Namespace: apiv1.NamespaceDefault,
			Labels: map[string]string{
				"lable_sig": "foo_pod",
			},
		},
		syncQueue: task.NewTaskQueue(fakeSynFn),
		Config: Config{
			Client:        buildSimpleClientSet(),
			IngressLister: buildIngressListener(),
		},
	}
}

func TestCallback(t *testing.T) {
	buildStatusSync()
}

func TestKeyfunc(t *testing.T) {
	fk := buildStatusSync()

	i := "foo_base_pod"
	r, err := fk.keyfunc(i)

	if err != nil {
		t.Fatalf("unexpected error")
	}
	if r != i {
		t.Errorf("returned %v but expected %v", r, i)
	}
}

func TestRunningAddresessWithPods(t *testing.T) {
	fk := buildStatusSync()

	r, _ := fk.runningAddresses()
	if r == nil {
		t.Fatalf("returned nil but expected valid []string")
	}
	rl := len(r)
	if len(r) != 1 {
		t.Fatalf("returned %v but expected %v", rl, 1)
	}
	rv := r[0]
	if rv != "11.0.0.1" {
		t.Errorf("returned %v but expected %v", rv, "11.0.0.1")
	}
}

func TestSliceToStatus(t *testing.T) {
	fkEndpoints := []string{
		"10.0.0.1",
		"2001:db8::68",
		"opensource-k8s-ingress",
	}

	r := sliceToStatus(fkEndpoints)

	if r == nil {
		t.Fatalf("returned nil but expected a valid []apiv1.LoadBalancerIngress")
	}
	rl := len(r)
	if rl != 3 {
		t.Fatalf("returned %v but expected %v", rl, 3)
	}
	re1 := r[0]
	if re1.Hostname != "opensource-k8s-ingress" {
		t.Fatalf("returned %v but expected %v", re1, apiv1.LoadBalancerIngress{Hostname: "opensource-k8s-ingress"})
	}
	re2 := r[1]
	if re2.IP != "10.0.0.1" {
		t.Fatalf("returned %v but expected %v", re2, apiv1.LoadBalancerIngress{IP: "10.0.0.1"})
	}
	re3 := r[2]
	if re3.IP != "2001:db8::68" {
		t.Fatalf("returned %v but expected %v", re3, apiv1.LoadBalancerIngress{IP: "2001:db8::68"})
	}
}

func TestIngressSliceEqual(t *testing.T) {
	fk1 := buildLoadBalancerIngressByIP()
	fk2 := append(buildLoadBalancerIngressByIP(), apiv1.LoadBalancerIngress{
		IP:       "10.0.0.5",
		Hostname: "foo5",
	})
	fk3 := buildLoadBalancerIngressByIP()
	fk3[0].Hostname = "foo_no_01"
	fk4 := buildLoadBalancerIngressByIP()
	fk4[2].IP = "11.0.0.3"

	fooTests := []struct {
		lhs []apiv1.LoadBalancerIngress
		rhs []apiv1.LoadBalancerIngress
		er  bool
	}{
		{fk1, fk1, true},
		{fk2, fk1, false},
		{fk3, fk1, false},
		{fk4, fk1, false},
		{fk1, nil, false},
		{nil, nil, true},
		{[]apiv1.LoadBalancerIngress{}, []apiv1.LoadBalancerIngress{}, true},
	}

	for _, fooTest := range fooTests {
		r := ingressSliceEqual(fooTest.lhs, fooTest.rhs)
		if r != fooTest.er {
			t.Errorf("returned %v but expected %v", r, fooTest.er)
		}
	}
}
