NAME = tinyirc

PREFIX = /usr/local

${NAME}:
	go build -o ${NAME} main.go

clean:
	rm -f ${NAME}

install: ${NAME}
	cp -f ${NAME} "${DESTDIR}${PREFIX}/bin"
	chmod 755 "${DESTDIR}${PREFIX}/bin/${BIN}"

uninstall:
	rm -f "${DESTDIR}${PREFIX}/bin/${NAME}"

.PHONY: clean install uninstall
