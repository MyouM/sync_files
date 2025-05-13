test: 
	@cd ./internal/stream/test && go test . -v

bench: 
	@cd ./internal/stream/test && go test -bench=. -benchmem -benchtime=3s
run:
	@go run ./cmd/app/ $(filter-out $@,$(MAKECMDGOALS))

