load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "solver_lib",
    srcs = ["solver.go"],
    importpath = "github.com/dkhasanov/games/wordcube",
    visibility = ["//visibility:private"],
)

go_library(
    name = "game",
    srcs = ["game.go"],
    importpath = "github.com/dkhasanov/games/wordcube/game",
    visibility = ["//visibility:private"],
)

go_test(
    name = "game_test",
    srcs = ["game_test.go"],
    visibility = ["//visibility:private"],
    deps = [":game"],
)
