# gosmopolitan

![GitHub Workflow Status (main branch)](https://img.shields.io/github/actions/workflow/status/xen0n/gosmopolitan/go.yml?branch=main)
![Codecov](https://img.shields.io/codecov/c/gh/xen0n/gosmopolitan)
![GitHub license info](https://img.shields.io/github/license/xen0n/gosmopolitan)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/xen0n/gosmopolitan)
[![Go Report Card](https://goreportcard.com/badge/github.com/xen0n/gosmopolitan)](https://goreportcard.com/report/github.com/xen0n/gosmopolitan)
[![Go Reference](https://pkg.go.dev/badge/github.com/xen0n/gosmopolitan.svg)](https://pkg.go.dev/github.com/xen0n/gosmopolitan)

[English](./README.md)

用 `gosmopolitan` 检查你的 Go 代码库里有没有国际化（“i18n“）或者本地化（”l10n“）的阻碍。

项目名字来自“cosmopolitan”的文字游戏。

## 检查

`gosmopolitan` 目前会检查以下的反模式（anti-patterns）：

*   含有来自特定书写系统字符的字符串字面量（string literals）。

    项目中存在这种字符串，通常意味着相关的逻辑不便于国际化，或者至少在国际化/本地化适配过程中会涉及特殊对待。

*   `time.Local` 的使用。

    支持国际化的应用或程序库，几乎永远不应以程序当前运行环境的时区来处理时间、日期数据。
    相反，在这种场景下，开发者应该使用相应的用户偏好，或者按照领域逻辑确定应该使用的时区。

注意：除了直接向 `time.Local` 转换之外，还有很多其他写法会产生本地时区的时刻，例如：

* `time.LoadLocation("Local")`
* 从 `time.Ticker` 收到的值
* 文档中明确了会返回本地时刻的函数
    * `time.Now()`
    * `time.Unix()`
    * `time.UnixMilli()`
    * `time.UnixMicro()`

为了正确识别这些使用场景，需要有一个相当完善的数据流分析 pass，目前还没实现。
此外，当前您还需要自行密切注意从外部传入的时刻值（例如从您使用的 Gin 或 gRPC
之类框架传来的那些），因为这些值当前也没有被正确跟踪。

## 注意事项

请注意，本库中实现的检查仅适用于具有以下性质的代码库，因此可能不适用于您的具体场景：

* 项目原先是为使用非拉丁字母书写系统的受众群体开发的，
* 项目会返回包含这些非拉丁字母字符的裸的字符串（即，未经处理或变换的），
* 项目可能偶尔（或者经常）引用程序当前运行环境的本地时区。

举个例子：如果您在翻新一个本来面向中国用户群体（因此到处都在产生含有汉字的字符串）的
web 服务，以使其更加国际化，这里的 lints 可能会很有价值。
反之，如果您想在一个仅支持英语的应用里，寻找其中不利于国际化的那部分写法，本
linter 则什么都不会输出。

## 与 golangci-lint 集成

`gosmopolitan` 目前没有集成进上游 [`golangci-lint`][gcl-home]，但您仍然可以[以自定义插件的方式][gcl-plugin]使用本项目。

[gcl-home]: https://golangci-lint.run
[gcl-plugin]: https://golangci-lint.run/contributing/new-linters/#how-to-add-a-private-linter-to-golangci-lint

首先像这样做一个插件 `.so` 文件：

```go
// 用类似 `go build -buildmode=plugin` 的方式编译

package main

import (
	"github.com/xen0n/gosmopolitan"
	"golang.org/x/tools/go/analysis"
)

type analyzerPlugin struct{}

func (analyzerPlugin) GetAnalyzers() []*analysis.Analyzer {
	// 你可以用 gosmopolitan.NewAnalyzer 来自定义配置。
	return []*analysis.Analyzer{
		gosmopolitan.DefaultAnalyzer,
	}
}

var AnalyzerPlugin analyzerPlugin
```

您只需要保证构建时使用的 `golang.org/x/tools` 模块版本和您的 `golangci-lint`
二进制的相应模块版本一致。（当然，`golangci-lint` 二进制也应该包含插件支持；
[Homebrew 的 `golangci-lint` 没有插件支持][hb-issue]，尤其需要注意。）

[hb-issue]: https://github.com/golangci/golangci-lint/issues/1182

|`golangci-lint` 版本|对应可用的 `gosmopolitan` tag|
|--------------------|-----------------------------|
|1.50.x|v1.0.0|

然后在您的 `.golangci.yml` 中引用它，在 `linters` 一节中启用它：

```yaml
linters:
  # ...
  enable:
    # ...
    - gosmopolitan
    # ...

linters-settings:
  custom:
    gosmopolitan:
      path: 'path/to/your/plugin.so'
      description: 'Report certain i18n/l10n anti-patterns in your Go codebase'
      original-url: 'https://github.com/xen0n/gosmopolitan'
  # ...
```

这样您就可以像使用其他 linters 一样 `golangci-lint run` 和
`//nolint:gosmopolitan` 了。

## 许可证

`gosmopolitan` 以 GPL v3 或更新的版本许可使用。
