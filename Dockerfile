FROM registry.access.redhat.com/ubi9/ubi-minimal:latest AS build

 RUN microdnf update -y --nodocs && microdnf install ca-certificates findutils -y --nodocs

 FROM registry.access.redhat.com/ubi9/ubi-micro:latest

 ARG TAG

 COPY --from=build /etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem /etc/pki/ca-trust/extracted/pem/
 COPY --from=build /usr/bin/find /usr/bin/find
 COPY --from=build /usr/bin/cat /usr/bin/cat

 RUN mkdir -p tmp/webhook && chmod -R 777 tmp/webhook

 COPY upgrademinio /upgrademinio

 ENTRYPOINT ["/upgrademinio"]
