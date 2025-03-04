// Copyright (c) 2021, WSO2 LLC. (http://www.wso2.org) All Rights Reserved.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

// Package sotw provides an implementation of GRPC SoTW (State of The World) part of XDS server
package sotw

import (
	"context"
	"errors"
	"strconv"
	"sync/atomic"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/sotw/v3"
	streamv3 "github.com/envoyproxy/go-control-plane/pkg/server/stream/v3"
	"github.com/wso2/apk/adapter/pkg/discovery/protocol/resource/v3"
)

// NewServer creates handlers from a config watcher and callbacks.
func NewServer(ctx context.Context, config cache.ConfigWatcher, callbacks sotw.Callbacks) sotw.Server {
	return &server{cache: config, callbacks: callbacks, ctx: ctx}
}

type server struct {
	cache     cache.ConfigWatcher
	callbacks sotw.Callbacks
	ctx       context.Context

	// streamCount for counting bi-di streams
	streamCount int64
}

// watches for all xDS resource types
type watches struct {
	configs                   chan cache.Response
	apis                      chan cache.Response
	subscriptionList          chan cache.Response
	applicationList           chan cache.Response
	apiList                   chan cache.Response
	applicationPolicyList     chan cache.Response
	jwtIssuerList             chan cache.Response
	subscriptionPolicyList    chan cache.Response
	applicationKeyMappingList chan cache.Response
	keyManagers               chan cache.Response
	revokedTokens             chan cache.Response
	throttleData              chan cache.Response
	APKMgtApplications        chan cache.Response

	configCancel                    func()
	apiCancel                       func()
	subscriptionListCancel          func()
	applicationListCancel           func()
	jwtIssuerListCancel             func()
	apiListCancel                   func()
	applicationPolicyListCancel     func()
	subscriptionPolicyListCancel    func()
	applicationKeyMappingListCancel func()
	keyManagerCancel                func()
	revokedTokenCancel              func()
	throttleDataCancel              func()
	APKMgtApplicationCancel         func()

	configNonce                    string
	apiNonce                       string
	subscriptionListNonce          string
	applicationListNonce           string
	jwtIssuerListNonce             string
	apiListNonce                   string
	applicationPolicyListNonce     string
	subscriptionPolicyListNonce    string
	applicationKeyMappingListNonce string
	keyManagerNonce                string
	revokedTokenNonce              string
	throttleDataNonce              string
	APKMgtApplicationNonce         string

	// Opaque resources share a muxed channel. Nonces and watch cancellations are indexed by type URL.
	responses     chan cache.Response
	cancellations map[string]func()
	nonces        map[string]string
}

// Discovery response that is sent over GRPC stream
// We need to record what resource names are already sent to a client
// So if the client requests a new name we can respond back
// regardless current snapshot version (even if it is not changed yet)
type lastDiscoveryResponse struct {
	nonce     string
	resources map[string]struct{}
}

// Initialize all watches
func (values *watches) Init() {
	// muxed channel needs a buffer to release go-routines populating it
	values.responses = make(chan cache.Response, 11)
	values.cancellations = make(map[string]func())
	values.nonces = make(map[string]string)
}

// Token response value used to signal a watch failure in muxed watches.
var errorResponse = &cache.RawResponse{}

// Cancel all watches
func (values *watches) Cancel() {
	if values.configCancel != nil {
		values.configCancel()
	}
	if values.apiCancel != nil {
		values.apiCancel()
	}
	if values.subscriptionListCancel != nil {
		values.subscriptionListCancel()
	}
	if values.applicationListCancel != nil {
		values.applicationListCancel()
	}
	if values.jwtIssuerListCancel != nil {
		values.jwtIssuerListCancel()
	}
	if values.apiListCancel != nil {
		values.apiListCancel()
	}
	if values.applicationPolicyListCancel != nil {
		values.applicationPolicyListCancel()
	}
	if values.subscriptionPolicyListCancel != nil {
		values.subscriptionPolicyListCancel()
	}
	if values.applicationKeyMappingListCancel != nil {
		values.applicationKeyMappingListCancel()
	}
	if values.keyManagerCancel != nil {
		values.keyManagerCancel()
	}
	if values.revokedTokenCancel != nil {
		values.revokedTokenCancel()
	}
	if values.throttleDataCancel != nil {
		values.throttleDataCancel()
	}
	if values.APKMgtApplicationCancel != nil {
		values.APKMgtApplicationCancel()
	}

	for _, cancel := range values.cancellations {
		if cancel != nil {
			cancel()
		}
	}
}

// process handles a bi-di stream request
func (s *server) process(stream streamv3.Stream, reqCh <-chan *discovery.DiscoveryRequest, defaultTypeURL string) error {
	// increment stream count
	streamID := atomic.AddInt64(&s.streamCount, 1)

	// unique nonce generator for req-resp pairs per xDS stream; the server
	// ignores stale nonces. nonce is only modified within send() function.
	var streamNonce int64

	streamState := streamv3.NewStreamState(false, map[string]string{})
	lastDiscoveryResponses := map[string]lastDiscoveryResponse{}

	var node = &core.Node{}

	// a collection of stack allocated watches per request type
	var values watches
	values.Init()
	defer func() {
		values.Cancel()
		if s.callbacks != nil {
			s.callbacks.OnStreamClosed(streamID, node)
		}
	}()

	// sends a response by serializing to protobuf Any
	send := func(resp cache.Response) (string, error) {
		if resp == nil {
			return "", errors.New("missing response")
		}

		out, err := resp.GetDiscoveryResponse()
		if err != nil {
			return "", err
		}

		// increment nonce
		streamNonce = streamNonce + 1
		out.Nonce = strconv.FormatInt(streamNonce, 10)

		lastResponse := lastDiscoveryResponse{
			nonce:     out.Nonce,
			resources: make(map[string]struct{}),
		}
		for _, r := range resp.GetRequest().ResourceNames {
			lastResponse.resources[r] = struct{}{}
		}
		lastDiscoveryResponses[resp.GetRequest().TypeUrl] = lastResponse

		if s.callbacks != nil {
			s.callbacks.OnStreamResponse(resp.GetContext(), streamID, resp.GetRequest(), out)
		}
		return out.Nonce, stream.Send(out)
	}

	if s.callbacks != nil {
		if err := s.callbacks.OnStreamOpen(stream.Context(), streamID, defaultTypeURL); err != nil {
			return err
		}
	}

	for {
		select {
		case <-s.ctx.Done():
			return nil
			// config watcher can send the requested resources types in any order
		case resp, more := <-values.configs:
			if !more {
				return status.Errorf(codes.Unavailable, "configs watch failed")
			}
			nonce, err := send(resp)
			if err != nil {
				return err
			}
			values.configNonce = nonce

		case resp, more := <-values.apis:
			if !more {
				return status.Errorf(codes.Unavailable, "apis watch failed")
			}
			nonce, err := send(resp)
			if err != nil {
				return err
			}
			values.apiNonce = nonce

		case resp, more := <-values.subscriptionList:
			if !more {
				return status.Errorf(codes.Unavailable, "subscriptionList watch failed")
			}
			nonce, err := send(resp)
			if err != nil {
				return err
			}
			values.subscriptionListNonce = nonce

		case resp, more := <-values.apiList:
			if !more {
				return status.Errorf(codes.Unavailable, "apiList watch failed")
			}
			nonce, err := send(resp)
			if err != nil {
				return err
			}
			values.apiListNonce = nonce

		case resp, more := <-values.applicationList:
			if !more {
				return status.Errorf(codes.Unavailable, "applicationList watch failed")
			}
			nonce, err := send(resp)
			if err != nil {
				return err
			}
			values.applicationListNonce = nonce

		case resp, more := <-values.jwtIssuerList:
			if !more {
				return status.Errorf(codes.Unavailable, "jwtIssuerList watch failed")
			}
			nonce, err := send(resp)
			if err != nil {
				return err
			}
			values.jwtIssuerListNonce = nonce

		case resp, more := <-values.applicationPolicyList:
			if !more {
				return status.Errorf(codes.Unavailable, "applicationPolicyList watch failed")
			}
			nonce, err := send(resp)
			if err != nil {
				return err
			}
			values.applicationPolicyListNonce = nonce

		case resp, more := <-values.subscriptionPolicyList:
			if !more {
				return status.Errorf(codes.Unavailable, "subscriptionPolicyList watch failed")
			}
			nonce, err := send(resp)
			if err != nil {
				return err
			}
			values.subscriptionPolicyListNonce = nonce

		case resp, more := <-values.applicationKeyMappingList:
			if !more {
				return status.Errorf(codes.Unavailable, "applicationKeyMappingList watch failed")
			}
			nonce, err := send(resp)
			if err != nil {
				return err
			}
			values.applicationKeyMappingListNonce = nonce

		case resp, more := <-values.keyManagers:
			if !more {
				return status.Errorf(codes.Unavailable, "keyManagers watch failed")
			}
			nonce, err := send(resp)
			if err != nil {
				return err
			}
			values.keyManagerNonce = nonce

		case resp, more := <-values.revokedTokens:
			if !more {
				return status.Errorf(codes.Unavailable, "revoked tokens watch failed")
			}
			nonce, err := send(resp)
			if err != nil {
				return err
			}
			values.revokedTokenNonce = nonce

		case resp, more := <-values.throttleData:
			if !more {
				return status.Errorf(codes.Unavailable, "throttle data watch failed")
			}
			nonce, err := send(resp)
			if err != nil {
				return err
			}
			values.throttleDataNonce = nonce
		case resp, more := <-values.APKMgtApplications:
			if !more {
				return status.Errorf(codes.Unavailable, "global adapter apis watch failed")
			}
			nonce, err := send(resp)
			if err != nil {
				return err
			}
			values.APKMgtApplicationNonce = nonce
		case resp, more := <-values.responses:
			if more {
				if resp == errorResponse {
					return status.Errorf(codes.Unavailable, "resource watch failed")
				}
				typeURL := resp.GetRequest().TypeUrl
				nonce, err := send(resp)
				if err != nil {
					return err
				}
				values.nonces[typeURL] = nonce
			}

		case req, more := <-reqCh:
			// input stream ended or errored out
			if !more {
				return nil
			}
			if req == nil {
				return status.Errorf(codes.Unavailable, "empty request")
			}

			// node field in discovery request is delta-compressed
			if req.Node != nil {
				node = req.Node
			} else {
				req.Node = node
			}

			// nonces can be reused across streams; we verify nonce only if nonce is not initialized
			nonce := req.GetResponseNonce()

			// type URL is required for ADS but is implicit for xDS
			if defaultTypeURL == resource.AnyType {
				if req.TypeUrl == "" {
					return status.Errorf(codes.InvalidArgument, "type URL is required for ADS")
				}
			} else if req.TypeUrl == "" {
				req.TypeUrl = defaultTypeURL
			}

			if s.callbacks != nil {
				if err := s.callbacks.OnStreamRequest(streamID, req); err != nil {
					return err
				}
			}

			if lastResponse, ok := lastDiscoveryResponses[req.TypeUrl]; ok {
				if lastResponse.nonce == "" || lastResponse.nonce == nonce {
					// Let's record Resource names that a client has received.
					streamState.SetKnownResourceNames(req.TypeUrl, lastResponse.resources)
				}
			}

			// cancel existing watches to (re-)request a newer version
			switch {
			case req.TypeUrl == resource.ConfigType:
				if values.configNonce == "" || values.configNonce == nonce {
					if values.configCancel != nil {
						values.configCancel()
					}
					values.configs = make(chan cache.Response, 1)
					values.configCancel = s.cache.CreateWatch(req, streamState, values.configs)
				}
			case req.TypeUrl == resource.APIType:
				if values.apiNonce == "" || values.apiNonce == nonce {
					if values.apiCancel != nil {
						values.apiCancel()
					}
					values.apis = make(chan cache.Response, 1)
					values.apiCancel = s.cache.CreateWatch(req, streamState, values.apis)
				}
			case req.TypeUrl == resource.SubscriptionListType:
				if values.subscriptionListNonce == "" || values.subscriptionListNonce == nonce {
					if values.subscriptionListCancel != nil {
						values.subscriptionListCancel()
					}
					values.subscriptionList = make(chan cache.Response, 1)
					values.subscriptionListCancel = s.cache.CreateWatch(req, streamState, values.subscriptionList)
				}
			case req.TypeUrl == resource.APIListType:
				if values.apiListNonce == "" || values.apiListNonce == nonce {
					if values.apiListCancel != nil {
						values.apiListCancel()
					}
					values.apiList = make(chan cache.Response, 1)
					values.apiListCancel = s.cache.CreateWatch(req, streamState, values.apiList)
				}
			case req.TypeUrl == resource.ApplicationListType:
				if values.applicationListNonce == "" || values.applicationListNonce == nonce {
					if values.applicationListCancel != nil {
						values.applicationListCancel()
					}
					values.applicationList = make(chan cache.Response, 1)
					values.applicationListCancel = s.cache.CreateWatch(req, streamState, values.applicationList)
				}
			case req.TypeUrl == resource.JWTIssuerListType:
				if values.jwtIssuerListNonce == "" || values.jwtIssuerListNonce == nonce {
					if values.jwtIssuerListCancel != nil {
						values.jwtIssuerListCancel()
					}
					values.jwtIssuerList = make(chan cache.Response, 1)
					values.jwtIssuerListCancel = s.cache.CreateWatch(req, streamState, values.jwtIssuerList)
				}
			case req.TypeUrl == resource.ApplicationPolicyListType:
				if values.applicationPolicyListNonce == "" || values.applicationPolicyListNonce == nonce {
					if values.applicationPolicyListCancel != nil {
						values.applicationPolicyListCancel()
					}
					values.applicationPolicyList = make(chan cache.Response, 1)
					values.applicationPolicyListCancel = s.cache.CreateWatch(req, streamState, values.applicationPolicyList)
				}

			case req.TypeUrl == resource.SubscriptionPolicyListType:
				if values.subscriptionPolicyListNonce == "" || values.subscriptionPolicyListNonce == nonce {
					if values.subscriptionPolicyListCancel != nil {
						values.subscriptionPolicyListCancel()
					}
					values.subscriptionPolicyList = make(chan cache.Response, 1)
					values.subscriptionPolicyListCancel = s.cache.CreateWatch(req, streamState, values.subscriptionPolicyList)
				}
			case req.TypeUrl == resource.ApplicationKeyMappingListType:
				if values.applicationKeyMappingListNonce == "" || values.applicationKeyMappingListNonce == nonce {
					if values.applicationKeyMappingListCancel != nil {
						values.applicationKeyMappingListCancel()
					}
					values.applicationKeyMappingList = make(chan cache.Response, 1)
					values.applicationKeyMappingListCancel = s.cache.CreateWatch(req, streamState, values.applicationKeyMappingList)
				}
			case req.TypeUrl == resource.KeyManagerType:
				if values.keyManagerNonce == "" || values.keyManagerNonce == nonce {
					if values.keyManagerCancel != nil {
						values.keyManagerCancel()
					}
					values.keyManagers = make(chan cache.Response, 1)
					values.keyManagerCancel = s.cache.CreateWatch(req, streamState, values.keyManagers)
				}
			case req.TypeUrl == resource.RevokedTokensType:
				if values.revokedTokenNonce == "" || values.revokedTokenNonce == nonce {
					if values.revokedTokenCancel != nil {
						values.revokedTokenCancel()
					}
					values.revokedTokens = make(chan cache.Response, 1)
					values.revokedTokenCancel = s.cache.CreateWatch(req, streamState, values.revokedTokens)
				}
			case req.TypeUrl == resource.ThrottleDataType:
				if values.throttleDataNonce == "" || values.throttleDataNonce == nonce {
					if values.throttleDataCancel != nil {
						values.throttleDataCancel()
					}
					values.throttleData = make(chan cache.Response, 1)
					values.throttleDataCancel = s.cache.CreateWatch(req, streamState, values.throttleData)
				}
			case req.TypeUrl == resource.APKMgtApplicationType:
				if values.APKMgtApplicationNonce == "" || values.APKMgtApplicationNonce == nonce {
					if values.APKMgtApplicationCancel != nil {
						values.APKMgtApplicationCancel()
					}
					values.APKMgtApplications = make(chan cache.Response, 1)
					values.APKMgtApplicationCancel = s.cache.CreateWatch(req, streamState, values.APKMgtApplications)
				}
			default:
				typeURL := req.TypeUrl
				responseNonce, seen := values.nonces[typeURL]
				if !seen || responseNonce == nonce {

					if cancel, seen := values.cancellations[typeURL]; seen && cancel != nil {
						cancel()
					}
					values.cancellations[typeURL] = s.cache.CreateWatch(req, streamState, values.responses)
				}
			}
		}
	}
}

// StreamHandler converts a blocking read call to channels and initiates stream processing
func (s *server) StreamHandler(stream streamv3.Stream, typeURL string) error {
	// a channel for receiving incoming requests
	reqCh := make(chan *discovery.DiscoveryRequest)
	go func() {
		defer close(reqCh)
		for {
			req, err := stream.Recv()
			if err != nil {
				return
			}
			select {
			case reqCh <- req:
			case <-stream.Context().Done():
				return
			case <-s.ctx.Done():
				return
			}
		}
	}()

	return s.process(stream, reqCh, typeURL)
}
