FROM --platform=linux/amd64 ubuntu:18.04
# This prevents Docker from "loading metadata" on every docker build. Helps with
# poor internet connections
