load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "cli_lib",
    srcs = ["main.go"],
    importpath = "github.com/danielkhasanov/wordcube/cli",
    visibility = ["//visibility:public"],
    deps = ["//game:game_lib"],
)

go_binary(
    name = "cli",
    embed = [":cli_lib"],
    visibility = ["//visibility:public"],
)
