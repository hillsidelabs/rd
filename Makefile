SRC = $(shell find . -name '*.go')

rd: $(SRC)
	go mod tidy
	go build .
	go install github.com/hillsidelabs/rd

rd-server: $(SRC)
	go mod tidy
	templ fmt .
	go build ./cli/rd-server


# Run templ generation in watch mode to detect all .templ files and
# re-create _templ.txt files on change, then send reload event to browser.
# Default url: http://localhost:7331
templ:
	templ generate --watch --proxy="http://localhost:8090" --open-browser=false -v

# Run air to detect any go file changes to re-build and re-run the server.
server:
	air \
	--build.cmd "make rd-server" \
	--build.bin "./rd-server" \
	--build.delay "100" \
	--build.exclude_dir "node_modules" \
	--build.include_ext "go" \
	--build.stop_on_error "false" \
	--misc.clean_on_exit true

tailwind-clean:
	tailwindcss -i ./web/assets/css/input.css -o ./web/assets/css/output.css --clean

# Run tailwindcss to generate the styles.css bundle in watch mode.
tailwind-watch:
	tailwindcss -i ./web/assets/css/input.css -o ./web/assets/css/output.css --watch

# Start development server
dev:
	make tailwind-clean
	make -j3 templ server tailwind-watch

bootstrap:
	go install github.com/air-verse/air@latest
	brew install tailwindcss
