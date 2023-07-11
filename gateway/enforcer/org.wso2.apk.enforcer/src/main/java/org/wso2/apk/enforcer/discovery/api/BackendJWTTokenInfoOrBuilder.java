// Generated by the protocol buffer compiler.  DO NOT EDIT!
// source: wso2/discovery/api/BackendJWTTokenInfo.proto

package org.wso2.apk.enforcer.discovery.api;

public interface BackendJWTTokenInfoOrBuilder extends
    // @@protoc_insertion_point(interface_extends:wso2.discovery.api.BackendJWTTokenInfo)
    com.google.protobuf.MessageOrBuilder {

  /**
   * <code>bool enabled = 1;</code>
   * @return The enabled.
   */
  boolean getEnabled();

  /**
   * <code>string encoding = 2;</code>
   * @return The encoding.
   */
  java.lang.String getEncoding();
  /**
   * <code>string encoding = 2;</code>
   * @return The bytes for encoding.
   */
  com.google.protobuf.ByteString
      getEncodingBytes();

  /**
   * <code>string header = 3;</code>
   * @return The header.
   */
  java.lang.String getHeader();
  /**
   * <code>string header = 3;</code>
   * @return The bytes for header.
   */
  com.google.protobuf.ByteString
      getHeaderBytes();

  /**
   * <code>string signingAlgorithm = 4;</code>
   * @return The signingAlgorithm.
   */
  java.lang.String getSigningAlgorithm();
  /**
   * <code>string signingAlgorithm = 4;</code>
   * @return The bytes for signingAlgorithm.
   */
  com.google.protobuf.ByteString
      getSigningAlgorithmBytes();

  /**
   * <code>map&lt;string, .wso2.discovery.api.Claim&gt; customClaims = 5;</code>
   */
  int getCustomClaimsCount();
  /**
   * <code>map&lt;string, .wso2.discovery.api.Claim&gt; customClaims = 5;</code>
   */
  boolean containsCustomClaims(
      java.lang.String key);
  /**
   * Use {@link #getCustomClaimsMap()} instead.
   */
  @java.lang.Deprecated
  java.util.Map<java.lang.String, org.wso2.apk.enforcer.discovery.api.Claim>
  getCustomClaims();
  /**
   * <code>map&lt;string, .wso2.discovery.api.Claim&gt; customClaims = 5;</code>
   */
  java.util.Map<java.lang.String, org.wso2.apk.enforcer.discovery.api.Claim>
  getCustomClaimsMap();
  /**
   * <code>map&lt;string, .wso2.discovery.api.Claim&gt; customClaims = 5;</code>
   */

  org.wso2.apk.enforcer.discovery.api.Claim getCustomClaimsOrDefault(
      java.lang.String key,
      org.wso2.apk.enforcer.discovery.api.Claim defaultValue);
  /**
   * <code>map&lt;string, .wso2.discovery.api.Claim&gt; customClaims = 5;</code>
   */

  org.wso2.apk.enforcer.discovery.api.Claim getCustomClaimsOrThrow(
      java.lang.String key);

  /**
   * <code>int32 tokenTTL = 6;</code>
   * @return The tokenTTL.
   */
  int getTokenTTL();
}
