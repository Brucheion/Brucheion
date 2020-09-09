<img src="static/img/logo-flat.png" alt="" width="500">

# Brucheion, the Virtual Research Environment

Brucheion is a Virtual Research Environment (VRE) to create Linked Open Data (LOD) for historical languages and the research of historical objects. Brucheion runs on a backend server written in the [Go programming language](https://golang.org/) and benefits from Go's performance features and [multilingual text processing capabilities](https://blog.golang.org/strings). The VRE is still under active development and may contain bugs or security issues. If you encounter any issues, please don't hesitate to reach out to us!


## Setup

To install Brucheion simply obtain the latest release from [the Brucheion GitHub repository](https://github.com/Brucheion/Brucheion/releases). Pre-built binaries for Brucheion are provided for both Windows (32-bit and 64-bit) and macOS (64-bit). If your operating system isn't supported you will need to compile Brucheion on your machine after obtaining the Brucheion source code. Please conduct [the development section below](#development) on how to build Brucheion from source.

Brucheion expects to be situated next to configuration and content files such as the [CITE Exchange (CEX)](https://github.com/cite-architecture/citedx) collection and optional [Deep Zoom Images (DZI)](https://openseadragon.github.io/examples/tilesource-dzi/). If you need help for setting up your Brucheion instance, please reach out to us!

```
/
  Brucheion*
  cex/
    sample.cex
  config.json
  image_archive/
    sample/
      ...
```

For configuring Brucheion, please head to [the configuration chapter](docs/configuration.md).

With everything in place, you can now run Brucheion in your terminal. Brucheion offers several options for specifying configuration files and enabling further development access. Running `./Brucheion --help` will state:

```
Usage of ./Brucheion:
  -config string
        Specify where to load the JSON config from. (defalult: ./config.json (default "./config.json")
  -localAssets
        Obtain static assets from the local filesystem during development. (default: false)
  -noauth
        Start Brucheion without authenticating with a provider (default: false)
```

To learn about how to access Brucheion with accounts and login providers, please head to [the usage chapter](docs/usage.md).


## Development

For developing or building Brucheion, you will need the following software: [Go](https://golang.org/) (`>= 1.14`) and [Node.js](https://nodejs.org/) (`>= v12`). Furthermore, you will need to obtain a fork of the [`pkger`](https://github.com/markbates/pkger) tool:

```bash
git clone https://github.com/falafeljan/pkger.git
cd pkger/cmd/pkger
go install
```

The Brucheion development workflow has been tested on macOS but should work on any Unix-based machine. By calling `make`, you can create a production build of Brucheion that will obtain all dependencies, run all tests, and build all sources.

Running `make` should sufficiently prepare your machine for development. More recent components of the Brucheion UI are build as an interactive JavaScript application using the [Svelte](https://svelte.dev/) JavaScript framework. Parts related to this UI are situated in the `ui` folder.

In order to develop the Svelte-based UI, start a development process via `make dev-ui`; it will process and bundle all JavaScript files as they are being changed. Then, in parallel, run the Brucheion binary via `./Brucheion -localAssets` and access Brucheion via `https://localhost:7000/` in your browser. After changing parts of the UI JavaScript code, simply refresh the website.

## License

Brucheion is licensed under the [MIT License](/LICENSE).