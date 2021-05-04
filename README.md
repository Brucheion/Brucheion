[![DOI](https://zenodo.org/badge/135154939.svg)](https://zenodo.org/badge/latestdoi/135154939)

<img src="static/img/logo-flat.png" alt="" width="500">

# Brucheion, the Virtual Research Environment

Brucheion is a Virtual Research Environment (VRE) to create Linked Open Data (LOD) for historical languages and the research of historical objects. Brucheion runs on a backend server written in the [Go programming language](https://golang.org/) and benefits from Go's performance features and [multilingual text processing capabilities](https://blog.golang.org/strings). The VRE is still under active development and may contain bugs or security issues. If you encounter any issues, please don't hesitate to reach out to us!


## Setup

To install Brucheion simply obtain the latest release from [the Brucheion GitHub repository](https://github.com/Brucheion/Brucheion/releases). Pre-built binaries for Brucheion are provided for both Windows (32-bit and 64-bit) and macOS (64-bit). If your operating system isn't supported you will need to compile Brucheion on your machine after obtaining the Brucheion source code. Please conduct [the development section below](#development) on how to build Brucheion from source.

Brucheion expects to be situated next to configuration and content files such as [Deep Zoom Images (DZI)](https://openseadragon.github.io/examples/tilesource-dzi/) collections. If you need help for setting up your Brucheion instance, please reach out to us!

If you obtained a pre-built release of Brucheion, you will need to provide a configuration file. The `config.json` file of this repository offers sensible defaults and can be put into the same directory as the Brucheion binary. For configuring Brucheion more in-depth, please head to [the configuration chapter](docs/configuration.md).

After putting everything into place, your Brucheion directory should look like this:

```
/
  Brucheion*
  config.json
  image_archive/
    sample/
      ...
```


You can now run Brucheion in your terminal. Brucheion offers several options for specifying configuration files and enabling further development access. Running `./Brucheion --help` will state:

```
Usage of ./Brucheion:
  -config string
        Specify where to load the JSON config from. (default: from data directory)
  -localAssets
        Obtain static assets from the local filesystem during development. (default: false)
  -noauth
        Start Brucheion without authenticating with a provider (default: false)
  -update
        Check for updates and install them at startup. (default: false)
```

To learn about how to access Brucheion with accounts and login providers, please head to [the usage chapter](docs/usage.md).


## Development

For developing or building Brucheion, you will need the following software: [Go](https://golang.org/) (`>= 1.16`) and [Node.js](https://nodejs.org/) (`>= v12`).

For configuring the necessary parts of Brucheion, you will need to create a `providers.json` file that will provide the required keys and secrets to authentication providers. You can create your own provider configuration based off `providers.json.example`. Be advised that all providers are expected to be specified, not just a subset.

The Brucheion development workflow has been tested on macOS but should work on any Unix-based machine. By calling `make`, you can create a production build of Brucheion that will obtain all dependencies, run all tests, and build all sources.

Brucheion has two main software components:

1. The backend and legacy frontend, written in Go and using the [Go template engine](https://golang.org/pkg/html/template/). The frontend handles all data server-side and delivers the compiled HTML to the client.
2. The newer frontend is built with [Svelte](https://svelte.dev/) and renders pages client-side while obtaining data via the respective API endpoints of the Go backend. The code corresponding to this frontend is situated in the `app` folder.

Assets such as Go templates, stylesheets, and Svelte bundles are embedded into the Brucheion binary via [Go embeds](https://golang.org/pkg/embed/). From within the binary they are then served via HTTP.

Running `make` should furthermore sufficiently prepare your machine for development. In order to develop the Svelte-based UI, start a development process via `make dev`; it will process and bundle all JavaScript files as they are being changed. Then, in parallel, run the Brucheion binary via `./Brucheion -localAssets` and access Brucheion via `https://localhost:7000/` in your browser. After changing parts of the UI JavaScript code, simply refresh the website.

## License

Brucheion is licensed under the [MIT License](/LICENSE).

