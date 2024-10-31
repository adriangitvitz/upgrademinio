binary:
 	@CGO_ENABLED=0 GOOS=linux go build -o upgrademinio .

docker: binary
	@docker buildx build --no-cache --load --platform linux/arm64 -t minioupgrade:local .
