/*
Copyright 2020 GramLabs, Inc.

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

package secrets

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

// SecretManagerLoader loads secret data from the GCP Secret Manager.
type SecretManagerLoader struct {
	client *secretmanager.Client
}

// NewSecretManagerLoader creates a new Secret Manager loader.
func NewSecretManagerLoader(ctx context.Context) (*SecretManagerLoader, error) {
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &SecretManagerLoader{
		client: client,
	}, nil
}

// Load the secret identified by the supplied URL.
func (l *SecretManagerLoader) Load(ctx context.Context, u url.URL) ([]byte, error) {
	if l == nil || l.client == nil {
		return nil, fmt.Errorf("unavailable")
	}

	name := []string{u.Host, strings.TrimPrefix(u.Path, "/"), u.Fragment}
	if name[0] == "" {
		name[0], name[1] = split(name[1], '/')
	}
	if name[2] == "" {
		name[2] = "latest"
	}

	if u.Scheme != "sm" || strings.Contains(name[1], "/") {
		return nil, fmt.Errorf("invalid secret manager reference %q", u.String())
	}

	accessRequest := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/%s", name[0], name[1], name[2]),
	}
	result, err := l.client.AccessSecretVersion(ctx, accessRequest)
	if err != nil {
		return nil, err
	}
	return result.Payload.Data, nil
}
