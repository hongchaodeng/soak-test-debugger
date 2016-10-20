FROM alpine:latest

ADD _output/bin/soak-test-debugger /usr/local/bin
CMD ["/bin/sh", "-c", "/usr/local/bin/soak-test-debugger"]
