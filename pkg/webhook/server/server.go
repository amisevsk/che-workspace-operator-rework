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
package server

import (
	"io/ioutil"
	"os"

	"github.com/che-incubator/che-workspace-operator/internal/cluster"
	"github.com/che-incubator/che-workspace-operator/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("webhook.server")

// CABundle contains the contents of the ca cert
var CABundle []byte

// ConfigureWebhookServer sets up the webhook server if webhooks and certs are available
func ConfigureWebhookServer(mgr manager.Manager) (bool, error) {
	log.Info("Checking if webhook configuration is enabled")
	enabled, err := cluster.IsWebhookConfigurationEnabled()
	log.Info("Finished checking cluster")

	if err != nil {
		log.Info("ERROR: Could not evaluate if admission webhook configurations are available", "error", err)
		return false, err
	}

	if !enabled {
		log.Info("WARN: AdmissionWebhooks are not configured at your cluster." +
			"    To make your workspaces more secure, please configuring them." +
			"    Skipping setting up Webhook Server")
		return false, nil
	}

	log.Info("Attempting to read CA cert")
	CABundle, err = ioutil.ReadFile(config.WebhookServerCertDir + "/ca.crt")
	if os.IsNotExist(err) {
		log.Info("CA certificate is not found. Webhook server is not set up")
		return false, nil
	}
	if err != nil {
		log.Info("Recieved error when trying to read CA certificate")
		return false, err
	}

	log.Info("Setting up webhook server")
	mgr.GetWebhookServer().Port = config.WebhookServerPort
	mgr.GetWebhookServer().Host = config.WebhookServerHost
	mgr.GetWebhookServer().CertDir = config.WebhookServerCertDir

	return true, nil
}
