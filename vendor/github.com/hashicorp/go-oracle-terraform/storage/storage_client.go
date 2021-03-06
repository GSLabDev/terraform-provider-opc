package storage

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-oracle-terraform/client"
	"github.com/hashicorp/go-oracle-terraform/opc"
)

const STR_ACCOUNT = "/Storage-%s"
const STR_USERNAME = "/Storage-%s:%s"
const AUTH_HEADER = "X-Auth-Token"
const STR_QUALIFIED_NAME = "%s%s/%s"
const API_VERSION = "v1"

// Client represents an authenticated compute client, with compute credentials and an api client.
type StorageClient struct {
	client      *client.Client
	authToken   *string
	tokenIssued time.Time
}

func NewStorageClient(c *opc.Config) (*StorageClient, error) {
	sClient := &StorageClient{}
	opcClient, err := client.NewClient(c)
	if err != nil {
		return nil, err
	}
	sClient.client = opcClient

	if err := sClient.getAuthenticationToken(); err != nil {
		return nil, err
	}

	return sClient, nil
}

// Execute a request with a nil body
func (c *StorageClient) executeRequest(method, path string, headers interface{}) (*http.Response, error) {
	return c.executeRequestBody(method, path, headers, nil)
}

// Execute a request with a body supplied. The body can be nil for the request.
// Does not marshal the body into json to create the request
func (c *StorageClient) executeRequestBody(method, path string, headers interface{}, body io.ReadSeeker) (*http.Response, error) {
	req, err := c.client.BuildNonJSONRequest(method, path, body)
	if err != nil {
		return nil, err
	}

	debugReqString := fmt.Sprintf("%s (%s) %s", req.Method, req.URL, req.Proto)
	var debugHeaders []string

	if headers != nil {
		for k, v := range headers.(map[string]string) {
			debugHeaders = append(debugHeaders,
				fmt.Sprintf("%v: %v\n", strings.ToLower(k), v))

			req.Header.Add(k, v)
		}
		debugReqString = fmt.Sprintf("%s\n%s", debugReqString, debugHeaders)
	}

	if !strings.Contains(path, "/auth/") {
		c.client.DebugLogString(debugReqString)
	}

	// If we have an authentication token, let's authenticate, refreshing cookie if need be
	if c.authToken != nil {
		if time.Since(c.tokenIssued).Minutes() > 25 {
			if err := c.getAuthenticationToken(); err != nil {
				return nil, err
			}
		}
		req.Header.Add(AUTH_HEADER, *c.authToken)
	}

	resp, err := c.client.ExecuteRequest(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *StorageClient) getUserName() string {
	return fmt.Sprintf(STR_USERNAME, *c.client.IdentityDomain, *c.client.UserName)
}

func (c *StorageClient) getAccount() string {
	return fmt.Sprintf(STR_ACCOUNT, *c.client.IdentityDomain)
}

// GetQualifiedName returns the fully-qualified name of a storage object, e.g. /v1/{account}/{name}
func (c *StorageClient) getQualifiedName(name string) string {
	if name == "" {
		return ""
	}
	if strings.HasPrefix(name, "/Storage-") || strings.HasPrefix(name, API_VERSION+"/") {
		return name
	}
	return fmt.Sprintf(STR_QUALIFIED_NAME, API_VERSION, c.getAccount(), name)
}

// GetUnqualifiedName returns the unqualified name of a Storage object, e.g. the {name} part of /v1/{account}/{name}
func (c *StorageClient) getUnqualifiedName(name string) string {
	if name == "" {
		return name
	}
	if !strings.Contains(name, "/") {
		return name
	}

	nameParts := strings.Split(name, "/")
	return strings.Join(nameParts[len(nameParts)-1:], "/")
}

func (c *StorageClient) unqualify(names ...*string) {
	for _, name := range names {
		*name = c.getUnqualifiedName(*name)
	}
}
