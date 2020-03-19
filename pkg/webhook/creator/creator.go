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
package creator

import (
	"context"

	"github.com/che-incubator/che-workspace-operator/pkg/common"
	"github.com/che-incubator/che-workspace-operator/pkg/config"
	"github.com/che-incubator/che-workspace-operator/pkg/controller/ownerref"
	"k8s.io/api/admissionregistration/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var log = logf.Log.WithName("webhook.creator")

//SetUp set up mutate webhook that manager creator annotations for workspaces
func SetUp(webhookServer *webhook.Server, ctx context.Context) error {
	log.Info("Configuring creator mutating webhook")
	client, err := common.CreateClient()
	if err != nil {
		return err
	}

	log.Info("Finished creating  the client")

	mutateWebhookCfg := buildMutateWebhookCfg()
	log.Info("Building the mutating config")

	ownRef, err := ownerref.FindControllerOwner(ctx, client)
	if err != nil {
		return err
	}

	log.Info("Got controller owner")

	//TODO For some reasons it's still possible to update reference by user
	//TODO Investigate if we can block it. The same issue is valid for Deployment owner
	mutateWebhookCfg.SetOwnerReferences([]metav1.OwnerReference{*ownRef})

	log.Info("set owner references")

	if err := client.Create(ctx, mutateWebhookCfg); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return err
		}
		// Webhook Configuration already exists, we want to update it
		// as we do not know if any fields might have changed.
		existingCfg := &v1beta1.MutatingWebhookConfiguration{}
		err := client.Get(ctx, types.NamespacedName{
			Name:      mutateWebhookCfg.Name,
			Namespace: mutateWebhookCfg.Namespace,
		}, existingCfg)

		mutateWebhookCfg.ResourceVersion = existingCfg.ResourceVersion
		err = client.Update(ctx, mutateWebhookCfg)
		if err != nil {
			return err
		}
		log.Info("Updated creator mutating webhook configuration")
	} else {
		log.Info("Created creator mutating webhook configuration")
	}

	webhookServer.Register(config.MutateWebhookPath, &webhook.Admission{Handler: &WorkspaceAnnotator{}})
	return nil
}
