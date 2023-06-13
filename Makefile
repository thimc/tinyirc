NAME = tinyirc

PREFIX = /usr/local

${NAME}:
	go build -o bin/${NAME} main.go

clean:
	rm -f bin/${NAME}

install: ${NAME}
	cp -f bin/${NAME} "${DESTDIR}${PREFIX}/bin"
	chmod 755 "${DESTDIR}${PREFIX}/bin/${BIN}"

uninstall:
		rm -f "${DESTDIR}${PREFIX}/bin/${NAME}"

.PHONY: clean install uninstall
