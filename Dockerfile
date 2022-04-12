FROM golang:1.17 AS build
WORKDIR /src
COPY ["go.mod", "go.sum", "./"]
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -mod=readonly

FROM gcr.io/distroless/static:nonroot
LABEL org.opencontainers.image.source https://github.com/Robbilie/azure-ad-extended-userinfo
COPY --from=build /src/azure-ad-extended-userinfo /
ENTRYPOINT ["/azure-ad-extended-userinfo"]
