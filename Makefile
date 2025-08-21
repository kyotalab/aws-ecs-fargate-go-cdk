.PHONY: test test-watch deploy synth clean

test:
  go test ./tests/... -v

test-watch:
  find . -name "*.go" | entr -r go test ./tests/... -v

deploy:
  cdk deploy --all

synth:
  cdk synth --all

clean:
  rm -rf cdk.out
