# Go FaaS

> A lightweight Function-as-a-Service platform built with Go, providing isolated script execution with automatic container management.

[![pkg](https://pkg.go.dev/badge/github.com/pardnchiu/go-faas.svg)](https://pkg.go.dev/github.com/pardnchiu/go-faas)
[![version](https://img.shields.io/github/v/tag/pardnchiu/go-faas?label=release)](https://github.com/pardnchiu/go-faas/releases)
[![license](https://img.shields.io/github/license/pardnchiu/go-faas)](LICENSE)

## Features

* Support JavaScript, TypeScript, and Python scripts execution.
* Isolated runtime with Docker for protecting host system.
* Automatic detection of unhealthy containers and rebuild.

## Performance

- Cold Start: ~50ms (containers pre-warmed)
- Execution Time: Depends on script complexity
- Concurrent Requests: Up to 5 simultaneous executions

## License

This source code project is licensed under the [MIT](LICENSE) License.

## Author

<img src="https://avatars.githubusercontent.com/u/25631760" align="left" width="96" height="96" style="margin-right: 0.5rem;">

<h4 style="padding-top: 0">邱敬幃 Pardn Chiu</h4>

<a href="mailto:dev@pardn.io" target="_blank">
  <img src="https://pardn.io/image/email.svg" width="48" height="48">
</a> <a href="https://linkedin.com/in/pardnchiu" target="_blank">
  <img src="https://pardn.io/image/linkedin.svg" width="48" height="48">
</a>

***

©️ 2025 [邱敬幃 Pardn Chiu](https://pardn.io)
