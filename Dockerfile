# Dockerfile References: https://docs.docker.com/engine/reference/builder/

# Start from the latest golang base image
FROM golang:alpine AS builder

# Add Maintainer Info
LABEL maintainer=""

RUN apk --no-cache add build-base git
WORKDIR /root/
ADD . /root
RUN cd /root/src && go build -o pubsub-fn

######## Start a new stage from scratch #######
FROM alpine

# RUN apk update
WORKDIR /root/bin
RUN mkdir /root/config/

# Copy the Pre-built binary file and default configuraitons from the previous stage
COPY --from=builder /root/src/pubsub-fn /root/bin
COPY --from=builder /root/src/unit-test/example_p* /root/config/

# Command to run the executable
ENTRYPOINT ["./pubsub-fn"]
