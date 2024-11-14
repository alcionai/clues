.PHONY: all
all:
	go test -bench . -benchmem

.PHONY: err
err:
	go test -bench BenchmarkErr -benchmem

.PHONY: clues
clues:
	go test -bench BenchmarkClues -benchmem

.PHONY: prof
prof:
	go test -bench BenchmarkProf -benchmem
