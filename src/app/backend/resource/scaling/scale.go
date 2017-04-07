// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package scaling

import (
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client "k8s.io/client-go/kubernetes"
	"strconv"
	"strings"
	"fmt"
)

// ReplicaCounts provide the desired and actual number of replicas.
type ReplicaCounts struct {
	DesiredReplicas int32 `json:"desiredReplicas"`
	ActualReplicas  int32 `json:"actualReplicas"`
}

// GetScaleSpec returns a populated ReplicaCounts object with desired and actual number of replicas.
func GetScaleSpec(client client.Interface, kind, namespace, name string) (rc *ReplicaCounts, err error) {
	rc = new(ReplicaCounts)
	s, err := client.Extensions().Scales(namespace).Get(kind, name)
	if err != nil {
		return nil, err
	}
	rc.DesiredReplicas = s.Spec.Replicas
	rc.ActualReplicas = s.Status.Replicas

	return
}

// ScaleResource scales the provided resource using the client scale method in the case of Deployment,
// ReplicaSet, Replication Controller. In the case of a job we are using the jobs resource update
// method since the client scale method does not provide one for the job.
func ScaleResource(client client.Interface, kind, namespace, name, count string) (rc *ReplicaCounts, err error) {
	rc = new(ReplicaCounts)
	if strings.ToLower(kind) == "job" {
		err = scaleJobResource(client, namespace, name, count, rc)
	} else {
		err = scaleGenericResource(client, kind, namespace, name, count, rc)
	}
	if err != nil {
		return nil, err
	}

	return
}

//ScaleGenericResource is used for Deployment, ReplicaSet, Replication Controller scaling.
func scaleGenericResource(client client.Interface, kind, namespace, name, count string, rc *ReplicaCounts) error {
	s, err := client.Extensions().Scales(namespace).Get(kind, name)
	if err != nil {
		return err
	}
	c, err := strconv.Atoi(count)
	if err != nil {
		return fmt.Errorf("Cannot parse scale by value: %s", count)
	}
	s.Spec.Replicas = int32(c)
	s, err = client.Extensions().Scales(namespace).Update(kind, s)
	if err != nil {
		return err
	}
	rc.DesiredReplicas = s.Spec.Replicas
	rc.ActualReplicas = s.Status.Replicas

	return nil
}

// scaleJobResource is exclusively used for jobs as it does not increase/decrease pods but jobs parallelism attribute.
func scaleJobResource(client client.Interface, namespace, name, count string, rc *ReplicaCounts) error {
	j, err := client.BatchV1().Jobs(namespace).Get(name, metaV1.GetOptions{})
	c, err := strconv.Atoi(count)
	if err != nil {
		return err
	}
	*j.Spec.Parallelism = int32(c)
	j, err = client.BatchV1().Jobs(namespace).Update(j)

	rc.DesiredReplicas = *j.Spec.Parallelism
	rc.ActualReplicas = *j.Spec.Parallelism

	return nil
}
