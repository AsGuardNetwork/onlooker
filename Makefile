all: build

BUILD_FLAGS := -trimpath -ldflags "-s -w"

install:
	@echo "Installing onlooker"
	@go install -mod readonly $(BUILD_FLAGS) main.go

build:
	@go build --mod readonly $(BUILD_FLAGS) -o onlooker main.go

clean:
	rm -rf onlooker main vendor || true
	docker compose down -v -t0 || true
	kubectl delete cm onlooker-config || true
	kubectl delete deploy onlooker || true

docker:
	@docker compose up

k8s:
	@kubectl create configmap onlooker-config --from-file=onlooker.yaml
	@kubectl apply -f k8s-deploy.yaml

run:
	@go run main.go

.PHONY: all install build clean docker run
