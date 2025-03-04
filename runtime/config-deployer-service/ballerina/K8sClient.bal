//
// Copyright (c) 2022, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
//

import ballerina/io;
import ballerina/url;
import ballerina/http;
import wso2/apk_common_lib as commons;
import ballerina/crypto;
import config_deployer_service.model as model;

const string K8S_API_ENDPOINT = "/api/v1";
final string token = check io:fileReadString(k8sConfiguration.serviceAccountPath + "/token");
final string caCertPath = k8sConfiguration.serviceAccountPath + "/ca.crt";
string namespaceFile = k8sConfiguration.serviceAccountPath + "/namespace";
final string currentNameSpace = check io:fileReadString(namespaceFile);
final http:Client k8sApiServerEp = check initializeK8sClient();

# This initialize the k8s Client.
# + return - k8s http client
public function initializeK8sClient() returns http:Client|error {
    http:Client k8sApiClient = check new ("https://" + k8sConfiguration.host,
        auth = {
            token: token
        },
        secureSocket = {
            cert: caCertPath

        }
    );
    return k8sApiClient;
}

# This returns ConfigMap value according to name and namespace.
#
# + name - Name of ConfigMap  
# + namespace - Namespace of Configmap
# + return - Return configmap value for name and namespace
isolated function getConfigMapValueFromNameAndNamespace(string name, string namespace) returns http:Response|error {
    string endpoint = "/api/v1/namespaces/" + namespace + "/configmaps/" + name;
    return k8sApiServerEp->get(endpoint, targetType = http:Response);
}

isolated function deleteAPICR(string name, string namespace) returns http:Response|http:ClientError {
    string endpoint = "/apis/dp.wso2.com/v1alpha1/namespaces/" + namespace + "/apis/" + name;
    return k8sApiServerEp->delete(endpoint, targetType = http:Response);
}

isolated function deleteAuthenticationCR(string name, string namespace) returns http:Response|http:ClientError {
    string endpoint = "/apis/dp.wso2.com/v1alpha1/namespaces/" + namespace + "/authentications/" + name;
    return k8sApiServerEp->delete(endpoint, targetType = http:Response);
}

isolated function deployAuthenticationCR(model:Authentication authentication, string namespace) returns http:Response|http:ClientError {
    string endpoint = "/apis/dp.wso2.com/v1alpha1/namespaces/" + namespace + "/authentications";
    return k8sApiServerEp->post(endpoint, authentication, targetType = http:Response);
}

isolated function getHttpRoute(string name, string namespace) returns model:Httproute|http:ClientError {
    string endpoint = "/apis/gateway.networking.k8s.io/v1beta1/namespaces/" + namespace + "/httproutes/" + name;
    return k8sApiServerEp->get(endpoint, targetType = model:Httproute);
}

isolated function deleteHttpRoute(string name, string namespace) returns http:Response|http:ClientError {
    string endpoint = "/apis/gateway.networking.k8s.io/v1beta1/namespaces/" + namespace + "/httproutes/" + name;
    return k8sApiServerEp->delete(endpoint, targetType = http:Response);
}

isolated function deleteConfigMap(string name, string namespace) returns http:Response|http:ClientError {
    string endpoint = "/api/v1/namespaces/" + namespace + "/configmaps/" + name;
    return k8sApiServerEp->delete(endpoint, targetType = http:Response);
}

isolated function deployAPICR(model:API api, string namespace) returns http:Response|http:ClientError {
    string endpoint = "/apis/dp.wso2.com/v1alpha1/namespaces/" + namespace + "/apis";
    return k8sApiServerEp->post(endpoint, api, targetType = http:Response);
}

isolated function updateAPICR(model:API api, string namespace) returns http:Response|http:ClientError {
    string endpoint = "/apis/dp.wso2.com/v1alpha1/namespaces/" + namespace + "/apis/" + api.metadata.name;
    return k8sApiServerEp->put(endpoint, api, targetType = http:Response);
}

isolated function deployConfigMap(model:ConfigMap configMap, string namespace) returns http:Response|http:ClientError {
    string endpoint = "/api/v1/namespaces/" + namespace + "/configmaps";
    return k8sApiServerEp->post(endpoint, configMap, targetType = http:Response);
}

isolated function updateConfigMap(model:ConfigMap configMap, string namespace) returns http:Response|http:ClientError {
    string endpoint = "/api/v1/namespaces/" + namespace + "/configmaps/" + configMap.metadata.name;
    return k8sApiServerEp->put(endpoint, configMap, targetType = http:Response);
}

isolated function deployHttpRoute(model:Httproute httproute, string namespace) returns http:Response|http:ClientError {
    string endpoint = "/apis/gateway.networking.k8s.io/v1beta1/namespaces/" + namespace + "/httproutes";
    return k8sApiServerEp->post(endpoint, httproute, targetType = http:Response);
}

public isolated function getK8sAPIByNameAndNamespace(string name, string namespace) returns model:API?|commons:APKError {
    string endpoint = "/apis/dp.wso2.com/v1alpha1/namespaces/" + namespace + "/apis/" + name;
    do {
        http:Response response = check k8sApiServerEp->get(endpoint);
        if response.statusCode == 200 {
            json jsonPayload = check response.getJsonPayload();
            return check jsonPayload.cloneWithType(model:API);
        } else if (response.statusCode == 404) {
            return ();
        } else {
            return error("Internal Error occured", message = "Internal Error occured", statusCode = 500, code = 909101, description = "Internal Error occured");
        }
    } on fail var e {
        return error("Internal Error occured", e, message = "Internal Error occured", statusCode = 500, code = 909101, description = "Internal Error occured");
    }
}

isolated function getAuthenticationCrsForAPI(string apiName, string apiVersion, string namespace, string organization) returns model:AuthenticationList|http:ClientError|error {
    string endpoint = "/apis/dp.wso2.com/v1alpha1/namespaces/" + namespace + "/authentications?labelSelector=" + check generateUrlEncodedLabelSelector(apiName, apiVersion, organization);
    return k8sApiServerEp->get(endpoint, targetType = model:AuthenticationList);
}

isolated function getScopeCrsForAPI(string apiName, string apiVersion, string namespace, string organization) returns model:ScopeList|http:ClientError|error {
    string endpoint = "/apis/dp.wso2.com/v1alpha1/namespaces/" + namespace + "/scopes?labelSelector=" + check generateUrlEncodedLabelSelector(apiName, apiVersion, organization);
    return k8sApiServerEp->get(endpoint, targetType = model:ScopeList);
}

isolated function deleteScopeCr(string name, string namespace) returns http:Response|http:ClientError {
    string endpoint = "/apis/dp.wso2.com/v1alpha1/namespaces/" + namespace + "/scopes/" + name;
    return k8sApiServerEp->delete(endpoint, targetType = http:Response);
}

isolated function deleteBackendPolicyCR(string name, string namespace) returns http:Response|http:ClientError {
    string endpoint = "/apis/dp.wso2.com/v1alpha1/namespaces/" + namespace + "/backends/" + name;
    return k8sApiServerEp->delete(endpoint, targetType = http:Response);
}

isolated function deployBackendCR(model:Backend backend, string namespace) returns http:Response|http:ClientError {
    string endpoint = "/apis/dp.wso2.com/v1alpha1/namespaces/" + namespace + "/backends";
    return k8sApiServerEp->post(endpoint, backend, targetType = http:Response);
}

isolated function deployScopeCR(model:Scope scope, string namespace) returns http:Response|http:ClientError {
    string endpoint = "/apis/dp.wso2.com/v1alpha1/namespaces/" + namespace + "/scopes";
    return k8sApiServerEp->post(endpoint, scope, targetType = http:Response);
}

isolated function getBackendPolicyCRsForAPI(string apiName, string apiVersion, string namespace, string organization) returns model:BackendList|http:ClientError|error {
    string endpoint = "/apis/dp.wso2.com/v1alpha1/namespaces/" + namespace + "/backends?labelSelector=" + check generateUrlEncodedLabelSelector(apiName, apiVersion, organization);
    return k8sApiServerEp->get(endpoint, targetType = model:BackendList);
}

isolated function generateUrlEncodedLabelSelector(string apiName, string apiVersion, string organization) returns string|error {
    string apiNameHash = crypto:hashSha1(apiName.toBytes()).toBase16();
    string apiVersionHash = crypto:hashSha1(apiVersion.toBytes()).toBase16();
    string organizationHash = crypto:hashSha1(organization.toBytes()).toBase16();
    string labelSelector = string:'join("", API_NAME_HASH_LABEL, "=", apiNameHash, ",", API_VERSION_HASH_LABEL, "=", apiVersionHash, ",", ORGANIZATION_HASH_LABEL, "=", organizationHash);
    return url:encode(labelSelector, "UTF-8");
}

isolated function getBackendServicesForAPI(string apiName, string apiVersion, string namespace, string organization) returns model:ServiceList|http:ClientError|error {
    string endpoint = "/api/v1/namespaces/" + namespace + "/services?labelSelector=" + check generateUrlEncodedLabelSelector(apiName, apiVersion, organization);
    return k8sApiServerEp->get(endpoint, targetType = model:ServiceList);
}

public isolated function getHttproutesForAPIS(string apiName, string apiVersion, string namespace, string organization) returns model:HttprouteList|http:ClientError|error {
    string endpoint = "/apis/gateway.networking.k8s.io/v1beta1/namespaces/" + namespace + "/httproutes/?labelSelector=" + check generateUrlEncodedLabelSelector(apiName, apiVersion, organization);
    return k8sApiServerEp->get(endpoint, targetType = model:HttprouteList);
}

isolated function deployRateLimitPolicyCR(model:RateLimitPolicy rateLimitPolicy, string namespace) returns http:Response|http:ClientError {
    string endpoint = "/apis/dp.wso2.com/v1alpha1/namespaces/" + namespace + "/ratelimitpolicies";
    return k8sApiServerEp->post(endpoint, rateLimitPolicy, targetType = http:Response);
}

isolated function deleteRateLimitPolicyCR(string name, string namespace) returns http:Response|http:ClientError {
    string endpoint = "/apis/dp.wso2.com/v1alpha1/namespaces/" + namespace + "/ratelimitpolicies/" + name;
    return k8sApiServerEp->delete(endpoint, targetType = http:Response);
}

isolated function getRateLimitPolicyCRsForAPI(string apiName, string apiVersion, string namespace, string organization) returns model:RateLimitPolicyList|http:ClientError|error {
    string endpoint = "/apis/dp.wso2.com/v1alpha1/namespaces/" + namespace + "/ratelimitpolicies?labelSelector=" + check generateUrlEncodedLabelSelector(apiName, apiVersion, organization);
    return k8sApiServerEp->get(endpoint, targetType = model:RateLimitPolicyList);
}

isolated function deployAPIPolicyCR(model:APIPolicy apiPolicy, string namespace) returns http:Response|http:ClientError {
    string endpoint = "/apis/dp.wso2.com/v1alpha1/namespaces/" + namespace + "/apipolicies";
    return k8sApiServerEp->post(endpoint, apiPolicy, targetType = http:Response);
}

isolated function deleteAPIPolicyCR(string name, string namespace) returns http:Response|http:ClientError {
    string endpoint = "/apis/dp.wso2.com/v1alpha1/namespaces/" + namespace + "/apipolicies/" + name;
    return k8sApiServerEp->delete(endpoint, targetType = http:Response);
}

isolated function getAPIPolicyCRsForAPI(string apiName, string apiVersion, string namespace, string organization) returns model:APIPolicyList|http:ClientError|error {
    string endpoint = "/apis/dp.wso2.com/v1alpha1/namespaces/" + namespace + "/apipolicies?labelSelector=" + check generateUrlEncodedLabelSelector(apiName, apiVersion, organization);
    return k8sApiServerEp->get(endpoint, targetType = model:APIPolicyList);
}

isolated function deployInterceptorServiceCR(model:InterceptorService interceptorService, string namespace) returns http:Response|http:ClientError {
    string endpoint = "/apis/dp.wso2.com/v1alpha1/namespaces/" + namespace + "/interceptorservices";
    return k8sApiServerEp->post(endpoint, interceptorService, targetType = http:Response);
}

isolated function deleteInterceptorServiceCR(string name, string namespace) returns http:Response|http:ClientError {
    string endpoint = "/apis/dp.wso2.com/v1alpha1/namespaces/" + namespace + "/interceptorservices/" + name;
    return k8sApiServerEp->delete(endpoint, targetType = http:Response);
}

isolated function getInterceptorServiceCRsForAPI(string apiName, string apiVersion, string namespace, string organization) returns model:InterceptorServiceList|http:ClientError|error {
    string endpoint = "/apis/dp.wso2.com/v1alpha1/namespaces/" + namespace + "/interceptorservices?labelSelector=" + check generateUrlEncodedLabelSelector(apiName, apiVersion, organization);
    return k8sApiServerEp->get(endpoint, targetType = model:InterceptorServiceList);
}
