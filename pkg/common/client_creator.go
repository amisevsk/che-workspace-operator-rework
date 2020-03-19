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
package common

import (
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func CreateClient() (crclient.Client, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	client, err := crclient.New(cfg, crclient.Options{})
	if err != nil {
		return nil, err
	}

	return client, nil
}
