# ProGuard/R8 rules for Edge Mobile
# Release build uses protobuf-lite + grpc-okhttp (client) and grpc-netty-shaded + grpc-services (server).
# Many optional/full protobuf classes are referenced but not on classpath; suppress R8 missing-class errors.

# ========== Missing classes (optional deps / full protobuf) – don't fail R8 ==========
-dontwarn com.google.protobuf.**
-dontwarn com.aayushatharva.brotli4j.**
-dontwarn com.github.luben.zstd.**
-dontwarn com.jcraft.jzlib.**
-dontwarn com.ning.compress.**
-dontwarn com.oracle.svm.**
-dontwarn com.squareup.okhttp.CipherSuite
-dontwarn com.squareup.okhttp.ConnectionSpec
-dontwarn com.squareup.okhttp.TlsVersion
-dontwarn javax.naming.**
-dontwarn lzma.sdk.**
-dontwarn net.jpountz.lz4.**
-dontwarn net.jpountz.xxhash.**
-dontwarn org.apache.log4j.**
-dontwarn org.apache.logging.log4j.**
-dontwarn org.eclipse.jetty.alpn.**
-dontwarn org.eclipse.jetty.npn.**
-dontwarn org.jboss.marshalling.**
-dontwarn org.slf4j.**
-dontwarn reactor.blockhound.**
-dontwarn sun.security.x509.**
-dontwarn com.google.common.**
-dontwarn com.google.errorprone.annotations.**
-dontwarn sun.misc.Unsafe

# ========== gRPC ==========
# Keep gRPC generated classes (our proto stubs) and gRPC runtime
-keep class io.grpc.** { *; }
-keep class com.example.edge_mobile.grpc.** { *; }

# Keep gRPC service implementations
-keepclassmembers class * extends io.grpc.stub.AbstractStub {
    <init>(...);
}

# gRPC uses reflection for service binding
-keepclassmembers class * {
    @io.grpc.stub.annotations.RpcMethod *;
}

# Keep protobuf lite classes
-keep class * extends com.google.protobuf.GeneratedMessageLite { *; }
-keep class * extends com.google.protobuf.GeneratedMessageLite$Builder { *; }

# ========== OkHttp ==========
-dontwarn okhttp3.**
-dontwarn okio.**
-keep class okhttp3.** { *; }
-keep interface okhttp3.** { *; }
-keepattributes Signature
-keepattributes *Annotation*

# ========== Kotlin Coroutines ==========
-keepnames class kotlinx.coroutines.internal.MainDispatcherFactory {}
-keepnames class kotlinx.coroutines.CoroutineExceptionHandler {}
-keepclassmembernames class kotlinx.** {
    volatile <fields>;
}

# ========== JSON ==========
-keep class org.json.** { *; }

# ========== Netty (for gRPC server) ==========
-dontwarn io.netty.**
-keep class io.netty.** { *; }

# ========== javax.annotation ==========
-dontwarn javax.annotation.**
-keep class javax.annotation.** { *; }

# ========== General Android ==========
-keepattributes SourceFile,LineNumberTable
-keepattributes RuntimeVisibleAnnotations,RuntimeVisibleParameterAnnotations

# Keep our app classes
-keep class com.example.edge_mobile.** { *; }
