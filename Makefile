.PHONY: test test-watch deploy synth clean

test:
	go test ./tests/... -v

test-unit:
	go test ./tests/stacks/... -v

test-integration:
	go test ./tests/integration/... -v

test-helpers:
	go test ./tests/helpers/... -v

test-coverage:
	go test ./tests/... -cover -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

test-race:
	go test ./tests/... -race -v

test-watch:
	find . -name "*.go" | entr -r go test ./tests/... -v

deploy:
	cdk deploy --all

synth:
	cdk synth --all

clean:
	rm -rf cdk.out
