load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "factory.go",
        "generic.go",
    ],
    importpath = "github.com/kminehart/ladon-resource-manager/pkg/client/informers/externalversions",
    visibility = ["//visibility:public"],
    deps = [
        "//vendor/k8s.io/apimachinery/pkg/apis/meta/v1:go_default_library",
        "//vendor/k8s.io/apimachinery/pkg/runtime:go_default_library",
        "//vendor/k8s.io/apimachinery/pkg/runtime/schema:go_default_library",
        "//vendor/k8s.io/client-go/tools/cache:go_default_library",
        "//vendor/github.com/kminehart/ladon-resource-manager/pkg/apis/ladoncontroller/v1alpha1:go_default_library",
        "//vendor/github.com/kminehart/ladon-resource-manager/pkg/client/clientset/versioned:go_default_library",
        "//vendor/github.com/kminehart/ladon-resource-manager/pkg/client/informers/externalversions/internalinterfaces:go_default_library",
        "//vendor/github.com/kminehart/ladon-resource-manager/pkg/client/informers/externalversions/ladoncontroller:go_default_library",
    ],
)

filegroup(
    name = "package-srcs",
    srcs = glob(["**"]),
    tags = ["automanaged"],
    visibility = ["//visibility:private"],
)

filegroup(
    name = "all-srcs",
    srcs = [
        ":package-srcs",
        "//staging/src/github.com/kminehart/ladon-resource-manager/pkg/client/informers/externalversions/internalinterfaces:all-srcs",
        "//staging/src/github.com/kminehart/ladon-resource-manager/pkg/client/informers/externalversions/ladoncontroller:all-srcs",
    ],
    tags = ["automanaged"],
    visibility = ["//visibility:public"],
)
