all: clear main

ifeq ($(OS), Windows_NT)
main: scrabble.exe
	./scrabble.exe

scrabble.exe:
	go build

clear:
	if exist "scrabble.exe" del scrabble.exe 
	if exist "scrabble" del scrabble
else
main: scrabble
	./scrabble

scrabble:
	go build

clear:
	@if [ -e scrabble ]; then \
        rm scrabble; \
	elif [ -e scrabble.exe ]; then \
		rm scrabble.exe; \
    fi
endif