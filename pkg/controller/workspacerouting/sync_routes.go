//
// Copyright (c) 2019-2020 Red Hat, Inc.
// This program and the accompanying materials are made
// available under the terms of the Eclipse Public License 2.0
// which is available at https://www.eclipse.org/legal/epl-2.0/
//
// SPDX-License-Identifier: EPL-2.0
//
// Contributors:
//   Red Hat, Inc. - initial API and implementation
//

package workspacerouting

import (
	"context"
	"fmt"
	"github.com/che-incubator/che-workspace-operator/pkg/apis/workspace/v1alpha1"
	"github.com/che-incubator/che-workspace-operator/pkg/config"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	routeV1 "github.com/openshift/api/route/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var routeDiffOpts = cmp.Options{
	cmpopts.IgnoreFields(routeV1.Route{}, "TypeMeta", "ObjectMeta", "Status"),
	cmpopts.IgnoreFields(routeV1.RouteSpec{}, "WildcardPolicy"),
	cmpopts.IgnoreFields(routeV1.RouteTargetReference{}, "Weight"),
}

func (r *ReconcileWorkspaceRouting) syncRoutes(routing *v1alpha1.WorkspaceRouting, specRoutes []routeV1.Route) (ok bool, err error) {
	routesInSync := true

	clusterRoutes, err := r.getClusterRoutes(routing)
	if err != nil {
		return false, err
	}

	toDelete := getRoutesToDelete(clusterRoutes, specRoutes)
	for _, route := range toDelete {
		err := r.client.Delete(context.TODO(), &route)
		if err != nil {
			return false, err
		}
		routesInSync = false
	}

	for _, specRoute := range specRoutes {
		if contains, idx := listContainsRouteByName(specRoute, clusterRoutes); contains {
			clusterRoute := clusterRoutes[idx]
			if !cmp.Equal(specRoute, clusterRoute, routeDiffOpts) {
				// Update route's spec
				clusterRoute.Spec = specRoute.Spec
				err := r.client.Update(context.TODO(), &clusterRoute)
				if err != nil && !errors.IsConflict(err) {
					return false, err
				}

				routesInSync = false
			}
		} else {
			err := r.client.Create(context.TODO(), &specRoute)
			if err != nil {
				return false, err
			}
			routesInSync = false
		}
	}

	return routesInSync, nil
}

func (r *ReconcileWorkspaceRouting) getClusterRoutes(routing *v1alpha1.WorkspaceRouting) ([]routeV1.Route, error) {
	found := &routeV1.RouteList{}
	labelSelector, err := labels.Parse(fmt.Sprintf("%s=%s", config.WorkspaceIDLabel, routing.Spec.WorkspaceId))
	if err != nil {
		return nil, err
	}
	listOptions := &client.ListOptions{
		Namespace:     routing.Namespace,
		LabelSelector: labelSelector,
	}
	err = r.client.List(context.TODO(), found, listOptions)
	if err != nil {
		return nil, err
	}

	var routes []routeV1.Route
	for _, route := range found.Items {
		for _, ownerref := range route.OwnerReferences {
			// We need to filter routes that are created automatically for ingresses on OpenShift
			if ownerref.Kind == "Ingress" {
				continue
			}
			routes = append(routes, route)
		}
	}
	return routes, nil
}

func getRoutesToDelete(clusterRoutes, specRoutes []routeV1.Route) []routeV1.Route {
	var toDelete []routeV1.Route
	for _, clusterRoute := range clusterRoutes {
		if contains, _ := listContainsRouteByName(clusterRoute, specRoutes); !contains {
			toDelete = append(toDelete, clusterRoute)
		}
	}
	return toDelete
}

func listContainsRouteByName(query routeV1.Route, list []routeV1.Route) (exists bool, idx int) {
	for idx, listRoute := range list {
		if query.Name == listRoute.Name {
			return true, idx
		}
	}
	return false, -1
}
