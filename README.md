# Jai Docker
Small, incomplete project to try to get early Jai support on unsupported platforms
like MacOS and ARM devices. Kinda buggy on ARM, but seems to work on x87 MacOS.

## How to use
You'll need to [install Golang](https://golang.org/doc/install) to create a Jai
shim and add it to your path, and [install Docker](https://docs.docker.com/get-docker/)
to virtualize the actual compiler. You'll want to install both before starting.

1. Clone this repository. You can download the folder using Github or clone using
   this command:

   ```
   git clone https://github.com/A1Liu/jai-docker.git
   ```

2. Install the Jai shim `cmd/jai.go`. You can do this by running this command at
   the root of this project:

   ```
   go install cmd/jai.go
   ```

3. Copy the latest Beta's folder to this project, as `jai`. *This is the only step
   you need to repeat* when a new Beta is released.

4. When you want to run the compiler, just write `jai` in your terminal!
