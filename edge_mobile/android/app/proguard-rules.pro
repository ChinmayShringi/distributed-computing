# ProGuard/R8 rules for Edge Mobile

# ========== gRPC ==========
# Keep gRPC generated classes
-keep class io.grpc.** { *; }
-keep class com.google.protobuf.** { *; }
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
