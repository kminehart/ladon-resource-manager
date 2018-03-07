load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["interface.go"],
    importpath = "github.com/kminehart/ladon-resource-manager/pkg/client/informers/externalversions/ladoncontroller",
    visibility = ["//visibility:public"],
    deps = [
        "//vendor/github.com/kminehart/ladon-resource-manager/pkg/client/informers/externalversions/internalinterfaces:go_default_library",
        "//vendor/github.com/kminehart/ladon-resource-manager/pkg/client/informers/externalversions/ladoncontroller/v1alpha1:go_default_library",
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
        "//staging/src/github.com/kminehart/ladon-resource-manager/pkg/client/informers/externalversions/ladoncontroller/v1alpha1:all-srcs",
    ],
    tags = ["automanaged"],
    visibility = ["//visibility:public"],
)
