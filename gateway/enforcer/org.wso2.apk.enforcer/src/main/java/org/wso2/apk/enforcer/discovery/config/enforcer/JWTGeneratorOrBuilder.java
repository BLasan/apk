// Generated by the protocol buffer compiler.  DO NOT EDIT!
// source: wso2/discovery/config/enforcer/jwt_generator.proto

package org.wso2.apk.enforcer.discovery.config.enforcer;

public interface JWTGeneratorOrBuilder extends
    // @@protoc_insertion_point(interface_extends:wso2.discovery.config.enforcer.JWTGenerator)
    com.google.protobuf.MessageOrBuilder {

  /**
   * <code>string public_certificate_path = 1;</code>
   * @return The publicCertificatePath.
   */
  java.lang.String getPublicCertificatePath();
  /**
   * <code>string public_certificate_path = 1;</code>
   * @return The bytes for publicCertificatePath.
   */
  com.google.protobuf.ByteString
      getPublicCertificatePathBytes();

  /**
   * <code>string private_key_path = 2;</code>
   * @return The privateKeyPath.
   */
  java.lang.String getPrivateKeyPath();
  /**
   * <code>string private_key_path = 2;</code>
   * @return The bytes for privateKeyPath.
   */
  com.google.protobuf.ByteString
      getPrivateKeyPathBytes();
}
