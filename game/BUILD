load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "game_lib",
    srcs = ["game.go"],
    importpath = "github.com/danielkhasanov/wordcube/game",
    visibility = ["//cli:__pkg__"],
)

go_test(
    name = "game_test",
    srcs = ["game_test.go"],
    visibility = ["//visibility:private"],
    deps = [":game_lib"],
)
