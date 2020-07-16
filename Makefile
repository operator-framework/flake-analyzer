GO := GOFLAGS="-mod=vendor" go

build: 
	$(GO) build -o ./bin/flake-analyzer ./cmd/

vendor:
	go mod vendor && go mod tidy

report-today: build
	./bin/flake-analyzer  $(if $(OWNER),-n $(OWNER)) $(if $(REPO),-r $(REPO)) $(if $(TOKEN),-t $(TOKEN))  $(if $(TEST_SUITE),-f $(TEST_SUITE)) $(if $(OUTPUT_FILE),-o $(OUTPUT_FILE)) --from 1 --to 0

report-last-7-days: build
	./bin/flake-analyzer  $(if $(OWNER),-n $(OWNER)) $(if $(REPO),-r $(REPO)) $(if $(TOKEN),-t $(TOKEN))  $(if $(TEST_SUITE),-f $(TEST_SUITE)) $(if $(OUTPUT_FILE),-o $(OUTPUT_FILE)) --from 7 --to 0

report-prev-7-days: build
	./bin/flake-analyzer  $(if $(OWNER),-n $(OWNER)) $(if $(REPO),-r $(REPO)) $(if $(TOKEN),-t $(TOKEN))  $(if $(TEST_SUITE),-f $(TEST_SUITE)) $(if $(OUTPUT_FILE),-o $(OUTPUT_FILE)) --from 14 --to 7

report-on-pr: build
	./bin/flake-analyzer  $(if $(OWNER),-n $(OWNER)) $(if $(REPO),-r $(REPO)) $(if $(TOKEN),-t $(TOKEN))  $(if $(TEST_SUITE),-f $(TEST_SUITE)) $(if $(PR),-p $(PR)) $(if $(OUTPUT_FILE),-o $(OUTPUT_FILE)) $(if $(COMMITS),-c $(COMMITS))
