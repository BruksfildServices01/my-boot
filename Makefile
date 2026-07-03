build:
	go build -o bin/myboot ./cmd/server

run:
	go run ./cmd/server

migrate:
	@export $$(grep -v '^\#' .env | xargs) && \
	for f in $$(ls migrations/*.sql | sort); do \
		echo "Aplicando $$f..."; \
		psql "$$DATABASE_URL" -f $$f; \
	done

tidy:
	go mod tidy

# Para o VPS Hostinger: compila para Linux e envia via scp
deploy:
	GOOS=linux GOARCH=amd64 go build -o bin/myboot-linux ./cmd/server
	@echo "Binário gerado em bin/myboot-linux"
	@echo "Envie para o VPS: scp bin/myboot-linux usuario@ip:/caminho/myboot"

.PHONY: build run migrate tidy deploy
